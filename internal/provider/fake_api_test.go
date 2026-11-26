package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const testToken = "test-token"

// fakeApp is the state the fake API keeps for a registered app.
type fakeApp struct {
	Slug      string `json:"slug"`
	RepoURL   string `json:"repo_url"`
	IsPublic  bool   `json:"is_public"`
	RepoSlug  string `json:"repo_slug"`
	Provider  string `json:"provider"`
	RepoOwner string `json:"repo_owner"`
	Title     string `json:"title"`
}

// fakeResponse is a canned reply that overrides an endpoint.
type fakeResponse struct {
	status      int
	contentType string
	body        string
}

// fakeRequest is a request received by the fake API.
type fakeRequest struct {
	Method string
	Path   string
	Body   string
	Header http.Header
}

// fakeAPI is an in-memory Bitrise API serving the endpoints the provider calls.
type fakeAPI struct {
	server *httptest.Server

	mu         sync.Mutex
	nextSlug   int
	apps       map[string]*fakeApp
	finished   map[string]PayloadFinish
	sshKeys    map[string]Payload
	ymls       map[string]string
	roles      map[string][]string
	secrets    map[string]*SecretResponse
	orgSecrets map[string]*orgSecretResponse
	groups     map[string][]GroupAPIModel
	stacks     []string
	overrides  map[string]fakeResponse
	requests   []fakeRequest

	// omitOrgSecretUpdateBody applies org secret updates but answers without a body.
	omitOrgSecretUpdateBody bool
}

// newFakeAPI starts a fake API that is shut down when the test ends.
func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	api := &fakeAPI{
		nextSlug:   123,
		apps:       map[string]*fakeApp{},
		finished:   map[string]PayloadFinish{},
		sshKeys:    map[string]Payload{},
		ymls:       map[string]string{},
		roles:      map[string][]string{},
		secrets:    map[string]*SecretResponse{},
		orgSecrets: map[string]*orgSecretResponse{},
		groups:     map[string][]GroupAPIModel{},
		overrides:  map[string]fakeResponse{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v0.1/apps/register", api.registerApp)
	mux.HandleFunc("GET /v0.1/apps/{app}", api.withApp(api.getApp))
	mux.HandleFunc("PATCH /v0.1/apps/{app}", api.withApp(api.updateApp))
	mux.HandleFunc("DELETE /v0.1/apps/{app}", api.withApp(api.deleteApp))
	mux.HandleFunc("POST /v0.1/apps/{app}/finish", api.withApp(api.finishApp))
	mux.HandleFunc("POST /v0.1/apps/{app}/register-ssh-key", api.withApp(api.registerSSHKey))
	mux.HandleFunc("GET /v0.1/apps/{app}/bitrise.yml", api.withApp(api.getYML))
	mux.HandleFunc("POST /v0.1/apps/{app}/bitrise.yml", api.withApp(api.uploadYML))
	mux.HandleFunc("GET /v0.1/apps/{app}/roles/{role}", api.withApp(api.getRole))
	mux.HandleFunc("PUT /v0.1/apps/{app}/roles/{role}", api.withApp(api.putRole))
	mux.HandleFunc("POST /v0.1/apps/{app}/secrets", api.createSecret)
	mux.HandleFunc("GET /v0.1/apps/{app}/secrets/{name}", api.getSecret)
	mux.HandleFunc("PATCH /v0.1/apps/{app}/secrets/{name}", api.updateSecret)
	mux.HandleFunc("DELETE /v0.1/apps/{app}/secrets/{name}", api.deleteSecret)
	mux.HandleFunc("GET /v0.1/organizations/{org}/groups", api.listGroups)
	mux.HandleFunc("POST /v0.1/organizations/{org}/secrets", api.createOrgSecret)
	mux.HandleFunc("GET /v0.1/organizations/{org}/secrets/{name}", api.getOrgSecret)
	mux.HandleFunc("PATCH /v0.1/organizations/{org}/secrets/{name}", api.updateOrgSecret)
	mux.HandleFunc("DELETE /v0.1/organizations/{org}/secrets/{name}", api.deleteOrgSecret)
	mux.HandleFunc("GET /v0.1/available-stacks", api.listStacks)

	api.server = httptest.NewServer(api.intercept(mux))
	t.Cleanup(api.server.Close)
	return api
}

// providerConfig returns the provider block pointing at the fake API.
func (a *fakeAPI) providerConfig() string {
	return fmt.Sprintf(`
provider "bitrise" {
  endpoint = %q
  token    = %q
}
`, a.server.URL, testToken)
}

// override makes method and path answer with status and body until cleared.
func (a *fakeAPI) override(method, path string, status int, contentType, body string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.overrides[method+" "+path] = fakeResponse{status: status, contentType: contentType, body: body}
}

// fail makes method and path answer with a JSON error until cleared.
func (a *fakeAPI) fail(method, path string, status int) {
	a.override(method, path, status, "application/json", fmt.Sprintf(`{"message":"%s"}`, http.StatusText(status)))
}

func (a *fakeAPI) clearOverrides() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.overrides = map[string]fakeResponse{}
}

