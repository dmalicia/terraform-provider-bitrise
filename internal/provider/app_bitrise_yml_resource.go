package provider

import (
	"context"
	"encoding/json"
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
	_ resource.ResourceWithConfigure   = &AppBitriseYmlResource{}
	_ resource.ResourceWithImportState = &AppBitriseYmlResource{}
)

// NewAppBitriseYmlResource returns a new bitrise_app_bitrise_yml resource.
func NewAppBitriseYmlResource() resource.Resource {
	return &AppBitriseYmlResource{}
}

// AppBitriseYmlResource manages the bitrise.yml of a Bitrise app.
type AppBitriseYmlResource struct {
	resourceWithClient
}

// AppBitriseYmlResourceModel describes the bitrise_app_bitrise_yml resource data model.
type AppBitriseYmlResourceModel struct {
	AppSlug            types.String `tfsdk:"app_slug"`
	YmlContent         types.String `tfsdk:"yml_content"`
	UpdateOnCreateOnly types.Bool   `tfsdk:"update_on_create_only"`
	ID                 types.String `tfsdk:"id"`
}

// BitriseYmlRequest is the request body that uploads a bitrise.yml.
type BitriseYmlRequest struct {
	AppConfigDatastoreYaml string `json:"app_config_datastore_yaml"`
}

// BitriseYmlResponse is the JSON form of the bitrise.yml read response.
type BitriseYmlResponse struct {
	AppConfigDatastoreYaml string `json:"app_config_datastore_yaml"`
}

// ignoreChangesIfUpdateOnCreateOnly is a custom plan modifier that prevents updates when update_on_create_only is true
type ignoreChangesIfUpdateOnCreateOnly struct{}

// Description returns a plain text description of the modifier.
func (m ignoreChangesIfUpdateOnCreateOnly) Description(_ context.Context) string {
	return "Ignores changes to yml_content when update_on_create_only is true"
}

// MarkdownDescription returns a markdown description of the modifier.
func (m ignoreChangesIfUpdateOnCreateOnly) MarkdownDescription(_ context.Context) string {
	return "Ignores changes to yml_content when update_on_create_only is true"
}

// PlanModifyString keeps the prior yml_content when update_on_create_only is true.
func (m ignoreChangesIfUpdateOnCreateOnly) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If this is a create operation, do nothing
	if req.State.Raw.IsNull() {
		return
	}

	// Get the update_on_create_only flag from the plan
	var updateOnCreateOnly types.Bool
	diag := req.Plan.GetAttribute(ctx, path.Root("update_on_create_only"), &updateOnCreateOnly)
	if diag.HasError() {
		return
	}

	// If update_on_create_only is true, use the state value instead of the config value
	if !updateOnCreateOnly.IsNull() && updateOnCreateOnly.ValueBool() {
		resp.PlanValue = req.StateValue
	}
}

// Metadata returns the resource type name.
func (r *AppBitriseYmlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_bitrise_yml"
}

// Schema returns the resource schema.
func (r *AppBitriseYmlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the bitrise.yml configuration file for a Bitrise application. This resource allows you to create and update the workflow configuration.",
		Attributes: map[string]schema.Attribute{
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the Bitrise app",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"yml_content": schema.StringAttribute{
				MarkdownDescription: "The content of the bitrise.yml file. This should be a valid YAML configuration for Bitrise workflows.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					ignoreChangesIfUpdateOnCreateOnly{},
				},
			},
			"update_on_create_only": schema.BoolAttribute{
				MarkdownDescription: "If set to true, the bitrise.yml will only be applied during resource creation. Subsequent updates will be ignored. Default is false.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the resource (app_slug)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create uploads the bitrise.yml.
func (r *AppBitriseYmlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upload(ctx, data); err != nil {
		bitriseYmlCallDiags().add(&resp.Diagnostics, err)
		return
	}

	data.ID = data.AppSlug
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes yml_content from the API.
func (r *AppBitriseYmlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	var raw bitriseapi.RawResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     "read bitrise.yml of app " + appSlug,
		Method: http.MethodGet,
		Path:   bitriseapi.AppBitriseYMLPath(appSlug),
	}, &raw)
	var callErr *bitriseapi.CallError
	switch {
	case err == nil:
	case errors.As(err, &callErr):
		bitriseYmlCallDiags().add(&resp.Diagnostics, err)
		return
	case errors.Is(err, bitriseapi.ErrNotFound):
		tflog.Warn(ctx, "Bitrise.yml not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	default:
		bitriseYmlCallDiags().add(&resp.Diagnostics, err)
		return
	}

	data.YmlContent = types.StringValue(ymlContent(ctx, raw))
	data.ID = data.AppSlug
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update uploads the planned bitrise.yml unless update_on_create_only is set.
func (r *AppBitriseYmlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.UpdateOnCreateOnly.IsNull() && data.UpdateOnCreateOnly.ValueBool() {
		tflog.Info(ctx, "Skipping bitrise.yml update (update_on_create_only is true)", map[string]interface{}{
			"app_slug": data.AppSlug.ValueString(),
		})
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	if err := r.upload(ctx, data); err != nil {
		bitriseYmlCallDiags().add(&resp.Diagnostics, err)
		return
	}

	data.ID = data.AppSlug
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete only removes the resource from state; the API cannot delete a bitrise.yml.
func (r *AppBitriseYmlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
}

// ImportState imports the resource by app slug.
func (r *AppBitriseYmlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("app_slug"), req, resp)
}

func (r *AppBitriseYmlResource) upload(ctx context.Context, data AppBitriseYmlResourceModel) error {
	appSlug := data.AppSlug.ValueString()
	return r.client.Do(ctx, bitriseapi.Request{
		Op:     "upload bitrise.yml of app " + appSlug,
		Method: http.MethodPost,
		Path:   bitriseapi.AppBitriseYMLPath(appSlug),
		Body:   BitriseYmlRequest{AppConfigDatastoreYaml: data.YmlContent.ValueString()},
		Accept: []int{http.StatusOK, http.StatusCreated},
	}, &bitriseapi.RawResponse{})
}

// ymlContent extracts the YAML from a read response, which is either raw YAML or a JSON envelope.
func ymlContent(ctx context.Context, raw bitriseapi.RawResponse) string {
	if !strings.Contains(raw.ContentType, "application/json") && (len(raw.Body) == 0 || raw.Body[0] != '{') {
		return string(raw.Body)
	}
	var yml BitriseYmlResponse
	if err := json.Unmarshal(raw.Body, &yml); err != nil {
		tflog.Debug(ctx, "Failed to parse as JSON, treating as plain text", map[string]interface{}{"error": err.Error()})
		return string(raw.Body)
	}
	return yml.AppConfigDatastoreYaml
}

func bitriseYmlCallDiags() callDiags {
	return callDiags{
		encode: diagText{"Error marshaling request payload", "Could not marshal payload: %s"},
		build:  diagText{"Error creating HTTP request", "Could not create request: %s"},
		send:   diagText{"Error sending HTTP request", "Could not send request: %s"},
		read:   diagText{"Error reading response body", "Could not read response: %s"},
		status: func(e *bitriseapi.StatusError) (string, string) {
			return "API Request Error", fmt.Sprintf("Request failed with status %d: %s", e.Code, e.Body)
		},
	}
}
