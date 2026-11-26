package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var (
	_ resource.ResourceWithConfigure   = &AppResource{}
	_ resource.ResourceWithImportState = &AppResource{}
)

// appImportedKey marks private state written by ImportState until the first Read.
const appImportedKey = "imported"

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
		Title     string `json:"title"`
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
	Title            string `json:"title,omitempty"`
	Type             string `json:"type"`
}

type appUpdateRequest struct {
	IsPublic      bool   `json:"is_public"`
	RepositoryURL string `json:"repository_url,omitempty"`
	Title         string `json:"title,omitempty"`
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
	Title                 types.String `tfsdk:"title"`
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
			"title": schema.StringAttribute{
				MarkdownDescription: "Display name of the app. When omitted, Bitrise names the app after the repository and Terraform does not manage the name.",
				Optional:            true,
				Validators:          []validator.String{nonEmptyString{}},
			},
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "App Slug",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			Title:            data.Title.ValueString(),
			Type:             data.Type.ValueString(),
		},
	}, &created)
	if err != nil {
		appCallDiags("Unable to create request: %s", "Unable to register app: %s").add(&resp.Diagnostics, err)
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
	imported, diags := req.Private.GetKey(ctx, appImportedKey)
	resp.Diagnostics.Append(diags...)
	// is_public is optional but not computed and an omitted value is sent as
	// false, so keep it null unless imported, set or changed remotely.
	if imported != nil || !data.IsPublic.IsNull() || app.Data.IsPublic {
		data.IsPublic = types.BoolValue(app.Data.IsPublic)
	}
	// title stays null (unmanaged) unless set or imported.
	if (imported != nil || !data.Title.IsNull()) && app.Data.Title != "" {
		data.Title = types.StringValue(app.Data.Title)
	}
	if imported != nil {
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, appImportedKey, nil)...)
	}
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

// Update changes the visibility, repository URL and title of the app.
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
			Title:         data.Title.ValueString(),
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
	// Read and Update address the app by app_slug, so it must be set too.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_slug"), req.ID)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, appImportedKey, []byte("true"))...)
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

// nonEmptyString rejects "", which the API would ignore, leaving a permanent diff.
type nonEmptyString struct{}

func (nonEmptyString) Description(context.Context) string { return "value must not be empty" }

func (v nonEmptyString) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (nonEmptyString) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "" {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid Attribute Value", "Must not be empty. Omit the attribute to leave the name unmanaged.")
}
