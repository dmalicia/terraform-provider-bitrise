package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &OrgGroupsDataSource{}

func NewOrgGroupsDataSource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *OrgGroupsDataSource {
	return &OrgGroupsDataSource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type OrgGroupsDataSource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

type OrgGroupsDataSourceModel struct {
	OrgSlug types.String `tfsdk:"org_slug"`
	ID      types.String `tfsdk:"id"`
	Groups  types.List   `tfsdk:"groups"`
}

type GroupAPIModel struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (d *OrgGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_groups"
}

func (d *OrgGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of groups for a Bitrise organization.",
		Attributes: map[string]schema.Attribute{
			"org_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the Bitrise organization",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier (org_slug)",
				Computed:            true,
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "List of groups in the organization",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"slug": types.StringType,
						"name": types.StringType,
					},
				},
			},
		},
	}
}

func (d *OrgGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OrgGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrgGroupsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgSlug := data.OrgSlug.ValueString()

	tflog.Debug(ctx, "Reading Bitrise organization groups", map[string]interface{}{
		"org_slug": orgSlug,
	})

	client := d.clientCreator(d.endpoint, d.token)
	url := fmt.Sprintf("%s/v0.1/organizations/%s/groups", d.endpoint, orgSlug)

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
		tflog.Error(ctx, "Failed to read organization groups", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to read organization groups: %s - %s", httpResp.Status, string(responseBody)),
		)
		return
	}

	var groupsResp []GroupAPIModel
	if err := json.Unmarshal(responseBody, &groupsResp); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Convert API response to terraform model
	groupObjects := make([]attr.Value, 0, len(groupsResp))
	for _, group := range groupsResp {
		obj, diags := types.ObjectValue(
			map[string]attr.Type{
				"slug": types.StringType,
				"name": types.StringType,
			},
			map[string]attr.Value{
				"slug": types.StringValue(group.Slug),
				"name": types.StringValue(group.Name),
			},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupObjects = append(groupObjects, obj)
	}

	groupsList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"slug": types.StringType,
				"name": types.StringType,
			},
		},
		groupObjects,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Groups = groupsList
	data.ID = types.StringValue(orgSlug)

	tflog.Info(ctx, "Successfully read Bitrise organization groups", map[string]interface{}{
		"org_slug":    orgSlug,
		"group_count": len(groupObjects),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
