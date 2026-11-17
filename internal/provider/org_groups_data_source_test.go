package provider

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestOrgGroupsDataSource_Read_returnsGroups(t *testing.T) {
	api := newFakeAPI(t)
	api.update(func(a *fakeAPI) {
		a.groups["org-abc"] = []GroupAPIModel{{Slug: "group-a", Name: "Developers"}, {Slug: "group-b", Name: "Testers"}}
	})

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `data "bitrise_org_groups" "test" { org_slug = "org-abc" }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.bitrise_org_groups.test", "id", "org-abc"),
				resource.TestCheckResourceAttr("data.bitrise_org_groups.test", "groups.#", "2"),
				resource.TestCheckResourceAttr("data.bitrise_org_groups.test", "groups.0.slug", "group-a"),
				resource.TestCheckResourceAttr("data.bitrise_org_groups.test", "groups.0.name", "Developers"),
				resource.TestCheckResourceAttr("data.bitrise_org_groups.test", "groups.1.slug", "group-b"),
			),
		}},
	})
}

func TestOrgGroupsDataSource_Read_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.fail(http.MethodGet, "/v0.1/organizations/org-abc/groups", http.StatusForbidden)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      api.providerConfig() + `data "bitrise_org_groups" "test" { org_slug = "org-abc" }`,
			ExpectError: errorMatching("Failed to read organization groups: 403 Forbidden"),
		}},
	})
}

func TestOrgGroupsDataSource_Read_whenResponseMalformed_reportsParseError(t *testing.T) {
	api := newFakeAPI(t)
	api.override(http.MethodGet, "/v0.1/organizations/org-abc/groups", http.StatusOK, "application/json", `{"slug":"group-a"}`)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      api.providerConfig() + `data "bitrise_org_groups" "test" { org_slug = "org-abc" }`,
			ExpectError: errorMatching("Error parsing response"),
		}},
	})
}
