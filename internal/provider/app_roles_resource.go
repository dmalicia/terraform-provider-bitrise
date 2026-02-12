package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AppRolesResource{}
var _ resource.ResourceWithImportState = &AppRolesResource{}

func NewAppRolesResource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AppRolesResource {
	return &AppRolesResource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type AppRolesResource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

type AppRolesResourceModel struct {
	AppSlug  types.String   `tfsdk:"app_slug"`
	RoleName types.String   `tfsdk:"role_name"`
	ID       types.String   `tfsdk:"id"`
	Groups   []types.String `tfsdk:"groups"`
}

type GroupRolesUpdateRequest struct {
	Groups []string `json:"groups"`
}

type GroupRolesReadResponse struct {
	Groups []string `json:"groups"`
}

func (r *AppRolesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_roles"
}

func (r *AppRolesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the groups assigned to a specific role type for a Bitrise application. This resource replaces all groups for the specified role with the provided list.",
		Attributes: map[string]schema.Attribute{
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the Bitrise app",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				MarkdownDescription: "The role type to manage. Supported values: admin, manager (developer), member (tester/qa), platform_engineer",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Resource identifier (app_slug/role_name)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "List of group slugs to assign to this role. This replaces all existing groups.",
				Required:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *AppRolesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clientCreator, ok := req.ProviderData.(func(endpoint, token string) *http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected func(endpoint, token string) *http.Client, got: %T", req.ProviderData),
		)
		return
	}

	r.clientCreator = clientCreator
}

func (r *AppRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()

	tflog.Debug(ctx, "Creating/Replacing Bitrise app role groups", map[string]interface{}{
		"app_slug":    appSlug,
		"role_name":   roleName,
		"group_count": len(data.Groups),
	})

	// Convert terraform model to API request
	groups := make([]string, 0, len(data.Groups))
	for _, group := range data.Groups {
		groups = append(groups, group.ValueString())
	}

	rolesReq := GroupRolesUpdateRequest{
		Groups: groups,
	}

	if err := r.updateRoleGroups(ctx, appSlug, roleName, rolesReq, &resp.Diagnostics); err != nil {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", appSlug, roleName))

	tflog.Info(ctx, "Successfully created/replaced Bitrise app role groups", map[string]interface{}{
		"app_slug":  appSlug,
		"role_name": roleName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()

	tflog.Debug(ctx, "Reading Bitrise app role groups", map[string]interface{}{
		"app_slug":  appSlug,
		"role_name": roleName,
	})

	client := r.clientCreator(r.endpoint, r.token)
	url := fmt.Sprintf("%s/v0.1/apps/%s/roles/%s", r.endpoint, appSlug, roleName)

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

	if httpResp.StatusCode == http.StatusNotFound {
		tflog.Info(ctx, "App or role not found, removing from state", map[string]interface{}{
			"app_slug":  appSlug,
			"role_name": roleName,
		})
		resp.State.RemoveResource(ctx)
		return
	}

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

	var rolesResp GroupRolesReadResponse
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()

	tflog.Debug(ctx, "Updating Bitrise app role groups", map[string]interface{}{
		"app_slug":    appSlug,
		"role_name":   roleName,
		"group_count": len(data.Groups),
	})

	// Convert terraform model to API request
	groups := make([]string, 0, len(data.Groups))
	for _, group := range data.Groups {
		groups = append(groups, group.ValueString())
	}

	rolesReq := GroupRolesUpdateRequest{
		Groups: groups,
	}

	if err := r.updateRoleGroups(ctx, appSlug, roleName, rolesReq, &resp.Diagnostics); err != nil {
		return
	}

	tflog.Info(ctx, "Successfully updated Bitrise app role groups", map[string]interface{}{
		"app_slug":  appSlug,
		"role_name": roleName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()

	tflog.Debug(ctx, "Deleting Bitrise app role groups", map[string]interface{}{
		"app_slug":  appSlug,
		"role_name": roleName,
	})

	// To "delete" role groups, we set it to an empty list
	rolesReq := GroupRolesUpdateRequest{
		Groups: []string{},
	}

	if err := r.updateRoleGroups(ctx, appSlug, roleName, rolesReq, &resp.Diagnostics); err != nil {
		return
	}

	tflog.Info(ctx, "Successfully deleted Bitrise app role groups (set to empty list)", map[string]interface{}{
		"app_slug":  appSlug,
		"role_name": roleName,
	})
}

func (r *AppRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID should be in the format: app_slug/role_name
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in the format 'app_slug/role_name', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// updateRoleGroups is a helper function to update role groups via the Bitrise API
func (r *AppRolesResource) updateRoleGroups(ctx context.Context, appSlug, roleName string, rolesReq GroupRolesUpdateRequest, diags *diag.Diagnostics) error {
	client := r.clientCreator(r.endpoint, r.token)
	url := fmt.Sprintf("%s/v0.1/apps/%s/roles/%s", r.endpoint, appSlug, roleName)

	payloadJSON, err := json.Marshal(rolesReq)
	if err != nil {
		diags.AddError("Error marshaling request", err.Error())
		return err
	}

	tflog.Debug(ctx, "Sending PUT request to update role groups", map[string]interface{}{
		"url":     url,
		"payload": string(payloadJSON),
	})

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(payloadJSON)))
	if err != nil {
		diags.AddError("Error creating HTTP request", err.Error())
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := client.Do(httpReq)
	if err != nil {
		diags.AddError("Error sending HTTP request", err.Error())
		return err
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		diags.AddError("Error reading response body", err.Error())
		return err
	}

	if httpResp.StatusCode != http.StatusOK {
		tflog.Error(ctx, "Failed to update role groups", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		diags.AddError(
			"API Error",
			fmt.Sprintf("Failed to update role groups: %s - %s", httpResp.Status, string(responseBody)),
		)
		return fmt.Errorf("API error: %s", httpResp.Status)
	}

	return nil
}
