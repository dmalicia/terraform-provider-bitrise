package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
	_ resource.ResourceWithConfigure   = &AppResource{}
	_ resource.ResourceWithImportState = &AppResource{}
)

// AppResource manages a Bitrise app registration.
type AppResource struct {
	resourceWithClient
}

// NewAppResource returns a new bitrise_app resource.
func NewAppResource() resource.Resource {
	return &AppResource{}
}

// CreateResponse is the response of the app registration endpoint.
type CreateResponse struct {
	Status     string `json:"status"`
	Slug       string `json:"slug"`
	ProviderID string `json:"provider_id"`
}

// AppDetailsResponse is the response of the app details endpoint.
type AppDetailsResponse struct {
	Data struct {
		Slug     string `json:"slug"`
		RepoURL  string `json:"repo_url"`
		IsPublic bool   `json:"is_public"`
		Owner    struct {
			AccountType string `json:"account_type"`
			Name        string `json:"name"`
			Slug        string `json:"slug"`
		} `json:"owner"`
		RepoSlug  string `json:"repo_slug"`
		Provider  string `json:"provider"`
		RepoOwner string `json:"repo_owner"`
	} `json:"data"`
}

// Field order matches the sorted keys of the original map payload.
type appRegisterRequest struct {
	GitOwner         string `json:"git_owner"`
	GitRepoSlug      string `json:"git_repo_slug"`
	IsPublic         bool   `json:"is_public"`
	OrganizationSlug string `json:"organization_slug"`
	Provider         string `json:"provider"`
	RepoURL          string `json:"repo_url"`
	Type             string `json:"type"`
}

type appUpdateRequest struct {
	IsPublic      bool   `json:"is_public"`
	RepositoryURL string `json:"repository_url,omitempty"`
}

// AppResourceModel describes the bitrise_app resource data model.
type AppResourceModel struct {
	ConfigurableAttribute types.String `tfsdk:"configurable_attribute"`
	ID                    types.String `tfsdk:"id"`
	Repo                  types.String `tfsdk:"repo"`
	IsPublic              types.Bool   `tfsdk:"is_public"`
	OrganizationSlug      types.String `tfsdk:"organization_slug"`
	RepoURL               types.String `tfsdk:"repo_url"`
	Type                  types.String `tfsdk:"type"`
	GitRepoSlug           types.String `tfsdk:"git_repo_slug"`
	GitOwner              types.String `tfsdk:"git_owner"`
	AppSlug               types.String `tfsdk:"app_slug"`
}

// Metadata returns the resource type name.
func (r *AppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

// Schema returns the resource schema.
func (r *AppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "App resource",
		Attributes: map[string]schema.Attribute{
			"configurable_attribute": schema.StringAttribute{
				MarkdownDescription: "App configurable attribute",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "App identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repo": schema.StringAttribute{
				MarkdownDescription: "Repo",
				Optional:            true,
			},
			"is_public": schema.BoolAttribute{
				MarkdownDescription: "Is Public",
				Optional:            true,
			},
			"organization_slug": schema.StringAttribute{
				MarkdownDescription: "Organization Slug",
				Optional:            true,
			},
			"repo_url": schema.StringAttribute{
				MarkdownDescription: "Repo URL",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type",
				Optional:            true,
			},
			"git_repo_slug": schema.StringAttribute{
				MarkdownDescription: "Git Repo Slug",
				Optional:            true,
			},
			"git_owner": schema.StringAttribute{
				MarkdownDescription: "Git Owner",
				Optional:            true,
			},
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "App Slug",
				Computed:            true,
			},
		},
	}
}

