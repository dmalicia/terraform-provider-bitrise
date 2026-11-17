package provider

import (
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAvailableStacksDataSource_Read_returnsSortedStackKeys(t *testing.T) {
	api := newFakeAPI(t)
	api.update(func(a *fakeAPI) { a.stacks = []string{"osx-xcode-16", "linux-ubuntu-22.04", "osx-xcode-15"} })

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `data "bitrise_available_stacks" "test" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.bitrise_available_stacks.test", "id", "available-stacks"),
				resource.TestCheckResourceAttr("data.bitrise_available_stacks.test", "stack_keys.#", "3"),
				resource.TestCheckResourceAttr("data.bitrise_available_stacks.test", "stack_keys.0", "linux-ubuntu-22.04"),
				resource.TestCheckResourceAttr("data.bitrise_available_stacks.test", "stack_keys.1", "osx-xcode-15"),
				resource.TestCheckResourceAttr("data.bitrise_available_stacks.test", "stack_keys.2", "osx-xcode-16"),
			),
		}},
	})
}

func TestAvailableStacksDataSource_Read_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.fail(http.MethodGet, "/v0.1/available-stacks", http.StatusInternalServerError)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      api.providerConfig() + `data "bitrise_available_stacks" "test" {}`,
			ExpectError: errorMatching("Failed to read available stacks: 500 Internal Server Error"),
		}},
	})
}
