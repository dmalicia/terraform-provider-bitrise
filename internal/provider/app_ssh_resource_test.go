package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const testPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\nfake\n-----END OPENSSH PRIVATE KEY-----\n"

func TestAppSSHResourceSchema_PrivateKeyIsSensitive(t *testing.T) {
	r := NewAppSSHResource(nil, "", "")
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["auth_ssh_private_key"].(schema.StringAttribute)
	if !ok {
		t.Fatal("auth_ssh_private_key attribute missing or not a string")
	}
	if !attr.Sensitive {
		t.Error("auth_ssh_private_key must be Sensitive so it is redacted from plan output")
	}
}

func TestRegisterSSHKey_SendsKeyFromMemoryWithoutTouchingDisk(t *testing.T) {
	var got Payload
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotPath = req.URL.Path
		body, _ := io.ReadAll(req.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("invalid JSON body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Run in an empty directory so any file written to the working directory
	// (as the previous implementation did with ./testtfkey) is detected.
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	r := NewAppSSHResource(func(string, string) *http.Client { return srv.Client() }, srv.URL, "token")
	err := r.registerSSHKey(context.Background(), AppSSHResourceModel{
		AppSlug:           "my-app",
		AuthSSHPrivateKey: testPrivateKey,
		AuthSSHPublicKey:  "ssh-ed25519 AAAA fake",
	})
	if err != nil {
		t.Fatalf("registerSSHKey returned error: %v", err)
	}

	if gotPath != "/v0.1/apps/my-app/register-ssh-key" {
		t.Errorf("unexpected path %q", gotPath)
	}
	if got.AuthSSHPrivateKey != testPrivateKey {
		t.Errorf("private key not forwarded unchanged")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files in working directory, found %d (first: %s)", len(entries), entries[0].Name())
	}
}

func TestRegisterSSHKey_ReturnsErrorOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	r := NewAppSSHResource(func(string, string) *http.Client { return srv.Client() }, srv.URL, "token")
	err := r.registerSSHKey(context.Background(), AppSSHResourceModel{AppSlug: "my-app"})
	if err == nil {
		t.Fatal("expected an error for a 422 response")
	}
}
