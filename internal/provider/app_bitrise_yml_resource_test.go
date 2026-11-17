package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"terraform-provider-bitrise/internal/bitriseapi"
)

const ymlPath = "/v0.1/apps/app-123/bitrise.yml"

func appBitriseYmlConfig(api *fakeAPI, yml, extra string) string {
	return api.providerConfig() + fmt.Sprintf(`
resource "bitrise_app_bitrise_yml" "test" {
  app_slug    = "app-123"
  yml_content = %q
  %s
}
`, yml, extra)
}

func TestAppBitriseYmlResource_Create_uploadsYml(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: appBitriseYmlConfig(api, "format_version: 13", ""),
			Check:  resource.TestCheckResourceAttr("bitrise_app_bitrise_yml.test", "id", "app-123"),
		}},
	})

	requests := api.received(http.MethodPost, ymlPath)
	require.Len(t, requests, 1)
	assert.JSONEq(t, `{"app_config_datastore_yaml": "format_version: 13"}`, requests[0].Body)
}

func TestAppBitriseYmlResource_Create_whenAPIRejects_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")
	api.fail(http.MethodPost, ymlPath, http.StatusUnprocessableEntity)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      appBitriseYmlConfig(api, "format_version: 13", ""),
			ExpectError: errorMatching("Request failed with status 422"),
		}},
	})
}

func TestAppBitriseYmlResource_Update_whenContentChanges_uploadsNewYml(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				Config: appBitriseYmlConfig(api, "format_version: 14", ""),
				Check:  resource.TestCheckResourceAttr("bitrise_app_bitrise_yml.test", "yml_content", "format_version: 14"),
			},
		},
	})

	requests := api.received(http.MethodPost, ymlPath)
	require.Len(t, requests, 2)
	assert.JSONEq(t, `{"app_config_datastore_yaml": "format_version: 14"}`, requests[1].Body)
}

func TestAppBitriseYmlResource_Update_whenAPIFails_reportsError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				PreConfig:   func() { api.fail(http.MethodPost, ymlPath, http.StatusInternalServerError) },
				Config:      appBitriseYmlConfig(api, "format_version: 14", ""),
				ExpectError: errorMatching("Request failed with status 500"),
			},
		},
	})
}

func TestAppBitriseYmlResource_Update_withUpdateOnCreateOnly_ignoresContentChanges(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "update_on_create_only = true")},
			{
				Config: appBitriseYmlConfig(api, "format_version: 14", "update_on_create_only = true"),
				Check:  resource.TestCheckResourceAttr("bitrise_app_bitrise_yml.test", "yml_content", "format_version: 13"),
			},
		},
	})

	assert.Len(t, api.received(http.MethodPost, ymlPath), 1)
}

func TestAppBitriseYmlResource_Update_whenEnablingUpdateOnCreateOnly_skipsUpload(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				Config: appBitriseYmlConfig(api, "format_version: 14", "update_on_create_only = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bitrise_app_bitrise_yml.test", "update_on_create_only", "true"),
					resource.TestCheckResourceAttr("bitrise_app_bitrise_yml.test", "yml_content", "format_version: 13"),
				),
			},
		},
	})

	assert.Len(t, api.received(http.MethodPost, ymlPath), 1)
}

func TestAppBitriseYmlResource_Read_whenYmlChangedRemotely_detectsDrift(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				PreConfig:          func() { api.update(func(a *fakeAPI) { a.ymls["app-123"] = "format_version: 12" }) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              resource.TestCheckResourceAttr("bitrise_app_bitrise_yml.test", "yml_content", "format_version: 12"),
			},
		},
	})
}

func TestAppBitriseYmlResource_Read_whenYmlMissing_removesFromState(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				PreConfig:          func() { api.update(func(a *fakeAPI) { delete(a.ymls, "app-123") }) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				Check:              checkRemovedFromState("bitrise_app_bitrise_yml.test"),
			},
		},
	})
}

func TestAppBitriseYmlResource_Read_whenAPIFails_reportsStatus(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				PreConfig:    func() { api.fail(http.MethodGet, ymlPath, http.StatusInternalServerError) },
				RefreshState: true,
				ExpectError:  errorMatching("Request failed with status 500"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appBitriseYmlConfig(api, "format_version: 13", ""),
			},
		},
	})
}

func TestAppBitriseYmlResource_Read_whenResponseTooLarge_reportsReadError(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				PreConfig:    func() { api.override(http.MethodGet, ymlPath, http.StatusOK, "application/x-yaml", oversizedJSON()) },
				RefreshState: true,
				ExpectError:  errorMatching("Could not read response: response body exceeds 1 MiB"),
			},
			{
				PreConfig: api.clearOverrides,
				Config:    appBitriseYmlConfig(api, "format_version: 13", ""),
			},
		},
	})
}

func TestAppBitriseYmlResource_Delete_keepsRemoteYml(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{Config: appBitriseYmlConfig(api, "format_version: 13", "")}},
	})

	var yml string
	api.update(func(a *fakeAPI) { yml = a.ymls["app-123"] })
	assert.Equal(t, "format_version: 13", yml)
}

func TestAppBitriseYmlResource_ImportState_withAppSlug_importsYml(t *testing.T) {
	api := newFakeAPI(t)
	api.addApp("app-123")

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: appBitriseYmlConfig(api, "format_version: 13", "")},
			{
				Config:            appBitriseYmlConfig(api, "format_version: 13", ""),
				ResourceName:      "bitrise_app_bitrise_yml.test",
				ImportState:       true,
				ImportStateId:     "app-123",
				ImportStateVerify: true,
			},
		},
	})
}

func TestYmlContent_extractsYamlFromEitherResponseFormat(t *testing.T) {
	cases := map[string]struct {
		raw  bitriseapi.RawResponse
		want string
	}{
		"raw yaml": {
			raw:  bitriseapi.RawResponse{ContentType: "application/x-yaml", Body: []byte("format_version: 13")},
			want: "format_version: 13",
		},
		"json envelope": {
			raw:  bitriseapi.RawResponse{ContentType: "application/json; charset=utf-8", Body: []byte(`{"app_config_datastore_yaml":"format_version: 13"}`)},
			want: "format_version: 13",
		},
		"json envelope without content type": {
			raw:  bitriseapi.RawResponse{ContentType: "text/plain", Body: []byte(`{"app_config_datastore_yaml":"format_version: 13"}`)},
			want: "format_version: 13",
		},
		"malformed json is kept as text": {
			raw:  bitriseapi.RawResponse{ContentType: "application/json", Body: []byte(`{not json`)},
			want: "{not json",
		},
		"empty body": {
			raw:  bitriseapi.RawResponse{ContentType: "text/plain"},
			want: "",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := ymlContent(context.Background(), tc.raw)

			assert.Equal(t, tc.want, got)
		})
	}
}
