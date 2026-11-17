package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var (
	_ resource.ResourceWithConfigure   = &AppRolesResource{}
	_ resource.ResourceWithImportState = &AppRolesResource{}
)

// NewAppRolesResource returns a new bitrise_app_roles resource.
func NewAppRolesResource() resource.Resource {
	return &AppRolesResource{}
}

// AppRolesResource manages the groups assigned to a role of a Bitrise app.
type AppRolesResource struct {
	resourceWithClient
}

// AppRolesResourceModel describes the bitrise_app_roles resource data model.
type AppRolesResourceModel struct {
	AppSlug  types.String   `tfsdk:"app_slug"`
	RoleName types.String   `tfsdk:"role_name"`
	ID       types.String   `tfsdk:"id"`
	Groups   []types.String `tfsdk:"groups"`
}

// GroupRolesUpdateRequest is the request body that replaces the groups of a role.
type GroupRolesUpdateRequest struct {
	Groups []string `json:"groups"`
}

// GroupRolesReadResponse is the response body listing the groups of a role.
type GroupRolesReadResponse struct {
	Groups []string `json:"groups"`
}

// Metadata returns the resource type name.
func (r *AppRolesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_roles"
}

// Schema returns the resource schema.
func (r *AppRolesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

// Create replaces the groups of the role.
func (r *AppRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()
	if err := r.putRoleGroups(ctx, appSlug, roleName, groupNames(data.Groups)); err != nil {
		standardCallDiags("Failed to update role groups: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", appSlug, roleName))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the groups of the role.
func (r *AppRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	roleName := data.RoleName.ValueString()
	var rolesResp GroupRolesReadResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("read groups of role %s of app %s", roleName, appSlug),
		Method: http.MethodGet,
		Path:   bitriseapi.AppRolePath(appSlug, roleName),
	}, &rolesResp)
	if errors.Is(err, bitriseapi.ErrNotFound) {
		tflog.Info(ctx, "App or role not found, removing from state", map[string]interface{}{
			"app_slug":  appSlug,
			"role_name": roleName,
		})
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		standardCallDiags("Failed to read role groups: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	groups := make([]types.String, 0, len(rolesResp.Groups))
	for _, group := range rolesResp.Groups {
		groups = append(groups, types.StringValue(group))
	}
	data.Groups = groups
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update replaces the groups of the role with the planned ones.
func (r *AppRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.putRoleGroups(ctx, data.AppSlug.ValueString(), data.RoleName.ValueString(), groupNames(data.Groups)); err != nil {
		standardCallDiags("Failed to update role groups: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes every group from the role.
func (r *AppRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppRolesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.putRoleGroups(ctx, data.AppSlug.ValueString(), data.RoleName.ValueString(), []string{}); err != nil {
		standardCallDiags("Failed to update role groups: %s - %s").add(&resp.Diagnostics, err)
	}
}

// ImportState imports the resource by "app_slug/role_name".
func (r *AppRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *AppRolesResource) putRoleGroups(ctx context.Context, appSlug, roleName string, groups []string) error {
	return r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("update groups of role %s of app %s", roleName, appSlug),
		Method: http.MethodPut,
		Path:   bitriseapi.AppRolePath(appSlug, roleName),
		Body:   GroupRolesUpdateRequest{Groups: groups},
	}, &bitriseapi.RawResponse{})
}

func groupNames(groups []types.String) []string {
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.ValueString())
	}
	return names
}
