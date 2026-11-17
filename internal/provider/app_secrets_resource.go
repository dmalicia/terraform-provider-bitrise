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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var (
	_ resource.ResourceWithConfigure   = &AppSecretsResource{}
	_ resource.ResourceWithImportState = &AppSecretsResource{}
)

// NewAppSecretsResource returns a new bitrise_app_secret resource.
func NewAppSecretsResource() resource.Resource {
	return &AppSecretsResource{}
}

// AppSecretsResource manages a secret of a Bitrise app.
type AppSecretsResource struct {
	resourceWithClient
}

// AppSecretsResourceModel describes the app secret resource data model.
type AppSecretsResourceModel struct {
	AppSlug                  types.String `tfsdk:"app_slug"`
	Name                     types.String `tfsdk:"name"`
	Value                    types.String `tfsdk:"value"`
	IsProtected              types.Bool   `tfsdk:"is_protected"`
	IsExposedForPullRequests types.Bool   `tfsdk:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       types.Bool   `tfsdk:"expand_in_step_inputs"`
	ID                       types.String `tfsdk:"id"`
}

// SecretCreateRequest is the request body that creates an app secret.
// Booleans are always sent so that false does not fall back to the API default.
type SecretCreateRequest struct {
	Name                     string `json:"name"`
	Value                    string `json:"value"`
	IsProtected              bool   `json:"is_protected"`
	IsExposedForPullRequests bool   `json:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       bool   `json:"expand_in_step_inputs"`
}

// SecretUpdateRequest is the request body that updates an app secret.
type SecretUpdateRequest struct {
	Value                    string `json:"value,omitempty"`
	IsProtected              *bool  `json:"is_protected,omitempty"`
	IsExposedForPullRequests *bool  `json:"is_exposed_for_pull_requests,omitempty"`
	ExpandInStepInputs       *bool  `json:"expand_in_step_inputs,omitempty"`
}

// SecretResponse is the API representation of an app secret.
type SecretResponse struct {
	Name                     string `json:"name"`
	Value                    string `json:"value,omitempty"`
	IsProtected              bool   `json:"is_protected"`
	IsExposedForPullRequests bool   `json:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       bool   `json:"expand_in_step_inputs"`
	ID                       string `json:"id"`
}

// Metadata returns the resource type name.
func (r *AppSecretsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_secret"
}

// Schema returns the resource schema.
func (r *AppSecretsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages secrets for a Bitrise application. Secrets are environment variables that are securely stored and can be used in your build workflows.",
		Attributes: map[string]schema.Attribute{
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the Bitrise app",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name (key) of the secret",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the secret",
				Required:            true,
				Sensitive:           true,
			},
			"is_protected": schema.BoolAttribute{
				MarkdownDescription: "If true, the secret value cannot be retrieved via the API. Default: false",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_exposed_for_pull_requests": schema.BoolAttribute{
				MarkdownDescription: "If true, the secret will be available for pull request builds. Default: false",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"expand_in_step_inputs": schema.BoolAttribute{
				MarkdownDescription: "If true, variable expansion will be enabled for this secret in step inputs. Default: true",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the secret",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the secret.
func (r *AppSecretsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	name := data.Name.ValueString()
	var secretResp SecretResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("create secret %s of app %s", name, appSlug),
		Method: http.MethodPost,
		Path:   bitriseapi.AppSecretsPath(appSlug),
		Body: SecretCreateRequest{
			Name:                     name,
			Value:                    data.Value.ValueString(),
			IsProtected:              data.IsProtected.ValueBool(),
			IsExposedForPullRequests: data.IsExposedForPullRequests.ValueBool(),
			ExpandInStepInputs:       data.ExpandInStepInputs.ValueBool(),
		},
		Accept: []int{http.StatusCreated},
	}, &secretResp)
	if err != nil {
		standardCallDiags("Failed to create secret: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", appSlug, name))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the secret settings; the value is only returned for unprotected secrets.
func (r *AppSecretsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	name := data.Name.ValueString()
	var secretResp SecretResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("read secret %s of app %s", name, appSlug),
		Method: http.MethodGet,
		Path:   bitriseapi.AppSecretPath(appSlug, name),
	}, &secretResp)
	if errors.Is(err, bitriseapi.ErrNotFound) {
		tflog.Info(ctx, "Secret not found, removing from state", map[string]interface{}{
			"app_slug": appSlug,
			"name":     name,
		})
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		standardCallDiags("Failed to read secret: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	data.IsProtected = types.BoolValue(secretResp.IsProtected)
	data.IsExposedForPullRequests = types.BoolValue(secretResp.IsExposedForPullRequests)
	data.ExpandInStepInputs = types.BoolValue(secretResp.ExpandInStepInputs)
	if !secretResp.IsProtected && secretResp.Value != "" {
		data.Value = types.StringValue(secretResp.Value)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the secret value and settings.
func (r *AppSecretsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	name := data.Name.ValueString()
	isProtected := data.IsProtected.ValueBool()
	isExposed := data.IsExposedForPullRequests.ValueBool()
	expandInputs := data.ExpandInStepInputs.ValueBool()
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("update secret %s of app %s", name, appSlug),
		Method: http.MethodPatch,
		Path:   bitriseapi.AppSecretPath(appSlug, name),
		Body: SecretUpdateRequest{
			Value:                    data.Value.ValueString(),
			IsProtected:              &isProtected,
			IsExposedForPullRequests: &isExposed,
			ExpandInStepInputs:       &expandInputs,
		},
	}, &bitriseapi.RawResponse{})
	if err != nil {
		standardCallDiags("Failed to update secret: %s - %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the secret; a secret that is already gone is not an error.
func (r *AppSecretsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	name := data.Name.ValueString()
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("delete secret %s of app %s", name, appSlug),
		Method: http.MethodDelete,
		Path:   bitriseapi.AppSecretPath(appSlug, name),
		Accept: []int{http.StatusNoContent},
	}, nil)
	var statusErr *bitriseapi.StatusError
	switch {
	case err == nil:
	case errors.Is(err, bitriseapi.ErrNotFound):
		tflog.Info(ctx, "Secret already deleted")
	case errors.As(err, &statusErr):
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to delete secret: %s - %s", statusErr.Status(), statusErr.Body))
	default:
		standardCallDiags("Failed to delete secret: %s - %s").add(&resp.Diagnostics, err)
	}
}

// ImportState imports the resource by "app_slug/secret_name".
func (r *AppSecretsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in the format 'app_slug/secret_name', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