// addApp registers an app directly, without going through the API.
func (a *fakeAPI) addApp(slug string) {
	a.update(func(a *fakeAPI) {
		a.apps[slug] = &fakeApp{Slug: slug, RepoURL: "https://github.com/example/repo.git", Provider: "github", RepoOwner: "example", RepoSlug: "repo", Title: "repo"}
	})
}

// removeApp removes an app directly, as if deleted outside Terraform.
func (a *fakeAPI) removeApp(slug string) {
	a.update(func(a *fakeAPI) { delete(a.apps, slug) })
}

// update runs fn with exclusive access to the fake's state.
func (a *fakeAPI) update(fn func(a *fakeAPI)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fn(a)
}

// received returns the requests sent to method and path.
func (a *fakeAPI) received(method, path string) []fakeRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	var matched []fakeRequest
	for _, r := range a.requests {
		if r.Method == method && r.Path == path {
			matched = append(matched, r)
		}
	}
	return matched
}

// allRequests returns every request received so far.
func (a *fakeAPI) allRequests() []fakeRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]fakeRequest(nil), a.requests...)
}

func (a *fakeAPI) intercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))

		a.mu.Lock()
		a.requests = append(a.requests, fakeRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: string(body), Header: r.Header.Clone()})
		override, overridden := a.overrides[r.Method+" "+r.URL.EscapedPath()]
		a.mu.Unlock()

		if r.Header.Get("Authorization") != testToken {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
			return
		}
		if overridden {
			if override.contentType != "" {
				w.Header().Set("Content-Type", override.contentType)
			}
			w.WriteHeader(override.status)
			_, _ = io.WriteString(w, override.body)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withApp answers 404 unless the app in the path exists.
func (a *fakeAPI) withApp(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		_, ok := a.apps[r.PathValue("app")]
		a.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
			return
		}
		next(w, r)
	}
}

func (a *fakeAPI) registerApp(w http.ResponseWriter, r *http.Request) {
	var req appRegisterRequest
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	slug := fmt.Sprintf("app-%d", a.nextSlug)
	a.nextSlug++
	// Like Bitrise, name the app after the repository when no title is sent.
	title := req.Title
	if title == "" {
		title = req.GitRepoSlug
	}
	a.apps[slug] = &fakeApp{Slug: slug, RepoURL: req.RepoURL, IsPublic: req.IsPublic, RepoSlug: req.GitRepoSlug, Provider: req.Provider, RepoOwner: req.GitOwner, Title: title}
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, CreateResponse{Status: "ok", Slug: slug})
}

func (a *fakeAPI) getApp(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	app := *a.apps[r.PathValue("app")]
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"data": app})
}

func (a *fakeAPI) updateApp(w http.ResponseWriter, r *http.Request) {
	var req appUpdateRequest
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	app := a.apps[r.PathValue("app")]
	app.IsPublic = req.IsPublic
	if req.RepositoryURL != "" {
		app.RepoURL = req.RepositoryURL
	}
	if req.Title != "" {
		app.Title = req.Title
	}
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *fakeAPI) deleteApp(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	delete(a.apps, r.PathValue("app"))
	a.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (a *fakeAPI) finishApp(w http.ResponseWriter, r *http.Request) {
	var req PayloadFinish
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	a.finished[r.PathValue("app")] = req
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *fakeAPI) registerSSHKey(w http.ResponseWriter, r *http.Request) {
	var req Payload
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	a.sshKeys[r.PathValue("app")] = req
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *fakeAPI) getYML(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	yml, ok := a.ymls[r.PathValue("app")]
	a.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	_, _ = io.WriteString(w, yml)
}

func (a *fakeAPI) uploadYML(w http.ResponseWriter, r *http.Request) {
	var req BitriseYmlRequest
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	a.ymls[r.PathValue("app")] = req.AppConfigDatastoreYaml
	a.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (a *fakeAPI) getRole(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	groups := append([]string{}, a.roles[r.PathValue("app")+"/"+r.PathValue("role")]...)
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, GroupRolesReadResponse{Groups: groups})
}

func (a *fakeAPI) putRole(w http.ResponseWriter, r *http.Request) {
	var req GroupRolesUpdateRequest
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	a.roles[r.PathValue("app")+"/"+r.PathValue("role")] = req.Groups
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *fakeAPI) createSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name                     string `json:"name"`
		Value                    string `json:"value"`
		IsProtected              *bool  `json:"is_protected"`
		IsExposedForPullRequests *bool  `json:"is_exposed_for_pull_requests"`
		ExpandInStepInputs       *bool  `json:"expand_in_step_inputs"`
	}
	if !decode(w, r, &req) {
		return
	}
	key := r.PathValue("app") + "/" + req.Name
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.secrets[key]; exists {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "secret already exists"})
		return
	}
	// Like the API, an omitted expand_in_step_inputs defaults to true.
	secret := &SecretResponse{ID: key, Name: req.Name, Value: req.Value, ExpandInStepInputs: true}
	setIfPresent(&secret.IsProtected, req.IsProtected)
	setIfPresent(&secret.IsExposedForPullRequests, req.IsExposedForPullRequests)
	setIfPresent(&secret.ExpandInStepInputs, req.ExpandInStepInputs)
	a.secrets[key] = secret
	writeJSON(w, http.StatusCreated, secret.redacted())
}

