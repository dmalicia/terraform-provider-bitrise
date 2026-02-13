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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AppBitriseYmlResource{}
var _ resource.ResourceWithImportState = &AppBitriseYmlResource{}

func NewAppBitriseYmlResource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AppBitriseYmlResource {
	return &AppBitriseYmlResource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type AppBitriseYmlResource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

type AppBitriseYmlResourceModel struct {
	AppSlug            types.String `tfsdk:"app_slug"`
	YmlContent         types.String `tfsdk:"yml_content"`
	UpdateOnCreateOnly types.Bool   `tfsdk:"update_on_create_only"`
	ID                 types.String `tfsdk:"id"`
}

type BitriseYmlRequest struct {
	AppConfigDatastoreYaml string `json:"app_config_datastore_yaml"`
}

type BitriseYmlResponse struct {
	AppConfigDatastoreYaml string `json:"app_config_datastore_yaml"`
}

// ignoreChangesIfUpdateOnCreateOnly is a custom plan modifier that prevents updates when update_on_create_only is true
type ignoreChangesIfUpdateOnCreateOnly struct{}

func (m ignoreChangesIfUpdateOnCreateOnly) Description(ctx context.Context) string {
	return "Ignores changes to yml_content when update_on_create_only is true"
}

func (m ignoreChangesIfUpdateOnCreateOnly) MarkdownDescription(ctx context.Context) string {
	return "Ignores changes to yml_content when update_on_create_only is true"
}

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

func (r *AppBitriseYmlResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_bitrise_yml"
}

func (r *AppBitriseYmlResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *AppBitriseYmlResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppBitriseYmlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating bitrise.yml", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
	})

	client := r.clientCreator(r.endpoint, r.token)
	appSlug := data.AppSlug.ValueString()
	ymlContent := data.YmlContent.ValueString()

	// Prepare request payload
	payload := BitriseYmlRequest{
		AppConfigDatastoreYaml: ymlContent,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshaling request payload",
			fmt.Sprintf("Could not marshal payload: %s", err.Error()),
		)
		return
	}

	// Create the PUT request
	url := fmt.Sprintf("%s/v0.1/apps/%s/bitrise.yml", r.endpoint, appSlug)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadJSON)))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating HTTP request",
			fmt.Sprintf("Could not create request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	tflog.Debug(ctx, "Sending HTTP request", map[string]interface{}{
		"method": "POST",
		"url":    url,
	})

	// Send the request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error sending HTTP request",
			fmt.Sprintf("Could not send request: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	// Read response body for debugging
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading response body",
			fmt.Sprintf("Could not read response: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Received HTTP response", map[string]interface{}{
		"status": httpResp.StatusCode,
		"body":   string(responseBody),
	})

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError(
			"API Request Error",
			fmt.Sprintf("Request failed with status %d: %s", httpResp.StatusCode, string(responseBody)),
		)
		return
	}

	// Set ID to app_slug
	data.ID = data.AppSlug

	tflog.Info(ctx, "Successfully created bitrise.yml")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppBitriseYmlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading bitrise.yml", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
	})

	client := r.clientCreator(r.endpoint, r.token)
	appSlug := data.AppSlug.ValueString()

	// Create the GET request
	url := fmt.Sprintf("%s/v0.1/apps/%s/bitrise.yml", r.endpoint, appSlug)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating HTTP request",
			fmt.Sprintf("Could not create request: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Sending HTTP request", map[string]interface{}{
		"method": "GET",
		"url":    url,
	})

	// Send the request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error sending HTTP request",
			fmt.Sprintf("Could not send request: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading response body",
			fmt.Sprintf("Could not read response: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Received HTTP response", map[string]interface{}{
		"status":       httpResp.StatusCode,
		"content-type": httpResp.Header.Get("Content-Type"),
		"body":         string(responseBody),
	})

	if httpResp.StatusCode == http.StatusNotFound {
		tflog.Warn(ctx, "Bitrise.yml not found, removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"API Request Error",
			fmt.Sprintf("Request failed with status %d: %s", httpResp.StatusCode, string(responseBody)),
		)
		return
	}

	// Parse response - handle both JSON and plain text responses
	var ymlContent string
	contentType := httpResp.Header.Get("Content-Type")
	
	// Try to parse as JSON first
	if strings.Contains(contentType, "application/json") || len(responseBody) > 0 && responseBody[0] == '{' {
		var ymlResponse BitriseYmlResponse
		err = json.Unmarshal(responseBody, &ymlResponse)
		if err != nil {
			tflog.Debug(ctx, "Failed to parse as JSON, treating as plain text", map[string]interface{}{
				"error": err.Error(),
			})
			// If JSON parsing fails, treat the response as plain YAML content
			ymlContent = string(responseBody)
		} else {
			ymlContent = ymlResponse.AppConfigDatastoreYaml
		}
	} else {
		// Response is likely plain text/YAML
		tflog.Debug(ctx, "Response appears to be plain text, not JSON")
		ymlContent = string(responseBody)
	}

	// Update state with current values
	data.YmlContent = types.StringValue(ymlContent)
	data.ID = data.AppSlug

	tflog.Info(ctx, "Successfully read bitrise.yml")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppBitriseYmlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if update_on_create_only is set to true
	if !data.UpdateOnCreateOnly.IsNull() && data.UpdateOnCreateOnly.ValueBool() {
		tflog.Info(ctx, "Skipping bitrise.yml update (update_on_create_only is true)", map[string]interface{}{
			"app_slug": data.AppSlug.ValueString(),
		})
		// Just update the state without making API call
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	tflog.Debug(ctx, "Updating bitrise.yml", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
	})

	client := r.clientCreator(r.endpoint, r.token)
	appSlug := data.AppSlug.ValueString()
	ymlContent := data.YmlContent.ValueString()

	// Prepare request payload
	payload := BitriseYmlRequest{
		AppConfigDatastoreYaml: ymlContent,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error marshaling request payload",
			fmt.Sprintf("Could not marshal payload: %s", err.Error()),
		)
		return
	}

	// Create the POST request (Bitrise API uses POST for updates)
	url := fmt.Sprintf("%s/v0.1/apps/%s/bitrise.yml", r.endpoint, appSlug)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadJSON)))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating HTTP request",
			fmt.Sprintf("Could not create request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	tflog.Debug(ctx, "Sending HTTP request", map[string]interface{}{
		"method": "POST",
		"url":    url,
	})

	// Send the request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error sending HTTP request",
			fmt.Sprintf("Could not send request: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	// Read response body for debugging
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading response body",
			fmt.Sprintf("Could not read response: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Received HTTP response", map[string]interface{}{
		"status": httpResp.StatusCode,
		"body":   string(responseBody),
	})

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError(
			"API Request Error",
			fmt.Sprintf("Request failed with status %d: %s", httpResp.StatusCode, string(responseBody)),
		)
		return
	}

	// Set ID to app_slug
	data.ID = data.AppSlug

	tflog.Info(ctx, "Successfully updated bitrise.yml")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppBitriseYmlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppBitriseYmlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting bitrise.yml from state", map[string]interface{}{
		"app_slug": data.AppSlug.ValueString(),
	})

	// Note: Bitrise API doesn't provide a delete endpoint for bitrise.yml
	// The resource is simply removed from Terraform state
	// The actual bitrise.yml file remains in the Bitrise app
	tflog.Info(ctx, "Removed bitrise.yml from Terraform state (file remains in Bitrise)")
}

func (r *AppBitriseYmlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using app_slug as the ID
	resource.ImportStatePassthroughID(ctx, path.Root("app_slug"), req, resp)
}
