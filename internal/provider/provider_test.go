package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runUnitTest runs tc against an in-process provider. It needs a Terraform CLI
// (PATH or TF_ACC_TERRAFORM_PATH) but neither TF_ACC, network nor credentials.
func runUnitTest(t *testing.T, tc resource.TestCase) {
	t.Helper()
	tc.IsUnitTest = true
	tc.ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"bitrise": providerserver.NewProtocol6WithError(New("test")()),
	}
	resource.ParallelTest(t, tc)
}

// errorMatching matches text in Terraform output, which may wrap it across lines.
func errorMatching(text string) *regexp.Regexp {
	words := strings.Fields(text)
	for i, word := range words {
		words[i] = regexp.QuoteMeta(word)
	}
	return regexp.MustCompile(strings.Join(words, `\s+`))
}

// checkRemovedFromState fails when name is still in the state.
func checkRemovedFromState(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if _, ok := s.RootModule().Resources[name]; ok {
			return fmt.Errorf("%s is still in state", name)
		}
		return nil
	}
}

func TestBitriseProvider_withToken_sendsItWithJSONContentType(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: api.providerConfig() + `data "bitrise_available_stacks" "test" {}`,
		}},
	})

	requests := api.received(http.MethodGet, "/v0.1/available-stacks")
	require.NotEmpty(t, requests)
	assert.Equal(t, testToken, requests[0].Header.Get("Authorization"))
	assert.Equal(t, "application/json", requests[0].Header.Get("Content-Type"))
}

func TestBitriseProvider_withWrongToken_reportsUnauthorized(t *testing.T) {
	api := newFakeAPI(t)

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`
provider "bitrise" {
  endpoint = %q
  token    = "wrong-token"
}

data "bitrise_available_stacks" "test" {}
`, api.server.URL),
			ExpectError: errorMatching("Failed to read available stacks: 401 Unauthorized"),
		}},
	})
}

func TestBitriseProvider_whenEndpointUnreachable_reportsSendError(t *testing.T) {
	api := newFakeAPI(t)
	config := api.providerConfig() + `data "bitrise_available_stacks" "test" {}`
	api.server.Close()

	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config:      config,
			ExpectError: errorMatching("Error sending HTTP request"),
		}},
	})
}

func TestBitriseProvider_withInvalidEndpoint_reportsRequestError(t *testing.T) {
	runUnitTest(t, resource.TestCase{
		Steps: []resource.TestStep{{
			Config: `
provider "bitrise" {
  endpoint = "http://bad host"
  token    = "test-token"
}

data "bitrise_available_stacks" "test" {}
`,
			ExpectError: errorMatching("Error creating HTTP request"),
		}},
	})
}

func TestBitriseProvider_Schema_tokenIsSensitive(t *testing.T) {
	var resp provider.SchemaResponse
	New("test")().Schema(context.Background(), provider.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["token"].(providerschema.StringAttribute)
	require.True(t, ok, "token must be a string attribute")
	assert.True(t, attr.Sensitive, "the API token must be redacted from output")
}
