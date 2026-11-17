package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func appFinishConfig(api *fakeAPI, mode string) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app_finish" "test" {
  app_slug          = "app-123"
  project_type      = "ios"
  stack_id          = "osx-xcode-16"
  config            = "default-ios-config"
  mode              = %q
  organization_slug = "org-abc"
  envs = {
    MY_ENV = "value"
  }
}
`, mode)
}

func TestAppFinishResource_Create_postsFinishPayload(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appFinishConfig(api, "manual"),
			Check:  resource.TestCheckResourceAttr("bitrise_app_finish.test", "envs.MY_ENV", "value"),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/finish")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"project_type": "ios",
		"stack_id": "osx-xcode-16",
		"config": "default-ios-config",
		"mode": "manual",
		"envs": {"MY_ENV": "value"},
		"organization_slug": "org-abc"
	}`, requests[0].Body)
}

func TestAppFinishResource_Create_whenAPIRejects_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")
	api.fail(http.MethodPost, "/v0.1/apps/app-123/finish", http.StatusBadRequest)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appFinishConfig(api, "manual"),
			ExpectError: errorMatching("Request did not succeed: 400 Bad Request"),
		}},
	})
}

func TestAppFinishResource_Update_whenModeChanges_postsFinishAgain(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appFinishConfig(api, "manual")},
			{Config: appFinishConfig(api, "auto")},
		},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/finish")
	require.Len(t, requests, 2)
	assert.Contains(t, requests[1].Body, `"mode":"auto"`)
}

func TestAppFinishResource_Update_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appFinishConfig(api, "manual")},
			{
				PreConfig:   func() { api.fail(http.MethodPost, "/v0.1/apps/app-123/finish", http.StatusInternalServerError) },
				Config:      appFinishConfig(api, "auto"),
				ExpectError: errorMatching("Request did not succeed: 500 Internal Server Error"),
			},
		},
	})
}

func TestAppFinishResource_Read_whenAppDeletedRemotely_removesFromState(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appFinishConfig(api, "manual")},
			{
				PreConfig:          func() { api.removeApp("app-123") },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              checkRemovedFromState("bitrise_app_finish.test"),
			},
		},
	})
}

func TestAppFinishResource_Read_whenAPIFails_reportsStatus(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appFinishConfig(api, "manual")},
			{
				PreConfig:    func() { api.fail(http.MethodGet, "/v0.1/apps/app-123", http.StatusServiceUnavailable) },
				RefreshState: true,
				ExpectError:  errorMatching("Unable to read app, got status: 503 Service Unavailable"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appFinishConfig(api, "manual"),
			},
		},
	})
}

func TestAppFinishResource_Delete_doesNotCallAPI(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{Config: appFinishConfig(api, "manual")}},
	})

	for _, r := range api.allRequests() {
		assert.NotEqual(t, http.MethodDelete, r.Method, r.Path)
	}
}

func TestAppFinishResource_ImportState_withAppSlug_importsResource(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appFinishConfig(api, "manual")},
			{
				Config:                               appFinishConfig(api, "manual"),
				ResourceName:                         "bitrise_app_finish.test",
				ImportState:                          true,
				ImportStateId:                        "app-123",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "app_slug",
				// The API does not return the finish settings, so only app_slug can be imported.
				ImportStateVerifyIgnore: []string{"project_type", "stack_id", "config", "mode", "envs", "organization_slug"},
			},
		},
	})
}
