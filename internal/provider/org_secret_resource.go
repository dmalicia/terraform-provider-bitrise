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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var (
	_ resource.ResourceWithConfigure   = &OrgSecretResource{}
	_ resource.ResourceWithImportState = &OrgSecretResource{}
)

// NewOrgSecretResource returns a new bitrise_org_secret resource.
func NewOrgSecretResource() resource.Resource {
	return &OrgSecretResource{}
}

// OrgSecretResource manages a secret shared by all apps of a Bitrise organization.
type OrgSecretResource struct {
	resourceWithClient
}

// OrgSecretResourceModel describes the bitrise_org_secret resource data model.
type OrgSecretResourceModel struct {
	OrgSlug                  types.String `tfsdk:"org_slug"`
	Name                     types.String `tfsdk:"name"`
	Value                    types.String `tfsdk:"value"`
	IsProtected              types.Bool   `tfsdk:"is_protected"`
	IsExposedForPullRequests types.Bool   `tfsdk:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       types.Bool   `tfsdk:"expand_in_step_inputs"`
	Scope                    types.String `tfsdk:"scope"`
	ID                       types.String `tfsdk:"id"`
}

// Booleans are sent explicitly (no omitempty) so that `false` is not replaced by the API default.
type orgSecretCreateRequest struct {
	Name                     string `json:"name"`
	Value                    string `json:"value"`
	IsProtected              bool   `json:"is_protected"`
	IsExposedForPullRequests bool   `json:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       bool   `json:"expand_in_step_inputs"`
}

type orgSecretUpdateRequest struct {
	Value                    string `json:"value"`
	IsProtected              bool   `json:"is_protected"`
	IsExposedForPullRequests bool   `json:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       bool   `json:"expand_in_step_inputs"`
}

type orgSecretResponse struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	Value                    *string `json:"value"` // nil when omitted (protected); "" is a real value
	IsProtected              bool    `json:"is_protected"`
	IsExposedForPullRequests bool    `json:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       bool    `json:"expand_in_step_inputs"`
	Scope                    string  `json:"scope"`
}

// Metadata returns the resource type name.
func (r *OrgSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_secret"
}

