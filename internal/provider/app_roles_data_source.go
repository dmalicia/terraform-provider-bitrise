package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var _ datasource.DataSourceWithConfigure = &AppRolesDataSource{}

// NewAppRolesDataSource returns a new bitrise_app_roles data source.
func NewAppRolesDataSource() datasource.DataSource {
	return &AppRolesDataSource{}
}

// AppRolesDataSource reads the groups assigned to a role of a Bitrise app.
type AppRolesDataSource struct {
	dataSourceWithClient
}

// AppRolesDataSourceModel describes the bitrise_app_roles data source data model.
type AppRolesDataSourceModel struct {
	AppSlug  types.String   `tfsdk:"app_slug"`
	RoleName types.String   `tfsdk:"role_name"`
	ID       types.String   `tfsdk:"id"`
	Groups   []types.String `tfsdk:"groups"`
}

// GroupRolesResponse is the response body listing the groups of a role.
type GroupRolesResponse struct {
	Groups []string `json:"groups"`
}

// Metadata returns the data source type name.
func (d *AppRolesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_roles"
}

// Schema returns the data source schema.
func (d *AppRolesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of groups assigned to a specific role type for a Bitrise application.",
		Attributes: map[string]schema.Attribute{
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the Bitrise app",
				Required:            true,
			},
			"role_name": schema.StringAttribute{
				MarkdownDescription: "The role type to query. Supported values: admin, manager (developer), member (tester/qa), platform_engineer",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Data source identifier (app_slug/role_name)",
				Computed:            true,
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "List of group slugs assigned to this role",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

// Read reads the groups of the role.
func (d *AppRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AppRolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()
	var rolesResp GroupRolesResponse
	err := d.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("read groups of role %s of app %s", roleName, appSlug),
		Method: http.MethodGet,
		Path:   bitriseapi.AppRolePath(appSlug, roleName),
	}, &rolesResp)
	if err != nil {
		standardCallDiags("Failed to read role groups: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	groups := make([]types.String, 0, len(rolesResp.Groups))
	for _, group := range rolesResp.Groups {
		groups = append(groups, types.StringValue(group))
	}
	data.Groups = groups
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", appSlug, roleName))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
