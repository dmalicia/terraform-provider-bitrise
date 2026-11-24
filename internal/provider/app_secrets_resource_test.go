package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const appSecretPath = "/v0.1/apps/app-123/secrets/MY_SECRET"

func appSecretConfig(api *fakeAPI, value, extra string) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app_secret" "test" {
  app_slug = "app-123"
  name     = "MY_SECRET"
  value    = %q
  %s
}
`, value, extra)
}

func TestAppSecretResource_Create_sendsValueAndAllFlags(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appSecretConfig(api, "s3cr3t", ""),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("bitrise_app_secret.test", "id", "app-123/MY_SECRET"),
				resource.TestCheckResourceAttr("bitrise_app_secret.test", "expand_in_step_inputs", "true"),
			),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/secrets")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"name": "MY_SECRET",
		"value": "s3cr3t",
		"is_protected": false,
		"is_exposed_for_pull_requests": false,
		"expand_in_step_inputs": true
	}`, requests[0].Body)
}

func TestAppSecretResource_Create_withExpandDisabled_sendsFalse(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appSecretConfig(api, "s3cr3t", "expand_in_step_inputs = false"),
			Check:  resource.TestCheckResourceAttr("bitrise_app_secret.test", "expand_in_step_inputs", "false"),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/app-123/secrets")
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body, `"expand_in_step_inputs":false`)
}

func TestAppSecretResource_Create_whenSecretExists_reportsConflict(t *testing.T) {
	api := newFakeAPI(t)
	api.update(func(a *fakeAPI) {
		a.secrets["app-123/MY_SECRET"] = &SecretResponse{Name: "MY_SECRET", Value: "old"}
	})

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appSecretConfig(api, "s3cr3t", ""),
			ExpectError: errorMatching("Failed to create secret: 409 Conflict"),
		}},
	})
}

func TestAppSecretResource_Read_whenValueChangedRemotely_detectsDrift(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:          func() { api.update(func(a *fakeAPI) { a.secrets["app-123/MY_SECRET"].Value = "changed" }) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr("bitrise_app_secret.test", "value", "changed"),
			},
		},
	})
}

func TestAppSecretResource_Read_whenProtected_keepsConfiguredValue(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appSecretConfig(api, "s3cr3t", "is_protected = true"),
			Check:  resource.TestCheckResourceAttr("bitrise_app_secret.test", "value", "s3cr3t"),
		}},
	})
}

func TestAppSecretResource_Read_whenSecretDeletedRemotely_removesFromState(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:          func() { api.update(func(a *fakeAPI) { delete(a.secrets, "app-123/MY_SECRET") }) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              checkRemovedFromState("bitrise_app_secret.test"),
			},
		},
	})
}

func TestAppSecretResource_Read_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:    func() { api.fail(http.MethodGet, appSecretPath, http.StatusInternalServerError) },
				RefreshState: true,
				ExpectError:  errorMatching("Failed to read secret: 500 Internal Server Error"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appSecretConfig(api, "s3cr3t", ""),
			},
		},
	})
}

func TestAppSecretResource_Update_whenValueChanges_patchesSecret(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{Config: appSecretConfig(api, "rotated", "is_exposed_for_pull_requests = true")},
		},
	})

	requests := api.received(http.MethodPatch, appSecretPath)
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"value": "rotated",
		"is_protected": false,
		"is_exposed_for_pull_requests": true,
		"expand_in_step_inputs": true
	}`, requests[0].Body)
}

func TestAppSecretResource_Update_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:   func() { api.fail(http.MethodPatch, appSecretPath, http.StatusBadRequest) },
				Config:      appSecretConfig(api, "rotated", ""),
				ExpectError: errorMatching("Failed to update secret: 400 Bad Request"),
			},
		},
	})
}

func TestAppSecretResource_Delete_deletesSecret(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{Config: appSecretConfig(api, "s3cr3t", "")}},
		CheckDestroy: func(*terraform.State) error {
			if len(api.received(http.MethodDelete, appSecretPath)) != 1 {
				return fmt.Errorf("MY_SECRET was not deleted")
			}
			return nil
		},
	})
}

func TestAppSecretResource_Delete_whenAlreadyDeleted_succeeds(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig: func() { api.fail(http.MethodDelete, appSecretPath, http.StatusNotFound) },
				Config:    appSecretConfig(api, "s3cr3t", ""),
				Destroy:   true,
			},
		},
	})

	assert.Len(t, api.received(http.MethodDelete, appSecretPath), 1)
}

func TestAppSecretResource_Delete_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:   func() { api.fail(http.MethodDelete, appSecretPath, http.StatusInternalServerError) },
				Config:      appSecretConfig(api, "s3cr3t", ""),
				Destroy:     true,
				ExpectError: errorMatching("Failed to delete secret: 500 Internal Server Error"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appSecretConfig(api, "s3cr3t", ""),
			},
		},
	})
}

func TestAppSecretResource_ImportState_withAppAndName_importsSecret(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				Config:            appSecretConfig(api, "s3cr3t", ""),
				ResourceName:      "bitrise_app_secret.test",
				ImportState:       true,
				ImportStateId:     "app-123/MY_SECRET",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAppSecretResource_ImportState_withInvalidID_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appSecretConfig(api, "s3cr3t", "")},
			{
				Config:        appSecretConfig(api, "s3cr3t", ""),
				ResourceName:  "bitrise_app_secret.test",
				ImportState:   true,
				ImportStateId: "MY_SECRET",
				ExpectError:   errorMatching("Import ID must be in the format 'app_slug/secret_name'"),
			},
		},
	})
}
