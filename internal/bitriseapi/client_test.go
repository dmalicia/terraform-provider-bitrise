package bitriseapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type appSummary struct {
	Slug string `json:"slug"`
}

// newTestServer starts an API server whose handler answers every request.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// respond returns a handler that replies with status, content type and body.
func respond(status int, contentType, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

type recordedRequest struct {
	method, uri, body string
	header            http.Header
}

// newRecordingServer answers 200 and hands each request over the returned channel.
func newRecordingServer(t *testing.T) (*httptest.Server, <-chan recordedRequest) {
	t.Helper()
	received := make(chan recordedRequest, 1)
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- recordedRequest{method: r.Method, uri: r.URL.RequestURI(), body: string(body), header: r.Header.Clone()}
		w.WriteHeader(http.StatusOK)
	})
	return server, received
}

func getApp(appSlug string) Request {
	return Request{Op: "read app " + appSlug, Method: http.MethodGet, Path: AppPath(appSlug)}
}

func TestClient_Do_whenServerReturns200_decodesJSONBody(t *testing.T) {
	server := newTestServer(t, respond(http.StatusOK, "application/json", `{"slug":"app-123"}`))
	client := New(server.URL, "test-token")
	var out appSummary

	err := client.Do(t.Context(), getApp("app-123"), &out)

	require.NoError(t, err)
	assert.Equal(t, "app-123", out.Slug)
}

func TestClient_Do_whenStatusIsAccepted_decodesJSONBody(t *testing.T) {
	server := newTestServer(t, respond(http.StatusCreated, "application/json", `{"slug":"app-123"}`))
	client := New(server.URL, "test-token")
	req := Request{Op: "register app", Method: http.MethodPost, Path: RegisterAppPath, Accept: []int{http.StatusOK, http.StatusCreated}}
	var out appSummary

	err := client.Do(t.Context(), req, &out)

	require.NoError(t, err)
	assert.Equal(t, "app-123", out.Slug)
}

func TestClient_Do_withNilOut_whenServerReturns204_succeeds(t *testing.T) {
	server := newTestServer(t, respond(http.StatusNoContent, "", ""))
	client := New(server.URL, "test-token")
	req := Request{Op: "delete secret", Method: http.MethodDelete, Path: AppSecretPath("app-123", "MY_SECRET"), Accept: []int{http.StatusNoContent}}

	err := client.Do(t.Context(), req, nil)

	assert.NoError(t, err)
}

func TestClient_Do_withNilOut_whenBodyIsNotJSON_succeeds(t *testing.T) {
	server := newTestServer(t, respond(http.StatusOK, "text/html", "<html>ok</html>"))
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), nil)

	assert.NoError(t, err)
}

func TestClient_Do_withRawResponse_returnsBodyAndContentType(t *testing.T) {
	server := newTestServer(t, respond(http.StatusOK, "text/yaml", "format_version: 11\n"))
	client := New(server.URL, "test-token")
	var raw RawResponse

	err := client.Do(t.Context(), getApp("app-123"), &raw)

	require.NoError(t, err)
	assert.Equal(t, "text/yaml", raw.ContentType)
	assert.Equal(t, "format_version: 11\n", string(raw.Body))
}

func TestClient_Do_whenServerReturns404_returnsErrNotFound(t *testing.T) {
	server := newTestServer(t, respond(http.StatusNotFound, "application/json", `{"message":"Not Found"}`))
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), &appSummary{})

	assert.ErrorIs(t, err, ErrNotFound)
}

