package provider

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rolePath = "/v0.1/apps/app-123/roles/admin"

func appRolesConfig(api *fakeAPI, groups ...string) string {
	quoted := make([]string, 0, len(groups))
	for _, g := range groups {
		quoted = append(quoted, fmt.Sprintf("%q", g))
	}
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app_roles" "test" {
  app_slug  = "app-123"
  role_name = "admin"
  groups    = [%s]
}
`, strings.Join(quoted, ", "))
}

func TestAppRolesResource_Create_putsGroups(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appRolesConfig(api, "group-a", "group-b"),
			Check:  resource.TestCheckResourceAttr("bitrise_app_roles.test", "id", "app-123/admin"),
		}},
	})

	requests := api.received(http.MethodPut, rolePath)
	require.NotEmpty(t, requests)
	assert.JSONEq(t, `{"groups": ["group-a", "group-b"]}`, requests[0].Body)
}

func TestAppRolesResource_Create_whenAPIRejects_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")
	api.fail(http.MethodPut, rolePath, http.StatusForbidden)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appRolesConfig(api, "group-a"),
			ExpectError: errorMatching("Failed to update role groups: 403 Forbidden"),
		}},
	})
}

func TestAppRolesResource_Update_whenGroupsChange_putsNewGroups(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appRolesConfig(api, "group-a", "group-b")},
			{
				Config: appRolesConfig(api, "group-b"),
				Check:  resource.TestCheckResourceAttr("bitrise_app_roles.test", "groups.#", "1"),
			},
		},
	})

	requests := api.received(http.MethodPut, rolePath)
	require.GreaterOrEqual(t, len(requests), 2)
	assert.JSONEq(t, `{"groups": ["group-b"]}`, requests[1].Body)
}

func TestAppRolesResource_Delete_putsEmptyGroups(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{Config: appRolesConfig(api, "group-a")}},
	})

	requests := api.received(http.MethodPut, rolePath)
	require.Len(t, requests, 2)
	assert.JSONEq(t, `{"groups": []}`, requests[1].Body)
}

func TestAppRolesResource_Read_whenGroupsChangedRemotely_detectsDrift(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appRolesConfig(api, "group-a")},
			{
				PreConfig:          func() { api.update(func(a *fakeAPI) { a.roles["app-123/admin"] = []string{"group-a", "group-c"} }) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr("bitrise_app_roles.test", "groups.1", "group-c"),
			},
		},
	})
}

func TestAppRolesResource_Read_whenAppDeletedRemotely_removesFromState(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appRolesConfig(api, "group-a")},
			{
				PreConfig:          func() { api.removeApp("app-123") },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              checkRemovedFromState("bitrise_app_roles.test"),
			},
		},
	})
}

func TestAppRolesResource_Read_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appRolesConfig(api, "group-a")},
			{
				PreConfig:    func() { api.fail(http.MethodGet, rolePath, http.StatusInternalServerError) },
				RefreshState: true,
				ExpectError:  errorMatching("Failed to read role groups: 500 Internal Server Error"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appRolesConfig(api, "group-a"),
			},
		},
	})
}

func TestAppRolesResource_ImportState_withAppAndRole_importsGroups(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appRolesConfig(api, "group-a", "group-b")},
			{
				Config:            appRolesConfig(api, "group-a", "group-b"),
				ResourceName:      "bitrise_app_roles.test",
				ImportState:       true,
				ImportStateId:     "app-123/admin",
				ImportStateVerify: true,
			},
		},
	})
}

func TestAppRolesResource_ImportState_withInvalidID_reportsError(t *testing.T) {
	cases := map[string]string{
		"missing role name": "app-123",
		"extra segment":     "app-123/admin/extra",
	}
	for name, importID := range cases {
		t.Run(name, func(t *testing.T) {
			api := newFakeAPI(t)
			api.addApp("app-123")

			runUnitTest(t, resource.TestCase{
				Steps: []resource.TestStep{
					{Config: appRolesConfig(api, "group-a")},
					{
						Config:        appRolesConfig(api, "group-a"),
						ResourceName:  "bitrise_app_roles.test",
						ImportState:   true,
						ImportStateId: importID,
						ExpectError:   errorMatching("Import ID must be in the format 'app_slug/role_name'"),
					},
				},
			})
		})
	}
}
