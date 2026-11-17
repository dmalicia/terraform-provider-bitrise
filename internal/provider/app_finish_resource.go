package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var (
	_ resource.ResourceWithConfigure   = &AppFinishResource{}
	_ resource.ResourceWithImportState = &AppFinishResource{}
)

// AppFinishResource finishes the setup of a registered Bitrise app.
type AppFinishResource struct {
	resourceWithClient
}

// AppFinishResourceModel describes the bitrise_app_finish resource data model.
type AppFinishResourceModel struct {
	AppSlug          types.String      `tfsdk:"app_slug"`
	ProjectType      types.String      `tfsdk:"project_type"`
	StackID          types.String      `tfsdk:"stack_id"`
	Config           types.String      `tfsdk:"config"`
	Mode             types.String      `tfsdk:"mode"`
	Envs             map[string]string `tfsdk:"envs"`
	OrganizationSlug types.String      `tfsdk:"organization_slug"`
}

// PayloadFinish is the request body of the finish endpoint.
type PayloadFinish struct {
	ProjectType      string            `json:"project_type"`
	StackID          string            `json:"stack_id"`
	Config           string            `json:"config"`
	Mode             string            `json:"mode"`
	Envs             map[string]string `json:"envs"`
	OrganizationSlug string            `json:"organization_slug"`
}

// NewAppFinishResource returns a new bitrise_app_finish resource.
func NewAppFinishResource() resource.Resource {
	return &AppFinishResource{}
}

// Metadata returns the resource type name.
func (r *AppFinishResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_finish"
}

// Schema returns the resource schema.
func (r *AppFinishResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Bitrise App Finish resource",
		Attributes: map[string]schema.Attribute{
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "App Slug",
				Required:            true,
			},
			"project_type": schema.StringAttribute{
				MarkdownDescription: "The type of the project",
				Required:            true,
			},
			"stack_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the stack on which the build will run",
				Required:            true,
			},
			"config": schema.StringAttribute{
				MarkdownDescription: "The configuration for the app",
				Required:            true,
			},
			"mode": schema.StringAttribute{
				MarkdownDescription: "The mode of the app (e.g., manual)",
				Required:            true,
			},
			"envs": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Envs",
			},
			"organization_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the organization that owns the app",
				Required:            true,
			},
		},
	}
}

// Create finishes the app setup.
func (r *AppFinishResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppFinishResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.finish(ctx, data); err != nil {
		appCallDiags("Unable to create request: %s", "Unable to finish app setup: %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read keeps the resource in state while the app exists.
func (r *AppFinishResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppFinishResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !readAppExists(ctx, r.client, data.AppSlug.ValueString(), resp) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update finishes the app setup again with the planned settings.
func (r *AppFinishResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppFinishResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.finish(ctx, data); err != nil {
		appCallDiags("Unable to create request: %s", "Unable to update app finish: %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete only removes the resource from state; the API has nothing to undo.
func (r *AppFinishResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppFinishResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
}

// ImportState imports the resource by app slug.
func (r *AppFinishResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("app_slug"), req, resp)
}

func (r *AppFinishResource) finish(ctx context.Context, data AppFinishResourceModel) error {
	appSlug := data.AppSlug.ValueString()
	return r.client.Do(ctx, bitriseapi.Request{
		Op:     "finish app " + appSlug,
		Method: http.MethodPost,
		Path:   bitriseapi.AppFinishPath(appSlug),
		Body: PayloadFinish{
			ProjectType:      data.ProjectType.ValueString(),
			StackID:          data.StackID.ValueString(),
			Config:           data.Config.ValueString(),
			Mode:             data.Mode.ValueString(),
			Envs:             data.Envs,
			OrganizationSlug: data.OrganizationSlug.ValueString(),
		},
	}, &bitriseapi.RawResponse{})
}

// readAppExists checks that the app behind a dependent resource still exists.
// It removes the resource from state when the app is gone and reports whether
// the caller should keep the resource.
func readAppExists(ctx context.Context, client apiCaller, appSlug string, resp *resource.ReadResponse) bool {
	if appSlug == "" {
		tflog.Warn(ctx, "AppSlug is empty, skipping Read")
		return true
	}

	err := client.Do(ctx, bitriseapi.Request{
		Op:     "read app " + appSlug,
		Method: http.MethodGet,
		Path:   bitriseapi.AppPath(appSlug),
	}, nil)
	var statusErr *bitriseapi.StatusError
	switch {
	case err == nil:
		return true
	case errors.Is(err, bitriseapi.ErrNotFound):
		tflog.Info(ctx, "App not found, removing from state", map[string]interface{}{"app_slug": appSlug})
		resp.State.RemoveResource(ctx)
	case errors.As(err, &statusErr):
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read app, got status: %s", statusErr.Status()))
	default:
		appCallDiags("Unable to create request: %s", "Unable to read app: %s").add(&resp.Diagnostics, err)
	}
	return false
}