// Schema returns the resource schema.
func (r *OrgSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a workspace (organization) secret in Bitrise. Workspace secrets are shared by every app in the workspace; an app-level secret with the same name overrides it.",
		Attributes: map[string]schema.Attribute{
			"org_slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the Bitrise workspace (organization)",
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
				MarkdownDescription: "If true, the secret value cannot be retrieved via the API or the UI. Bitrise does not allow unprotecting a secret, so changing this from `true` to `false` forces a new resource. Default: false",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplaceIf(
						func(_ context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = req.StateValue.ValueBool() && !req.PlanValue.ValueBool()
						},
						"Unprotecting a secret requires recreating it.",
						"Unprotecting a secret requires recreating it.",
					),
				},
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
			"scope": schema.StringAttribute{
				MarkdownDescription: "The scope reported by Bitrise (e.g. `workspace`)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The identifier of the secret in the format `org_slug/name`",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the org secret.
func (r *OrgSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgSlug := data.OrgSlug.ValueString()
	name := data.Name.ValueString()
	var secretResp orgSecretResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("create secret %s of organization %s", name, orgSlug),
		Method: http.MethodPost,
		Path:   bitriseapi.OrgSecretsPath(orgSlug),
		Body: orgSecretCreateRequest{
			Name:                     name,
			Value:                    data.Value.ValueString(),
			IsProtected:              data.IsProtected.ValueBool(),
			IsExposedForPullRequests: data.IsExposedForPullRequests.ValueBool(),
			ExpandInStepInputs:       data.ExpandInStepInputs.ValueBool(),
		},
		Accept: []int{http.StatusCreated},
	}, &secretResp)
	if err != nil {
		orgSecretCallDiags("creating", "create").add(&resp.Diagnostics, err)
		return
	}

	data.Scope = types.StringValue(secretResp.Scope)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", orgSlug, name))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the org secret settings; the value is only returned for unprotected secrets.
func (r *OrgSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgSlug := data.OrgSlug.ValueString()
	name := data.Name.ValueString()
	var secretResp orgSecretResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("read secret %s of organization %s", name, orgSlug),
		Method: http.MethodGet,
		Path:   bitriseapi.OrgSecretPath(orgSlug, name),
	}, &secretResp)
	var callErr *bitriseapi.CallError
	switch {
	case err == nil:
	case errors.As(err, &callErr):
		orgSecretCallDiags("reading", "read").add(&resp.Diagnostics, err)
		return
	case errors.Is(err, bitriseapi.ErrNotFound):
		tflog.Info(ctx, "Org secret not found, removing from state", map[string]interface{}{
			"org_slug": orgSlug,
			"name":     name,
		})
		resp.State.RemoveResource(ctx)
		return
	default:
		orgSecretCallDiags("reading", "read").add(&resp.Diagnostics, err)
		return
	}

	data.IsProtected = types.BoolValue(secretResp.IsProtected)
	data.IsExposedForPullRequests = types.BoolValue(secretResp.IsExposedForPullRequests)
	data.ExpandInStepInputs = types.BoolValue(secretResp.ExpandInStepInputs)
	data.Scope = types.StringValue(secretResp.Scope)
	if data.ID.IsNull() || data.ID.IsUnknown() {
		data.ID = types.StringValue(fmt.Sprintf("%s/%s", orgSlug, name))
	}

	// Protected values are never returned, so drift on them cannot be detected.
	if !secretResp.IsProtected && secretResp.Value != nil {
		data.Value = types.StringValue(*secretResp.Value)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the org secret value and settings.
func (r *OrgSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgSlug := data.OrgSlug.ValueString()
	name := data.Name.ValueString()
	var secretResp orgSecretResponse
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("update secret %s of organization %s", name, orgSlug),
		Method: http.MethodPatch,
		Path:   bitriseapi.OrgSecretPath(orgSlug, name),
		Body: orgSecretUpdateRequest{
			Value:                    data.Value.ValueString(),
			IsProtected:              data.IsProtected.ValueBool(),
			IsExposedForPullRequests: data.IsExposedForPullRequests.ValueBool(),
			ExpandInStepInputs:       data.ExpandInStepInputs.ValueBool(),
		},
	}, &secretResp)
	// The update response body is optional; only a returned scope is applied.
	var callErr *bitriseapi.CallError
	switch {
	case err == nil:
		if secretResp.Scope != "" {
			data.Scope = types.StringValue(secretResp.Scope)
		}
	case errors.As(err, &callErr) && callErr.Stage == bitriseapi.StageDecode:
	default:
		orgSecretCallDiags("updating", "update").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the org secret; a secret that is already gone is not an error.
func (r *OrgSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgSlug := data.OrgSlug.ValueString()
	name := data.Name.ValueString()
	err := r.client.Do(ctx, bitriseapi.Request{
		Op:     fmt.Sprintf("delete secret %s of organization %s", name, orgSlug),
		Method: http.MethodDelete,
		Path:   bitriseapi.OrgSecretPath(orgSlug, name),
		Accept: []int{http.StatusNoContent, http.StatusNotFound},
	}, &bitriseapi.RawResponse{})
	if err != nil {
		orgSecretCallDiags("deleting", "delete").add(&resp.Diagnostics, err)
	}
}

// ImportState imports the resource by "org_slug/secret_name".
func (r *OrgSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in the format 'org_slug/secret_name', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// orgSecretCallDiags reports failures as "Error <gerund> org secret", e.g. "Error creating org secret".
func orgSecretCallDiags(gerund, verb string) callDiags {
	failed := diagText{"Error " + gerund + " org secret", "%s"}
	return callDiags{
		encode: failed,
		build:  failed,
		send:   failed,
		read:   failed,
		decode: diagText{"Error parsing response", "%s"},
		status: func(e *bitriseapi.StatusError) (string, string) {
			return "API Error", fmt.Sprintf("Failed to %s org secret: HTTP %d - %s", verb, e.Code, e.Body)
		},
	}
}
