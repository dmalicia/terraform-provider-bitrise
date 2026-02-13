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

var _ datasource.DataSource = &AppRolesDataSource{}

func NewAppRolesDataSource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AppRolesDataSource {
	return &AppRolesDataSource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type AppRolesDataSource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

type AppRolesDataSourceModel struct {
	AppSlug             types.String   `tfsdk:"app_slug"`
	RoleName            types.String   `tfsdk:"role_name"`
	ID                  types.String   `tfsdk:"id"`
	Groups              []types.String `tfsdk:"groups"`
}

type GroupRolesResponse struct {
	Groups []string `json:"groups"`
}

func (d *AppRolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_roles"
}

func (d *AppRolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

func (d *AppRolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AppRolesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()

	tflog.Debug(ctx, "Reading Bitrise app role groups", map[string]interface{}{
		"app_slug":  appSlug,
		"role_name": roleName,
	})

	client := d.clientCreator(d.endpoint, d.token)
	url := fmt.Sprintf("%s/v0.1/apps/%s/roles/%s", d.endpoint, appSlug, roleName)

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
		tflog.Error(ctx, "Failed to read role groups", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to read role groups: %s - %s", httpResp.Status, string(responseBody)),
		)
		return
	}

	var rolesResp GroupRolesResponse
	if err := json.Unmarshal(responseBody, &rolesResp); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Convert API response to terraform model
	groups := make([]types.String, 0, len(rolesResp.Groups))
	for _, group := range rolesResp.Groups {
		groups = append(groups, types.StringValue(group))
	}

	data.Groups = groups
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", appSlug, roleName))

	tflog.Info(ctx, "Successfully read Bitrise app role groups", map[string]interface{}{
		"app_slug":    appSlug,
		"role_name":   roleName,
		"group_count": len(groups),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
