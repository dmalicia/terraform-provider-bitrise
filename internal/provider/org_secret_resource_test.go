package provider

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const orgSecretPath = "/v0.1/organizations/org-abc/secrets/MY_SECRET"

func orgSecretConfig(api *fakeAPI, value, extra string) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_org_secret" "test" {
  org_slug = "org-abc"
  name     = "MY_SECRET"
  value    = %q
  %s
}
`, value, extra)
}

func TestOrgSecretResource_Create_sendsSecretAndStoresScope(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: orgSecretConfig(api, "s3cr3t", ""),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("bitrise_org_secret.test", "id", "org-abc/MY_SECRET"),
				resource.TestCheckResourceAttr("bitrise_org_secret.test", "scope", "workspace"),
			),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/organizations/org-abc/secrets")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"name": "MY_SECRET",
		"value": "s3cr3t",
		"is_protected": false,
		"is_exposed_for_pull_requests": false,
		"expand_in_step_inputs": true
	}`, requests[0].Body)
}

func TestOrgSecretResource_Create_whenSecretExists_reportsConflict(t *testing.T) {
	api := newFakeAPI(t)
	api.update(func(a *fakeAPI) {
		a.orgSecrets["org-abc/MY_SECRET"] = &orgSecretResponse{Name: "MY_SECRET", Value: "old"}
	})

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      orgSecretConfig(api, "s3cr3t", ""),
			ExpectError: errorMatching("Failed to create org secret: HTTP 409"),
		}},
	})
}

func TestOrgSecretResource_Update_whenValueChanges_patchesSecret(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				Config: orgSecretConfig(api, "rotated", "is_exposed_for_pull_requests = true"),
				Check:  resource.TestCheckResourceAttr("bitrise_org_secret.test", "is_exposed_for_pull_requests", "true"),
			},
		},
	})

	requests := api.received(http.MethodPatch, orgSecretPath)
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"value": "rotated",
		"is_protected": false,
		"is_exposed_for_pull_requests": true,
		"expand_in_step_inputs": true
	}`, requests[0].Body)
}

func TestOrgSecretResource_Update_whenResponseHasNoBody_keepsScope(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig: func() { api.update(func(a *fakeAPI) { a.omitOrgSecretUpdateBody = true }) },
				Config:    orgSecretConfig(api, "rotated", ""),
				Check:     resource.TestCheckResourceAttr("bitrise_org_secret.test", "scope", "workspace"),
			},
		},
	})
}

func TestOrgSecretResource_Update_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:   func() { api.fail(http.MethodPatch, orgSecretPath, http.StatusBadRequest) },
				Config:      orgSecretConfig(api, "rotated", ""),
				ExpectError: errorMatching("Failed to update org secret: HTTP 400"),
			},
		},
	})
}

func TestOrgSecretResource_Update_whenProtecting_updatesInPlace(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{Config: orgSecretConfig(api, "s3cr3t", "is_protected = true")},
		},
	})

	assert.Len(t, api.received(http.MethodPatch, orgSecretPath), 1)
	assert.Len(t, api.received(http.MethodPost, "/v0.1/organizations/org-abc/secrets"), 1)
}

func TestOrgSecretResource_Update_whenUnprotecting_replacesSecret(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "is_protected = true")},
			{
				Config: orgSecretConfig(api, "s3cr3t", "is_protected = false"),
				Check:  resource.TestCheckResourceAttr("bitrise_org_secret.test", "is_protected", "false"),
			},
		},
	})

	assert.Empty(t, api.received(http.MethodPatch, orgSecretPath))
	assert.Len(t, api.received(http.MethodPost, "/v0.1/organizations/org-abc/secrets"), 2)
}

func TestOrgSecretResource_Read_whenProtected_keepsConfiguredValue(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: orgSecretConfig(api, "s3cr3t", "is_protected = true"),
			Check:  resource.TestCheckResourceAttr("bitrise_org_secret.test", "value", "s3cr3t"),
		}},
	})
}

func TestOrgSecretResource_Read_whenValueChangedRemotely_detectsDrift(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:          func() { api.update(func(a *fakeAPI) { a.orgSecrets["org-abc/MY_SECRET"].Value = "changed" }) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr("bitrise_org_secret.test", "value", "changed"),
			},
		},
	})
}

func TestOrgSecretResource_Read_whenSecretDeletedRemotely_removesFromState(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:          func() { api.update(func(a *fakeAPI) { delete(a.orgSecrets, "org-abc/MY_SECRET") }) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              checkRemovedFromState("bitrise_org_secret.test"),
			},
		},
	})
}

func TestOrgSecretResource_Read_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:    func() { api.fail(http.MethodGet, orgSecretPath, http.StatusInternalServerError) },
				RefreshState: true,
				ExpectError:  errorMatching("Failed to read org secret: HTTP 500"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    orgSecretConfig(api, "s3cr3t", ""),
			},
		},
	})
}

func TestOrgSecretResource_Read_whenResponseMalformed_reportsParseError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:    func() { api.override(http.MethodGet, orgSecretPath, http.StatusOK, "application/json", "{bad") },
				RefreshState: true,
				ExpectError:  errorMatching("Error parsing response"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    orgSecretConfig(api, "s3cr3t", ""),
			},
		},
	})
}

func TestOrgSecretResource_Delete_whenAlreadyDeleted_succeeds(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig: func() { api.fail(http.MethodDelete, orgSecretPath, http.StatusNotFound) },
				Config:    orgSecretConfig(api, "s3cr3t", ""),
				Destroy:   true,
			},
		},
	})

	assert.Len(t, api.received(http.MethodDelete, orgSecretPath), 1)
}

func TestOrgSecretResource_Delete_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				PreConfig:   func() { api.fail(http.MethodDelete, orgSecretPath, http.StatusInternalServerError) },
				Config:      orgSecretConfig(api, "s3cr3t", ""),
				Destroy:     true,
				ExpectError: errorMatching("Failed to delete org secret: HTTP 500"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    orgSecretConfig(api, "s3cr3t", ""),
			},
		},
	})
}

func TestOrgSecretResource_ImportState_withOrgAndName_importsSecret(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: orgSecretConfig(api, "s3cr3t", "")},
			{
				Config:            orgSecretConfig(api, "s3cr3t", ""),
				ResourceName:      "bitrise_org_secret.test",
				ImportState:       true,
				ImportStateId:     "org-abc/MY_SECRET",
				ImportStateVerify: true,
			},
		},
	})
}

func TestOrgSecretResource_ImportState_withInvalidID_reportsError(t *testing.T) {
	cases := map[string]string{
		"missing name":      "org-abc",
		"empty org slug":    "/MY_SECRET",
		"empty secret name": "org-abc/",
	}
	for name, importID := range cases {
		t.Run(name, func(t *testing.T) {
			api := newFakeAPI(t)

			runUnitTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{Config: orgSecretConfig(api, "s3cr3t", "")},
					{
						Config:        orgSecretConfig(api, "s3cr3t", ""),
						ResourceName:  "bitrise_org_secret.test",
						ImportState:   true,
						ImportStateId: importID,
						ExpectError:   errorMatching("Import ID must be in the format 'org_slug/secret_name'"),
					},
				},
			})
		})
	}
}
