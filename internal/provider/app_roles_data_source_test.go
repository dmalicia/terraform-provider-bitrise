package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAppRolesDataSource_Read_returnsGroupsOfRole(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")
	api.update(func(a *fakeAPI) { a.roles["app-123/admin"] = []string{"group-a", "group-b"} })

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
data "bitrise_app_roles" "test" {
  app_slug  = "app-123"
  role_name = "admin"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.bitrise_app_roles.test", "id", "app-123/admin"),
				resource.TestCheckResourceAttr("data.bitrise_app_roles.test", "groups.#", "2"),
				resource.TestCheckResourceAttr("data.bitrise_app_roles.test", "groups.0", "group-a"),
				resource.TestCheckResourceAttr("data.bitrise_app_roles.test", "groups.1", "group-b"),
			),
		}},
	})
}

func TestAppRolesDataSource_Read_whenAppMissing_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
data "bitrise_app_roles" "test" {
  app_slug  = "app-123"
  role_name = "admin"
}
`,
			ExpectError: errorMatching("Failed to read role groups: 404 Not Found"),
		}},
	})
}
