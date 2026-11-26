package provider

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func appConfig(api *fakeAPI, repoURL string) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app" "test" {
  repo              = "github"
  repo_url          = %q
  type              = "git"
  git_repo_slug     = "repo"
  git_owner         = "example"
  organization_slug = "org-abc"
  is_public         = true
}
`, repoURL)
}

const testRepoURL = "https://github.com/example/repo.git"

func TestAppResource_Create_registersAppAndStoresSlug(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appConfig(api, testRepoURL),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("bitrise_app.test", "app_slug", "app-123"),
				resource.TestCheckResourceAttr("bitrise_app.test", "id", "app-123"),
			),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/register")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"git_owner": "example",
		"git_repo_slug": "repo",
		"is_public": true,
		"organization_slug": "org-abc",
		"provider": "github",
		"repo_url": "https://github.com/example/repo.git",
		"type": "git"
	}`, requests[0].Body)
}

func TestAppResource_Create_withoutIsPublic_hasEmptyPlanAfterApply(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `
resource "bitrise_app" "test" {
  repo     = "github"
  repo_url = "https://github.com/example/repo.git"
}
`,
			Check: resource.TestCheckNoResourceAttr("bitrise_app.test", "is_public"),
		}},
	})
}

func TestAppResource_Create_whenAPIRejects_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.fail(http.MethodPost, "/v0.1/apps/register", http.StatusUnprocessableEntity)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appConfig(api, testRepoURL),
			ExpectError: errorMatching("Request did not succeed: 422 Unprocessable Entity"),
		}},
	})
}

func TestAppResource_Create_whenResponseIsMalformed_reportsParseError(t *testing.T) {
	api := newFakeAPI(t)
	api.override(http.MethodPost, "/v0.1/apps/register", http.StatusOK, "application/json", "not json")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appConfig(api, testRepoURL),
			ExpectError: errorMatching("Unable to parse API response"),
		}},
	})
}

func TestAppResource_Update_whenRepoURLChanges_patchesApp(t *testing.T) {
	api := newFakeAPI(t)
	const newRepoURL = "https://github.com/example/other.git"

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				Config: appConfig(api, newRepoURL),
				Check:  resource.TestCheckResourceAttr("bitrise_app.test", "repo_url", newRepoURL),
			},
		},
	})

	requests := api.received(http.MethodPatch, "/v0.1/apps/app-123")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{"is_public": true, "repository_url": "https://github.com/example/other.git"}`, requests[0].Body)
}

func TestAppResource_Update_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	const newRepoURL = "https://github.com/example/other.git"

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				PreConfig:   func() { api.fail(http.MethodPatch, "/v0.1/apps/app-123", http.StatusInternalServerError) },
				Config:      appConfig(api, newRepoURL),
				ExpectError: errorMatching("Request did not succeed: 500 Internal Server Error"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appConfig(api, newRepoURL),
			},
		},
	})
}

func TestAppResource_Read_whenAppDeletedRemotely_removesFromState(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				PreConfig:          func() { api.removeApp("app-123") },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              checkRemovedFromState("bitrise_app.test"),
			},
		},
	})
}

func TestAppResource_Read_whenAPIFails_reportsStatus(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				PreConfig:    func() { api.fail(http.MethodGet, "/v0.1/apps/app-123", http.StatusInternalServerError) },
				RefreshState: true,
				ExpectError:  errorMatching("Unable to read app, got status: 500 Internal Server Error"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appConfig(api, testRepoURL),
			},
		},
	})
}

func TestAppResource_Read_whenResponseTooLarge_reportsReadError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				PreConfig: func() {
					api.override(http.MethodGet, "/v0.1/apps/app-123", http.StatusOK, "application/json", oversizedJSON())
				},
				RefreshState: true,
				ExpectError:  errorMatching("Unable to read response: response body exceeds 1 MiB"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appConfig(api, testRepoURL),
			},
		},
	})
}

func TestAppResource_Delete_deletesApp(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{Config: appConfig(api, testRepoURL)}},
		CheckDestroy: func(*terraform.State) error {
			if len(api.received(http.MethodDelete, "/v0.1/apps/app-123")) != 1 {
				return fmt.Errorf("app-123 was not deleted")
			}
			return nil
		},
	})
}

func TestAppResource_Delete_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				PreConfig:   func() { api.fail(http.MethodDelete, "/v0.1/apps/app-123", http.StatusInternalServerError) },
				Config:      appConfig(api, testRepoURL),
				Destroy:     true,
				ExpectError: errorMatching("Unable to delete app: 500 Internal Server Error"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appConfig(api, testRepoURL),
			},
		},
	})
}

func TestAppResource_ImportState_withAppSlug_importsRemoteApp(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				Config:                  appConfig(api, testRepoURL),
				ResourceName:            "bitrise_app.test",
				ImportState:             true,
				ImportStateId:           "app-123",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"organization_slug", "type", "title"},
			},
		},
	})
}

func TestAppResource_ImportState_whenAppMissing_reportsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				Config:        appConfig(api, testRepoURL),
				ResourceName:  "bitrise_app.test",
				ImportState:   true,
				ImportStateId: "app-999",
				ExpectError:   errorMatching("Cannot import non-existent remote object"),
			},
		},
	})
}

func appConfigWithTitle(api *fakeAPI, title string) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app" "test" {
  repo              = "github"
  repo_url          = %q
  type              = "git"
  git_repo_slug     = "repo"
  git_owner         = "example"
  organization_slug = "org-abc"
  is_public         = true
  title             = %q
}
`, testRepoURL, title)
}

