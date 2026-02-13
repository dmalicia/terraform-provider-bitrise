package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &AvailableStacksDataSource{}

func NewAvailableStacksDataSource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AvailableStacksDataSource {
	return &AvailableStacksDataSource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type AvailableStacksDataSource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

type AvailableStacksDataSourceModel struct {
	ID         types.String   `tfsdk:"id"`
	StackKeys  []types.String `tfsdk:"stack_keys"`
}

func (d *AvailableStacksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_available_stacks"
}

func (d *AvailableStacksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

func (d *AvailableStacksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clientCreator, ok := req.ProviderData.(func(endpoint, token string) *http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected func(endpoint, token string) *http.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.clientCreator = clientCreator
}

func (d *AvailableStacksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AvailableStacksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Bitrise available stacks")

	client := d.clientCreator(d.endpoint, d.token)
	url := fmt.Sprintf("%s/v0.1/available-stacks", d.endpoint)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating HTTP request", err.Error())
		return
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error sending HTTP request", err.Error())
		return
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Error reading response body", err.Error())
		return
	}

	if httpResp.StatusCode != http.StatusOK {
		tflog.Error(ctx, "Failed to read available stacks", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to read available stacks: %s - %s", httpResp.Status, string(responseBody)),
		)
		return
	}

	// Parse the response as a map to get the keys
	var stacksMap map[string]interface{}
	if err := json.Unmarshal(responseBody, &stacksMap); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Extract the keys from the map
	stackKeys := make([]types.String, 0, len(stacksMap))
	for key := range stacksMap {
		stackKeys = append(stackKeys, types.StringValue(key))
	}

	// Set the data
	data.ID = types.StringValue("available-stacks")
	data.StackKeys = stackKeys

	tflog.Debug(ctx, "Successfully read available stacks", map[string]interface{}{
		"count": len(stackKeys),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