func TestClient_Do_whenStatusIsRejected_returnsStatusError(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"201 without accept":    {status: http.StatusCreated, body: `{"slug":"app-123"}`},
		"400 bad request":       {status: http.StatusBadRequest, body: `{"message":"bad"}`},
		"401 unauthorized":      {status: http.StatusUnauthorized, body: `{"message":"Unauthorized"}`},
		"422 plain text":        {status: http.StatusUnprocessableEntity, body: "unprocessable entity"},
		"500 internal error":    {status: http.StatusInternalServerError, body: `{"message":"internal error"}`},
		"503 empty body":        {status: http.StatusServiceUnavailable, body: ""},
		"404 not found as well": {status: http.StatusNotFound, body: "missing"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			server := newTestServer(t, respond(tc.status, "", tc.body))
			client := New(server.URL, "test-token")

			err := client.Do(t.Context(), getApp("app-123"), &appSummary{})

			var statusErr *StatusError
			require.ErrorAs(t, err, &statusErr)
			assert.Equal(t, "read app app-123", statusErr.Op)
			assert.Equal(t, tc.status, statusErr.Code)
			assert.Equal(t, tc.body, statusErr.Body)
			assert.Equal(t, fmt.Sprintf("%d %s", tc.status, http.StatusText(tc.status)), statusErr.Status())
		})
	}
}

func TestClient_Do_whenBodyIsMalformedJSON_returnsDecodeError(t *testing.T) {
	server := newTestServer(t, respond(http.StatusOK, "application/json", `{"slug":`))
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), &appSummary{})

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageDecode, callErr.Stage)
}

func TestClient_Do_whenBodyIsEmpty_returnsDecodeError(t *testing.T) {
	server := newTestServer(t, respond(http.StatusOK, "application/json", ""))
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), &appSummary{})

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageDecode, callErr.Stage)
}

func TestClient_Do_whenBodyHasWrongType_returnsDecodeError(t *testing.T) {
	server := newTestServer(t, respond(http.StatusOK, "application/json", `{"slug":123}`))
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), &appSummary{})

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageDecode, callErr.Stage)
}

func TestClient_Do_whenBodyExceedsLimit_returnsErrBodyTooLarge(t *testing.T) {
	server := newTestServer(t, respond(http.StatusOK, "application/json", `"`+strings.Repeat("x", maxBodyBytes)+`"`))
	client := New(server.URL, "test-token")
	var out string

	err := client.Do(t.Context(), getApp("app-123"), &out)

	require.ErrorIs(t, err, ErrBodyTooLarge)
	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageRead, callErr.Stage)
}

func TestClient_Do_withRawResponse_whenBodyIsAtLimit_returnsWholeBody(t *testing.T) {
	body := strings.Repeat("x", maxBodyBytes)
	server := newTestServer(t, respond(http.StatusOK, "text/plain", body))
	client := New(server.URL, "test-token")
	var raw RawResponse

	err := client.Do(t.Context(), getApp("app-123"), &raw)

	require.NoError(t, err)
	assert.Len(t, raw.Body, maxBodyBytes)
}

func TestClient_Do_whenRejectedBodyExceedsLimit_truncatesStatusErrorBody(t *testing.T) {
	server := newTestServer(t, respond(http.StatusBadGateway, "text/html", strings.Repeat("e", 3*maxErrorBodyBytes)))
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), nil)

	var statusErr *StatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, strings.Repeat("e", maxErrorBodyBytes)+" [truncated]", statusErr.Body)
}

func TestClient_Do_whenResponseIsTruncated_returnsReadError(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"slug":`)
	})
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), &appSummary{})

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageRead, callErr.Stage)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestClient_Do_whenRejectedResponseIsTruncated_returnsReadErrorMatchingStatus(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":`)
	})
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), &appSummary{})

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageRead, callErr.Stage)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestClient_Do_sendsTokenAndJSONContentType(t *testing.T) {
	server, received := newRecordingServer(t)
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), nil)

	require.NoError(t, err)
	got := <-received
	assert.Equal(t, "test-token", got.header.Get("Authorization"))
	assert.Equal(t, "application/json", got.header.Get("Content-Type"))
}

func TestClient_Do_withBody_sendsJSONToMethodAndPath(t *testing.T) {
	server, received := newRecordingServer(t)
	client := New(server.URL, "test-token")
	req := Request{Op: "update app", Method: http.MethodPatch, Path: AppPath("app-123"), Body: map[string]bool{"is_public": false}}

	err := client.Do(t.Context(), req, nil)

	require.NoError(t, err)
	got := <-received
	assert.Equal(t, http.MethodPatch, got.method)
	assert.Equal(t, "/v0.1/apps/app-123", got.uri)
	assert.JSONEq(t, `{"is_public":false}`, got.body)
}