// Create registers the app.
func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Starting AppResource Create")

	var created CreateResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     "register app",
		Method: http.MethodPost,
		Path:   bitriseapi.RegisterAppPath,
		Body: appRegisterRequest{
			GitOwner:         data.GitOwner.ValueString(),
			GitRepoSlug:      data.GitRepoSlug.ValueString(),
			IsPublic:         data.IsPublic.ValueBool(),
			OrganizationSlug: data.OrganizationSlug.ValueString(),
			Provider:         data.Repo.ValueString(),
			RepoURL:          data.RepoURL.ValueString(),
			Type:             data.Type.ValueString(),
		},
	}, &created)
	if err != nil {
		// Pre-existing behaviour: only rejections with a JSON body are reported.
		var callErr *bitriseapi.CallError
		var statusErr *bitriseapi.StatusError
		if !errors.As(err, &callErr) && errors.As(err, &statusErr) && json.Unmarshal([]byte(statusErr.Body), &created) == nil {
			resp.Diagnostics.AddError("API Request Error", fmt.Sprintf("Request did not succeed: %s", statusErr.Status()))
		}
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Captured app slug", map[string]interface{}{"slug": created.Slug})
	data.AppSlug = types.StringValue(created.Slug)
	data.ID = types.StringValue(created.Slug)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the app.
func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     "delete app " + appSlug,
		Method: http.MethodDelete,
		Path:   bitriseapi.AppPath(appSlug),
		Accept: []int{http.StatusOK, http.StatusNoContent},
	}, nil)
	var statusErr *bitriseapi.StatusError
	switch {
	case err == nil:
		resp.State.RemoveResource(ctx)
	case errors.As(err, &statusErr):
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to delete app: %s - %s", statusErr.Status(), statusErr.Body))
	default:
		appCallDiags("Unable to create delete request: %s", "Unable to delete app: %s").add(&resp.Diagnostics, err)
	}
}

// Read refreshes the app from the API.
func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	if appSlug == "" {
		tflog.Warn(ctx, "AppSlug is empty, skipping Read")
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	var app AppDetailsResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     "read app " + appSlug,
		Method: http.MethodGet,
		Path:   bitriseapi.AppPath(appSlug),
	}, &app)
	var statusErr *bitriseapi.StatusError
	switch {
	case err == nil:
	case errors.Is(err, bitriseapi.ErrNotFound):
		tflog.Info(ctx, "App not found, removing from state", map[string]interface{}{"app_slug": appSlug})
		resp.State.RemoveResource(ctx)
		return
	case errors.As(err, &statusErr):
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read app, got status: %s", statusErr.Status()))
		return
	default:
		appCallDiags("Unable to create request: %s", "Unable to read app: %s").add(&resp.Diagnostics, err)
		return
	}

	if app.Data.RepoURL != "" {
		data.RepoURL = types.StringValue(app.Data.RepoURL)
	}
	data.IsPublic = types.BoolValue(app.Data.IsPublic)
	if app.Data.RepoOwner != "" {
		data.GitOwner = types.StringValue(app.Data.RepoOwner)
	}
	if app.Data.RepoSlug != "" {
		data.GitRepoSlug = types.StringValue(app.Data.RepoSlug)
	}
	if app.Data.Provider != "" {
		data.Repo = types.StringValue(app.Data.Provider)
	}
	data.ID = types.StringValue(app.Data.Slug)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update changes the visibility and repository URL of the app.
func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appSlug := data.AppSlug.ValueString()
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     "update app " + appSlug,
		Method: http.MethodPatch,
		Path:   bitriseapi.AppPath(appSlug),
		Body: appUpdateRequest{
			IsPublic:      data.IsPublic.ValueBool(),
			RepositoryURL: data.RepoURL.ValueString(),
		},
	}, &bitriseapi.RawResponse{})
	if err != nil {
		appCallDiags("Unable to create request: %s", "Unable to update app: %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ImportState imports an app by its slug.
func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// appCallDiags returns the diagnostics shared by the app, app finish and app SSH resources.
func appCallDiags(buildDetail, sendDetail string) callDiags {
	return callDiags{
		encode: diagText{"JSON Error", "Unable to marshal payload: %s"},
		build:  diagText{"Client Error", buildDetail},
		send:   diagText{"Client Error", sendDetail},
		read:   diagText{"Client Error", "Unable to read response: %s"},
		decode: diagText{"Parse Error", "Unable to parse API response: %s"},
		status: func(e *bitriseapi.StatusError) (string, string) {
			return "API Request Error", fmt.Sprintf("Request did not succeed: %s - %s", e.Status(), e.Body)
		},
	}
}
