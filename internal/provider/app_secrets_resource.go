package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
)

var _ resource.Resource = &AppSecretsResource{}
var _ resource.ResourceWithImportState = &AppSecretsResource{}

func NewAppSecretsResource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AppSecretsResource {
	return &AppSecretsResource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type AppSecretsResource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

type AppSecretsResourceModel struct {
	AppSlug                      types.String `tfsdk:"app_slug"`
	Name                         types.String `tfsdk:"name"`
	Value                        types.String `tfsdk:"value"`
	IsProtected                  types.Bool   `tfsdk:"is_protected"`
	IsExposedForPullRequests     types.Bool   `tfsdk:"is_exposed_for_pull_requests"`
	ExpandInStepInputs           types.Bool   `tfsdk:"expand_in_step_inputs"`
	ID                           types.String `tfsdk:"id"`
}

type SecretCreateRequest struct {
	Name                     string `json:"name"`
	Value                    string `json:"value"`
	IsProtected              bool   `json:"is_protected,omitempty"`
	IsExposedForPullRequests bool   `json:"is_exposed_for_pull_requests,omitempty"`
	ExpandInStepInputs       bool   `json:"expand_in_step_inputs,omitempty"`
}

type SecretUpdateRequest struct {
	Value                    string `json:"value,omitempty"`
	IsProtected              *bool  `json:"is_protected,omitempty"`
	IsExposedForPullRequests *bool  `json:"is_exposed_for_pull_requests,omitempty"`
	ExpandInStepInputs       *bool  `json:"expand_in_step_inputs,omitempty"`
}

type SecretResponse struct {
	Name                     string `json:"name"`
	Value                    string `json:"value,omitempty"`
	IsProtected              bool   `json:"is_protected"`
	IsExposedForPullRequests bool   `json:"is_exposed_for_pull_requests"`
	ExpandInStepInputs       bool   `json:"expand_in_step_inputs"`
	ID                       string `json:"id"`
}

func (r *AppSecretsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_secret"
}

func (r *AppSecretsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *AppSecretsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppSecretsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Bitrise app secret", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
		"name":     data.Name.ValueString(),
	})

	client := r.clientCreator(r.endpoint, r.token)
	url := fmt.Sprintf("%s/v0.1/apps/%s/secrets", r.endpoint, data.AppSlug.ValueString())

	secretReq := SecretCreateRequest{
		Name:                     data.Name.ValueString(),
		Value:                    data.Value.ValueString(),
		IsProtected:              data.IsProtected.ValueBool(),
		IsExposedForPullRequests: data.IsExposedForPullRequests.ValueBool(),
		ExpandInStepInputs:       data.ExpandInStepInputs.ValueBool(),
	}

	payloadJSON, err := json.Marshal(secretReq)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request", err.Error())
		return
	}

	tflog.Debug(ctx, "Sending POST request to create secret", map[string]interface{}{
		"url": url,
	})

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadJSON)))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HTTP request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

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

	if httpResp.StatusCode != http.StatusCreated {
		tflog.Error(ctx, "Failed to create secret", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to create secret: %s - %s", httpResp.Status, string(responseBody)),
		)
		return
	}

	var secretResp SecretResponse
	if err := json.Unmarshal(responseBody, &secretResp); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Set the ID to a combination of app_slug and secret name for import/identification
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.AppSlug.ValueString(), data.Name.ValueString()))

	tflog.Info(ctx, "Successfully created Bitrise app secret", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppSecretsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading Bitrise app secret", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
		"name":     data.Name.ValueString(),
	})

	client := r.clientCreator(r.endpoint, r.token)
	url := fmt.Sprintf("%s/v0.1/apps/%s/secrets/%s", r.endpoint, data.AppSlug.ValueString(), data.Name.ValueString())

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
		tflog.Info(ctx, "Secret not found, removing from state", map[string]interface{}{
			"app_slug": data.AppSlug.ValueString(),
			"name":     data.Name.ValueString(),
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
		tflog.Error(ctx, "Failed to read secret", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to read secret: %s - %s", httpResp.Status, string(responseBody)),
		)
		return
	}

	var secretResp SecretResponse
	if err := json.Unmarshal(responseBody, &secretResp); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Update state with values from API
	// Note: If the secret is protected, the value won't be returned, so we keep the current state value
	data.IsProtected = types.BoolValue(secretResp.IsProtected)
	data.IsExposedForPullRequests = types.BoolValue(secretResp.IsExposedForPullRequests)
	data.ExpandInStepInputs = types.BoolValue(secretResp.ExpandInStepInputs)
	
	// Only update value if it's not protected and returned by API
	if !secretResp.IsProtected && secretResp.Value != "" {
		data.Value = types.StringValue(secretResp.Value)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppSecretsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Bitrise app secret", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
		"name":     data.Name.ValueString(),
	})

	client := r.clientCreator(r.endpoint, r.token)
	url := fmt.Sprintf("%s/v0.1/apps/%s/secrets/%s", r.endpoint, data.AppSlug.ValueString(), data.Name.ValueString())

	// Build update request with only the fields that should be updated
	isProtected := data.IsProtected.ValueBool()
	isExposed := data.IsExposedForPullRequests.ValueBool()
	expandInputs := data.ExpandInStepInputs.ValueBool()

	secretReq := SecretUpdateRequest{
		Value:                    data.Value.ValueString(),
		IsProtected:              &isProtected,
		IsExposedForPullRequests: &isExposed,
		ExpandInStepInputs:       &expandInputs,
	}

	payloadJSON, err := json.Marshal(secretReq)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request", err.Error())
		return
	}

	tflog.Debug(ctx, "Sending PATCH request to update secret", map[string]interface{}{
		"url": url,
	})

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, strings.NewReader(string(payloadJSON)))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HTTP request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
		tflog.Error(ctx, "Failed to update secret", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to update secret: %s - %s", httpResp.Status, string(responseBody)),
		)
		return
	}

	tflog.Info(ctx, "Successfully updated Bitrise app secret")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppSecretsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppSecretsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting Bitrise app secret", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
		"name":     data.Name.ValueString(),
	})

	client := r.clientCreator(r.endpoint, r.token)
	url := fmt.Sprintf("%s/v0.1/apps/%s/secrets/%s", r.endpoint, data.AppSlug.ValueString(), data.Name.ValueString())

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
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

	// 404 means already deleted, which is fine
	if httpResp.StatusCode == http.StatusNotFound {
		tflog.Info(ctx, "Secret already deleted")
		return
	}

	if httpResp.StatusCode != http.StatusNoContent {
		responseBody, _ := io.ReadAll(httpResp.Body)
		tflog.Error(ctx, "Failed to delete secret", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError(
			"API Error",
			fmt.Sprintf("Failed to delete secret: %s - %s", httpResp.Status, string(responseBody)),
		)
		return
	}

	tflog.Info(ctx, "Successfully deleted Bitrise app secret")
}

func (r *AppSecretsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID should be in the format: app_slug/secret_name
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
	
	// Note: The value won't be imported, user needs to set it manually after import
	// or if the secret is not protected, it will be fetched during the first read
}
