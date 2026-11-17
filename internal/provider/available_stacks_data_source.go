package provider

import (
	"context"
	"maps"
	"net/http"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var _ datasource.DataSourceWithConfigure = &AvailableStacksDataSource{}

// NewAvailableStacksDataSource returns a new bitrise_available_stacks data source.
func NewAvailableStacksDataSource() datasource.DataSource {
	return &AvailableStacksDataSource{}
}

// AvailableStacksDataSource lists the stacks builds can run on.
type AvailableStacksDataSource struct {
	dataSourceWithClient
}

// AvailableStacksDataSourceModel describes the bitrise_available_stacks data source data model.
type AvailableStacksDataSourceModel struct {
	ID        types.String   `tfsdk:"id"`
	StackKeys []types.String `tfsdk:"stack_keys"`
}

// Metadata returns the data source type name.
func (d *AvailableStacksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_stacks"
}

// Schema returns the data source schema.
func (d *AvailableStacksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of all available stacks from Bitrise.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier",
				Computed:            true,
			},
			"stack_keys": schema.ListAttribute{
				MarkdownDescription: "List of all available stack IDs",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Read lists the available stack IDs.
func (d *AvailableStacksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AvailableStacksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stacksMap map[string]interface{}
	err := d.client.Do(ctx, bitriseapi.Request{
		Op:     "list available stacks",
		Method: http.MethodGet,
		Path:   bitriseapi.AvailableStacksPath,
	}, &stacksMap)
	if err != nil {
		standardCallDiags("Failed to read available stacks: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	// Map iteration order is random; sort so the list is stable between reads.
	stackKeys := make([]types.String, 0, len(stacksMap))
	for _, key := range slices.Sorted(maps.Keys(stacksMap)) {
		stackKeys = append(stackKeys, types.StringValue(key))
	}
	data.ID = types.StringValue("available-stacks")
	data.StackKeys = stackKeys
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
