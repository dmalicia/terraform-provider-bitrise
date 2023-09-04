package provider

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &BitriseProvider{}

type ClientProvider interface {
	GetClient() (*http.Client, error)
}

type BitriseProvider struct {
	version       string
	endpoint      string
	token         string
	clientCreator func(endpoint, token string) *http.Client
}

type BitriseProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *BitriseProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bitrise"
	resp.Version = p.version
}

func (p *BitriseProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The endpoint of the Bitrise API",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The API token for authenticating with the Bitrise API",
				Optional:            true,
			},
		},
	}
}

// ... (other code)

func (p *BitriseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config BitriseProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	//p.endpoint = config.Endpoint.String()
	p.token = config.Token.String()

	// Remove any existing "https://" prefix from the endpoint value
	p.endpoint = strings.TrimPrefix(config.Endpoint.String(), "https://")
	dois := strings.TrimPrefix(config.Endpoint.String(), "https://")
	cleanedDois := strings.Trim(dois, "\"")

	// Construct the URL
	url := "https://" + p.endpoint

	// Print the constructed URL for debugging purposes
	log.Printf("Constructed URL: %s", url)
	log.Printf("Constructed URL dois : %s", dois)
	log.Printf("Constructed URL doiscleaned : %s", cleanedDois)

	log.Printf("Configuring Bitrise provider with Endpoint: %s", p.endpoint)
	log.Printf("Configuring Bitrise provider with Token: %s", p.token)

	p.clientCreator = func(endpoint, token string) *http.Client {
		// Remove any "https://" prefix and quotes from the endpoint value
		cleanedEndpoint := strings.TrimPrefix(endpoint, "https://")
		cleanedEndpoint = strings.Trim(cleanedEndpoint, "\"")

		// Remove quotes from the token
		cleanedToken := strings.Trim(token, "\"")

		return &http.Client{
			Transport: &authenticatedTransport{
				token:    cleanedToken,
				base:     http.DefaultTransport,
				headers:  map[string]string{"Content-Type": "application/json"},
				endpoint: cleanedEndpoint,
			},
		}
	}

	resp.DataSourceData = p.clientCreator

	// Instead of resp.ResourceData2, set these values directly in the provider instance
	p.endpoint = config.Endpoint.String()
	p.token = config.Token.String()

	log.Printf("Configured Endpoint: %s", p.endpoint)

}

func (p *BitriseProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource {
			return NewAppResource(p.clientCreator, p.endpoint, p.token)
		},
		func() resource.Resource {
			return NewAppSSHResource(p.clientCreator, p.endpoint, p.token) // Return the custom resource type instance
		},
		func() resource.Resource {
			return NewAppFinishResource(p.clientCreator, p.endpoint, p.token) // Return the custom resource type instance
		},
	}
}

type authenticatedTransport struct {
	token    string
	endpoint string
	base     http.RoundTripper
	headers  map[string]string
}

func (t *authenticatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.token)
	return t.base.RoundTrip(req)
}

func (p *BitriseProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BitriseProvider{
			version: version,
		}
	}
}