func (a *fakeAPI) getSecret(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	secret, ok := a.secrets[r.PathValue("app")+"/"+r.PathValue("name")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	writeJSON(w, http.StatusOK, secret.redacted())
}

func (a *fakeAPI) updateSecret(w http.ResponseWriter, r *http.Request) {
	var req SecretUpdateRequest
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	secret, ok := a.secrets[r.PathValue("app")+"/"+r.PathValue("name")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	if req.Value != "" {
		secret.Value = req.Value
	}
	setIfPresent(&secret.IsProtected, req.IsProtected)
	setIfPresent(&secret.IsExposedForPullRequests, req.IsExposedForPullRequests)
	setIfPresent(&secret.ExpandInStepInputs, req.ExpandInStepInputs)
	writeJSON(w, http.StatusOK, secret.redacted())
}

func (a *fakeAPI) deleteSecret(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("app") + "/" + r.PathValue("name")
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.secrets[key]; !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	delete(a.secrets, key)
	w.WriteHeader(http.StatusNoContent)
}

func (a *fakeAPI) listGroups(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	groups := append([]GroupAPIModel{}, a.groups[r.PathValue("org")]...)
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, groups)
}

func (a *fakeAPI) createOrgSecret(w http.ResponseWriter, r *http.Request) {
	var req orgSecretCreateRequest
	if !decode(w, r, &req) {
		return
	}
	key := r.PathValue("org") + "/" + req.Name
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.orgSecrets[key]; exists {
		writeJSON(w, http.StatusConflict, map[string]string{"message": "secret already exists"})
		return
	}
	secret := &orgSecretResponse{
		ID: key, Name: req.Name, Value: &req.Value, Scope: "workspace",
		IsProtected: req.IsProtected, IsExposedForPullRequests: req.IsExposedForPullRequests, ExpandInStepInputs: req.ExpandInStepInputs,
	}
	a.orgSecrets[key] = secret
	writeJSON(w, http.StatusCreated, secret.redacted())
}

func (a *fakeAPI) getOrgSecret(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	secret, ok := a.orgSecrets[r.PathValue("org")+"/"+r.PathValue("name")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	writeJSON(w, http.StatusOK, secret.redacted())
}

func (a *fakeAPI) updateOrgSecret(w http.ResponseWriter, r *http.Request) {
	var req orgSecretUpdateRequest
	if !decode(w, r, &req) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	secret, ok := a.orgSecrets[r.PathValue("org")+"/"+r.PathValue("name")]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	secret.Value = &req.Value
	secret.IsProtected = req.IsProtected
	secret.IsExposedForPullRequests = req.IsExposedForPullRequests
	secret.ExpandInStepInputs = req.ExpandInStepInputs
	if a.omitOrgSecretUpdateBody {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, http.StatusOK, secret.redacted())
}

func (a *fakeAPI) deleteOrgSecret(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("org") + "/" + r.PathValue("name")
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.orgSecrets[key]; !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
		return
	}
	delete(a.orgSecrets, key)
	w.WriteHeader(http.StatusNoContent)
}

func (a *fakeAPI) listStacks(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	stacks := make(map[string]map[string]string, len(a.stacks))
	for _, key := range a.stacks {
		stacks[key] = map[string]string{"title": key}
	}
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, stacks)
}

// redacted returns the secret as the API reports it: without the value when protected.
func (s *SecretResponse) redacted() SecretResponse {
	out := *s
	if out.IsProtected {
		out.Value = ""
	}
	return out
}

func (s *orgSecretResponse) redacted() orgSecretResponse {
	out := *s
	if out.IsProtected {
		out.Value = nil
	}
	return out
}

func setIfPresent(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// oversizedJSON returns a valid JSON document larger than the client accepts.
func oversizedJSON() string {
	return `{"padding":"` + strings.Repeat("a", 1<<20) + `"}`
}

func stringPtr(s string) *string { return &s }
