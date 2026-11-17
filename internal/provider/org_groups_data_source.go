package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var _ datasource.DataSourceWithConfigure = &OrgGroupsDataSource{}

// NewOrgGroupsDataSource returns a new bitrise_org_groups data source.
func NewOrgGroupsDataSource() datasource.DataSource {
	return &OrgGroupsDataSource{}
}

// OrgGroupsDataSource lists the groups of a Bitrise organization.
type OrgGroupsDataSource struct {
	dataSourceWithClient
}

// OrgGroupsDataSourceModel describes the bitrise_org_groups data source data model.
type OrgGroupsDataSourceModel struct {
	OrgSlug types.String `tfsdk:"org_slug"`
	ID      types.String `tfsdk:"id"`
	Groups  types.List   `tfsdk:"groups"`
}

// GroupAPIModel is the API representation of an organization group.
type GroupAPIModel struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// groupAttrTypes is the object type of a groups list element.
var groupAttrTypes = map[string]attr.Type{
	"slug": types.StringType,
	"name": types.StringType,
}

// Metadata returns the data source type name.
func (d *OrgGroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_groups"
}

// Schema returns the data source schema.
func (d *OrgGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

// Read lists the groups of the organization.
func (d *OrgGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OrgGroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgSlug := data.OrgSlug.ValueString()
	var groupsResp []GroupAPIModel
	err := d.client.Do(ctx, bitriseapi.Request{
		Op:     "list groups of organization " + orgSlug,
		Method: http.MethodGet,
		Path:   bitriseapi.OrgGroupsPath(orgSlug),
	}, &groupsResp)
	if err != nil {
		standardCallDiags("Failed to read organization groups: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	groupObjects := make([]attr.Value, 0, len(groupsResp))
	for _, group := range groupsResp {
		obj, diags := types.ObjectValue(groupAttrTypes, map[string]attr.Value{
			"slug": types.StringValue(group.Slug),
			"name": types.StringValue(group.Name),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		groupObjects = append(groupObjects, obj)
	}
	groupsList, diags := types.ListValue(types.ObjectType{AttrTypes: groupAttrTypes}, groupObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Groups = groupsList
	data.ID = types.StringValue(orgSlug)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
