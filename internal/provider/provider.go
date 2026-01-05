package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

	// Get the actual string values using ValueString()
	endpoint := config.Endpoint.ValueString()
	token := config.Token.ValueString()

	// Store the endpoint as-is (with https:// if provided)
	p.endpoint = endpoint
	p.token = token

	// Log configuration for debugging
	tflog.Debug(ctx, "MODULEDEBUG: Configuring Bitrise provider", map[string]interface{}{
		"endpoint": p.endpoint,
	})

	p.clientCreator = func(endpoint, token string) *http.Client {
		return &http.Client{
			Transport: &authenticatedTransport{
				token:    token,
				base:     http.DefaultTransport,
				headers:  map[string]string{"Content-Type": "application/json"},
				endpoint: endpoint,
			},
		}
	}

	resp.DataSourceData = p.clientCreator
	resp.ResourceData = p.clientCreator

	tflog.Info(ctx, "Bitrise provider configured successfully")
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
	// Apply all headers from the headers map
	for key, value := range t.headers {
		req.Header.Set(key, value)
	}
	return t.base.RoundTrip(req)
}

func (p *BitriseProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BitriseProvider{
			version: version,
		}
	}
}
