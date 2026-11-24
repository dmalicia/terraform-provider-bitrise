package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func appSSHConfig(api *fakeAPI, publicKey string) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app_ssh" "test" {
  app_slug             = "app-123"
  auth_ssh_private_key = "PRIVATE-KEY"
  auth_ssh_public_key  = %q
}
`, publicKey)
}

func TestAppSSHResource_Create_withoutRegisterFlag_sendsFalse(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appSSHConfig(api, "ssh-ed25519 AAAA"),
			Check:  resource.TestCheckNoResourceAttr("bitrise_app_ssh.test", "is_register_key_into_provider_service"),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"auth_ssh_private_key": "PRIVATE-KEY",
		"auth_ssh_public_key": "ssh-ed25519 AAAA",
		"is_register_key_into_provider_service": false
	}`, requests[0].Body)
}

func TestAppSSHResource_Create_withRegisterFlag_sendsTrue(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug                              = "app-123"
  auth_ssh_private_key                  = "PRIVATE-KEY"
  auth_ssh_public_key                   = "ssh-ed25519 AAAA"
  is_register_key_into_provider_service = true
}
`,
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key")
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body, `"is_register_key_into_provider_service":true`)
}

func TestAppSSHResource_Create_whenAppMissing_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appSSHConfig(api, "ssh-ed25519 AAAA"),
			ExpectError: errorMatching("Request did not succeed: 404 Not Found"),
		}},
	})
}

func TestAppSSHResource_Update_whenKeyChanges_registersNewKey(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSSHConfig(api, "ssh-ed25519 AAAA")},
			{Config: appSSHConfig(api, "ssh-ed25519 BBBB")},
		},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key")
	require.Len(t, requests, 2)
	assert.Contains(t, requests[1].Body, `"auth_ssh_public_key":"ssh-ed25519 BBBB"`)
}

func TestAppSSHResource_Update_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSSHConfig(api, "ssh-ed25519 AAAA")},
			{
				PreConfig: func() {
					api.fail(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key", http.StatusInternalServerError)
				},
				Config:      appSSHConfig(api, "ssh-ed25519 BBBB"),
				ExpectError: errorMatching("Request did not succeed: 500 Internal Server Error"),
			},
		},
	})
}

func TestAppSSHResource_Read_whenAppDeletedRemotely_removesFromState(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSSHConfig(api, "ssh-ed25519 AAAA")},
			{
				PreConfig:          func() { api.removeApp("app-123") },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              checkRemovedFromState("bitrise_app_ssh.test"),
			},
		},
	})
}

func TestAppSSHResource_ImportState_withAppSlug_importsResource(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSSHConfig(api, "ssh-ed25519 AAAA")},
			{
				Config:                               appSSHConfig(api, "ssh-ed25519 AAAA"),
				ResourceName:                         "bitrise_app_ssh.test",
				ImportState:                          true,
				ImportStateId:                        "app-123",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "app_slug",
				// The API never returns registered keys, so only app_slug can be imported.
				ImportStateVerifyIgnore: []string{"auth_ssh_private_key", "auth_ssh_public_key"},
			},
		},
	})
}

func TestAppSSHResource_Schema_privateKeyIsSensitive(t *testing.T) {
	var resp fwresource.SchemaResponse
	NewAppSSHResource().Schema(context.Background(), fwresource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["auth_ssh_private_key"].(schema.StringAttribute)
	require.True(t, ok, "auth_ssh_private_key must be a string attribute")
	assert.True(t, attr.Sensitive, "the private key must be redacted from plan and apply output")
}