func TestClient_Do_withoutBody_sendsEmptyBody(t *testing.T) {
	server, received := newRecordingServer(t)
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), nil)

	require.NoError(t, err)
	assert.Empty(t, (<-received).body)
}

func TestClient_Do_whenBodyCannotBeEncoded_returnsEncodeErrorWithoutSending(t *testing.T) {
	server, received := newRecordingServer(t)
	client := New(server.URL, "test-token")
	req := Request{Op: "update app", Method: http.MethodPatch, Path: AppPath("app-123"), Body: make(chan int)}

	err := client.Do(t.Context(), req, nil)

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageEncode, callErr.Stage)
	assert.Empty(t, received)
}

func TestClient_Do_whenBaseURLIsInvalid_returnsBuildError(t *testing.T) {
	client := New("http://[::1", "test-token")

	err := client.Do(t.Context(), getApp("app-123"), nil)

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageBuild, callErr.Stage)
}

func TestClient_Do_whenServerIsUnreachable_returnsSendError(t *testing.T) {
	server := httptest.NewServer(respond(http.StatusOK, "", ""))
	server.Close()
	client := New(server.URL, "test-token")

	err := client.Do(t.Context(), getApp("app-123"), nil)

	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageSend, callErr.Stage)
}

func TestClient_Do_whenContextIsCanceled_returnsContextCanceled(t *testing.T) {
	received := make(chan struct{})
	server := newTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
		close(received)
		<-r.Context().Done()
	})
	client := New(server.URL, "test-token")
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-received
		cancel()
	}()

	err := client.Do(ctx, getApp("app-123"), nil)

	require.ErrorIs(t, err, context.Canceled)
	var callErr *CallError
	require.ErrorAs(t, err, &callErr)
	assert.Equal(t, StageSend, callErr.Stage)
}

func TestClient_Do_whenServerDoesNotAnswerInTime_returnsTimeoutError(t *testing.T) {
	server := newTestServer(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	client := New(server.URL, "test-token")
	client.httpClient.Timeout = 50 * time.Millisecond

	err := client.Do(t.Context(), getApp("app-123"), nil)

	var urlErr *url.Error
	require.ErrorAs(t, err, &urlErr)
	assert.True(t, urlErr.Timeout())
}

func TestClient_Do_whenTransportFails_returnsSendErrorWrappingCause(t *testing.T) {
	cause := errors.New("dial refused")
	client := New("http://api.example.test", "test-token")
	client.httpClient.Transport = &authTransport{token: "test-token", base: failingTransport{err: cause}}

	err := client.Do(t.Context(), getApp("app-123"), nil)

	assert.ErrorIs(t, err, cause)
}

func TestAuthTransport_RoundTrip_doesNotModifyCallerRequest(t *testing.T) {
	var sent *http.Request
	transport := &authTransport{token: "test-token", base: recordingTransport{sent: &sent}}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://api.example.test/v0.1/apps/app-123", nil)
	require.NoError(t, err)

	_, err = transport.RoundTrip(req)

	require.NoError(t, err)
	assert.Empty(t, req.Header.Get("Authorization"))
	assert.Equal(t, "test-token", sent.Header.Get("Authorization"))
}

func TestExcerpt_whenCutFallsInsideRune_cutsBeforeRune(t *testing.T) {
	body := []byte(strings.Repeat("a", maxErrorBodyBytes-1) + "é" + "tail")

	got := excerpt(body)

	assert.Equal(t, strings.Repeat("a", maxErrorBodyBytes-1)+" [truncated]", got)
}

// failingTransport fails every round trip with err.
type failingTransport struct {
	err error
}

func (f failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.err
}

// recordingTransport stores the request it receives and answers 200.
type recordingTransport struct {
	sent **http.Request
}

func (r recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	*r.sent = req
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
}
