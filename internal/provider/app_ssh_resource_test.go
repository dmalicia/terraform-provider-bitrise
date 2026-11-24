package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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

func appSSHWriteOnlyConfig(api *fakeAPI, privateKey string, version int) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app_ssh" "test" {
  app_slug                        = "app-123"
  auth_ssh_private_key_wo         = %q
  auth_ssh_private_key_wo_version = %d
  auth_ssh_public_key             = "ssh-ed25519 AAAA"
}
`, privateKey, version)
}

func TestAppSSHResource_Create_withWriteOnlyKey_sendsKeyWithoutStoringIt(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{{
			Config: appSSHWriteOnlyConfig(api, "WO-KEY-1", 1),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckNoResourceAttr("bitrise_app_ssh.test", "auth_ssh_private_key_wo"),
				resource.TestCheckNoResourceAttr("bitrise_app_ssh.test", "auth_ssh_private_key"),
				resource.TestCheckResourceAttr("bitrise_app_ssh.test", "auth_ssh_private_key_wo_version", "1"),
			),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key")
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body, `"auth_ssh_private_key":"WO-KEY-1"`)
}

func TestAppSSHResource_Update_whenWriteOnlyVersionChanges_registersNewKey(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{Config: appSSHWriteOnlyConfig(api, "WO-KEY-1", 1)},
			{Config: appSSHWriteOnlyConfig(api, "WO-KEY-2", 2)},
		},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key")
	require.Len(t, requests, 2)
	assert.Contains(t, requests[1].Body, `"auth_ssh_private_key":"WO-KEY-2"`)
}

func TestAppSSHResource_Update_fromStoredToWriteOnlyKey_removesKeyFromState(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{
			{
				Config: appSSHConfig(api, "ssh-ed25519 AAAA"),
				Check:  resource.TestCheckResourceAttr("bitrise_app_ssh.test", "auth_ssh_private_key", "PRIVATE-KEY"),
			},
			{
				Config: appSSHWriteOnlyConfig(api, "WO-KEY-1", 1),
				Check:  resource.TestCheckNoResourceAttr("bitrise_app_ssh.test", "auth_ssh_private_key"),
			},
		},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key")
	require.Len(t, requests, 2)
	assert.Contains(t, requests[1].Body, `"auth_ssh_private_key":"WO-KEY-1"`)
}

func TestAppSSHResource_Create_withWriteOnlyKeyBeforeTerraform111_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipAbove(tfversion.Version1_10_0)},
		Steps: []resource.TestStep{{
			Config:      appSSHWriteOnlyConfig(api, "WO-KEY-1", 1),
			ExpectError: errorMatching("Write-only attributes are only supported in Terraform 1.11 and later"),
		}},
	})

	assert.Empty(t, api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key"))
}

func TestAppSSHResource_Validate_withBothKeys_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug                = "app-123"
  auth_ssh_private_key    = "PRIVATE-KEY"
  auth_ssh_private_key_wo = "WO-KEY"
  auth_ssh_public_key     = "ssh-ed25519 AAAA"
}
`,
			ExpectError: errorMatching("Set only one of auth_ssh_private_key or auth_ssh_private_key_wo."),
		}},
	})
}

func TestAppSSHResource_Validate_withoutKey_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug            = "app-123"
  auth_ssh_public_key = "ssh-ed25519 AAAA"
}
`,
			ExpectError: errorMatching("Set one of auth_ssh_private_key or auth_ssh_private_key_wo."),
		}},
	})
}

func TestAppSSHResource_Validate_withVersionButStoredKey_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug                        = "app-123"
  auth_ssh_private_key            = "PRIVATE-KEY"
  auth_ssh_private_key_wo_version = 1
  auth_ssh_public_key             = "ssh-ed25519 AAAA"
}
`,
			ExpectError: errorMatching("auth_ssh_private_key_wo_version can only be set together with auth_ssh_private_key_wo."),
		}},
	})
}

// timestamp() is unknown at plan time, so these keys only resolve during apply.
func TestAppSSHResource_Create_whenWriteOnlyKeyResolvesToNull_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug                = "app-123"
  auth_ssh_private_key_wo = timestamp() == "" ? "WO-KEY" : null
  auth_ssh_public_key     = "ssh-ed25519 AAAA"
}
`,
			ExpectError: errorMatching("Set one of auth_ssh_private_key or auth_ssh_private_key_wo."),
		}},
	})

	assert.Empty(t, api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key"))
}

func TestAppSSHResource_Create_whenKeysConflictAfterResolving_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug                = "app-123"
  auth_ssh_private_key    = timestamp() == "" ? null : "PRIVATE-KEY"
  auth_ssh_private_key_wo = "WO-KEY"
  auth_ssh_public_key     = "ssh-ed25519 AAAA"
}
`,
			ExpectError: errorMatching("Set only one of auth_ssh_private_key or auth_ssh_private_key_wo."),
		}},
	})

	assert.Empty(t, api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key"))
}

func TestAppSSHResource_Create_whenVersionResolvesToNull_registersStoredKey(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug                        = "app-123"
  auth_ssh_private_key            = "PRIVATE-KEY"
  auth_ssh_private_key_wo_version = timestamp() == "" ? 1 : null
  auth_ssh_public_key             = "ssh-ed25519 AAAA"
}
`,
			ExpectNonEmptyPlan: true, // timestamp() stays unknown on every plan
		}},
	})

	require.Len(t, api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key"), 1)
}

func TestAppSSHResource_Create_whenVersionResolvesWithStoredKey_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app_ssh" "test" {
  app_slug                        = "app-123"
  auth_ssh_private_key            = "PRIVATE-KEY"
  auth_ssh_private_key_wo_version = timestamp() == "" ? null : 1
  auth_ssh_public_key             = "ssh-ed25519 AAAA"
}
`,
			ExpectError: errorMatching("auth_ssh_private_key_wo_version can only be set together with auth_ssh_private_key_wo."),
		}},
	})

	assert.Empty(t, api.received(http.MethodPost, "/v0.1/apps/app-123/register-ssh-key"))
}

func TestAppSSHResource_Schema_writeOnlyKeyIsSensitiveAndWriteOnly(t *testing.T) {
	var resp fwresource.SchemaResponse
	NewAppSSHResource().Schema(context.Background(), fwresource.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["auth_ssh_private_key_wo"].(schema.StringAttribute)
	require.True(t, ok)
	assert.True(t, attr.Sensitive)
	assert.True(t, attr.WriteOnly)
}