func TestAppResource_Create_withTitle_registersAppWithTitle(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appConfigWithTitle(api, "team-repo"),
			Check:  resource.TestCheckResourceAttr("bitrise_app.test", "title", "team-repo"),
		}},
	})

	requests := api.received(http.MethodPost, "/v0.1/apps/register")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{
		"git_owner": "example",
		"git_repo_slug": "repo",
		"is_public": true,
		"organization_slug": "org-abc",
		"provider": "github",
		"repo_url": "https://github.com/example/repo.git",
		"title": "team-repo",
		"type": "git"
	}`, requests[0].Body)
}

func TestAppResource_Read_withoutTitle_leavesRemoteTitleUnmanaged(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: appConfig(api, testRepoURL),
				Check:  resource.TestCheckNoResourceAttr("bitrise_app.test", "title"),
			},
			{
				PreConfig: func() { api.update(func(a *fakeAPI) { a.apps["app-123"].Title = "renamed-in-ui" }) },
				Config:    appConfig(api, testRepoURL),
				Check:     resource.TestCheckNoResourceAttr("bitrise_app.test", "title"),
			},
		},
	})

	assert.Empty(t, api.received(http.MethodPatch, "/v0.1/apps/app-123"))
}

func TestAppResource_Update_whenTitleSetOnExistingApp_patchesTitle(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfig(api, testRepoURL)},
			{
				Config: appConfigWithTitle(api, "team-repo"),
				Check:  resource.TestCheckResourceAttr("bitrise_app.test", "title", "team-repo"),
			},
		},
	})

	requests := api.received(http.MethodPatch, "/v0.1/apps/app-123")
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{"is_public": true, "repository_url": "https://github.com/example/repo.git", "title": "team-repo"}`, requests[0].Body)
}

func TestAppResource_Read_whenTitleChangedRemotely_restoresTitle(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfigWithTitle(api, "team-repo")},
			{
				PreConfig: func() { api.update(func(a *fakeAPI) { a.apps["app-123"].Title = "renamed-in-ui" }) },
				Config:    appConfigWithTitle(api, "team-repo"),
				Check:     resource.TestCheckResourceAttr("bitrise_app.test", "title", "team-repo"),
			},
		},
	})

	requests := api.received(http.MethodPatch, "/v0.1/apps/app-123")
	require.Len(t, requests, 1)
	assert.Contains(t, requests[0].Body, `"title":"team-repo"`)
}

func TestAppResource_ImportState_withTitle_importsRemoteTitle(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appConfigWithTitle(api, "team-repo")},
			{
				Config:                  appConfigWithTitle(api, "team-repo"),
				ResourceName:            "bitrise_app.test",
				ImportState:             true,
				ImportStateId:           "app-123",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"organization_slug", "type"},
			},
		},
	})
}

func TestAppResource_Validate_whenTitleEmpty_returnsError(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appConfigWithTitle(api, ""),
			ExpectError: regexp.MustCompile(`Must not be empty`),
		}},
	})

	assert.Empty(t, api.allRequests())
}
