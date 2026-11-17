package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
)

var _ resource.Resource = &OrgSecretResource{}
var _ resource.ResourceWithImportState = &OrgSecretResource{}

func NewOrgSecretResource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *OrgSecretResource {
	return &OrgSecretResource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type OrgSecretResource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

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

func (r *OrgSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_secret"
}

func (r *OrgSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
						func(ctx context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
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

func (r *OrgSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgSecretResource) secretURL(orgSlug, name string) string {
	return fmt.Sprintf("%s/v0.1/organizations/%s/secrets/%s", r.endpoint, url.PathEscape(orgSlug), url.PathEscape(name))
}

func (r *OrgSecretResource) do(ctx context.Context, method, reqURL string, payload interface{}) (int, []byte, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(b)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpResp, err := r.clientCreator(r.endpoint, r.token).Do(httpReq)
	if err != nil {
		return 0, nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	return httpResp.StatusCode, respBody, err
}

func (r *OrgSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Bitrise org secret", map[string]interface{}{
		"org_slug": data.OrgSlug.ValueString(),
		"name":     data.Name.ValueString(),
	})

	reqURL := fmt.Sprintf("%s/v0.1/organizations/%s/secrets", r.endpoint, url.PathEscape(data.OrgSlug.ValueString()))
	status, body, err := r.do(ctx, http.MethodPost, reqURL, orgSecretCreateRequest{
		Name:                     data.Name.ValueString(),
		Value:                    data.Value.ValueString(),
		IsProtected:              data.IsProtected.ValueBool(),
		IsExposedForPullRequests: data.IsExposedForPullRequests.ValueBool(),
		ExpandInStepInputs:       data.ExpandInStepInputs.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating org secret", err.Error())
		return
	}
	if status != http.StatusCreated {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to create org secret: HTTP %d - %s", status, string(body)))
		return
	}

	var secretResp orgSecretResponse
	if err := json.Unmarshal(body, &secretResp); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	data.Scope = types.StringValue(secretResp.Scope)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.OrgSlug.ValueString(), data.Name.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrgSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, body, err := r.do(ctx, http.MethodGet, r.secretURL(data.OrgSlug.ValueString(), data.Name.ValueString()), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading org secret", err.Error())
		return
	}
	if status == http.StatusNotFound {
		tflog.Info(ctx, "Org secret not found, removing from state", map[string]interface{}{
			"org_slug": data.OrgSlug.ValueString(),
			"name":     data.Name.ValueString(),
		})
		resp.State.RemoveResource(ctx)
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to read org secret: HTTP %d - %s", status, string(body)))
		return
	}

	var secretResp orgSecretResponse
	if err := json.Unmarshal(body, &secretResp); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	data.IsProtected = types.BoolValue(secretResp.IsProtected)
	data.IsExposedForPullRequests = types.BoolValue(secretResp.IsExposedForPullRequests)
	data.ExpandInStepInputs = types.BoolValue(secretResp.ExpandInStepInputs)
	data.Scope = types.StringValue(secretResp.Scope)
	if data.ID.IsNull() || data.ID.IsUnknown() {
		data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.OrgSlug.ValueString(), data.Name.ValueString()))
	}

	// Protected values are never returned, so drift on them cannot be detected.
	if !secretResp.IsProtected && secretResp.Value != nil {
		data.Value = types.StringValue(*secretResp.Value)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrgSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, body, err := r.do(ctx, http.MethodPatch, r.secretURL(data.OrgSlug.ValueString(), data.Name.ValueString()), orgSecretUpdateRequest{
		Value:                    data.Value.ValueString(),
		IsProtected:              data.IsProtected.ValueBool(),
		IsExposedForPullRequests: data.IsExposedForPullRequests.ValueBool(),
		ExpandInStepInputs:       data.ExpandInStepInputs.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating org secret", err.Error())
		return
	}
	if status != http.StatusOK {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to update org secret: HTTP %d - %s", status, string(body)))
		return
	}

	var secretResp orgSecretResponse
	if err := json.Unmarshal(body, &secretResp); err == nil && secretResp.Scope != "" {
		data.Scope = types.StringValue(secretResp.Scope)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OrgSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OrgSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	status, body, err := r.do(ctx, http.MethodDelete, r.secretURL(data.OrgSlug.ValueString(), data.Name.ValueString()), nil)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting org secret", err.Error())
		return
	}
	if status != http.StatusNoContent && status != http.StatusNotFound {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Failed to delete org secret: HTTP %d - %s", status, string(body)))
	}
}

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
