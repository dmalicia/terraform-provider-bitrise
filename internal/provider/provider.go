// Package provider implements the Bitrise Terraform provider.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var _ provider.Provider = &BitriseProvider{}

// BitriseProvider is the Bitrise Terraform provider.
type BitriseProvider struct {
	version string
}

// BitriseProviderModel describes the provider configuration.
type BitriseProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

// Metadata returns the provider type name and version.
func (p *BitriseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bitrise"
	resp.Version = p.version
}

// Schema returns the provider configuration schema.
func (p *BitriseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
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

// Configure creates the API client shared by all resources and data sources.
func (p *BitriseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config BitriseProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()

	tflog.Debug(ctx, "MODULEDEBUG: Configuring Bitrise provider", map[string]interface{}{
		"endpoint": endpoint,
	})

	client := bitriseapi.New(endpoint, config.Token.ValueString())
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Bitrise provider configured successfully")
}

// Resources returns the resources the provider supports.
func (p *BitriseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
		NewAppSSHResource,
		NewAppFinishResource,
		NewAppSecretsResource,
		NewAppBitriseYmlResource,
		NewAppRolesResource,
		NewOrgSecretResource,
	}
}

// DataSources returns the data sources the provider supports.
func (p *BitriseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAppRolesDataSource,
		NewOrgGroupsDataSource,
		NewAvailableStacksDataSource,
	}
}

// New returns a factory for the provider at the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BitriseProvider{
			version: version,
		}
	}
}
