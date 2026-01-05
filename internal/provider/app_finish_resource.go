package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AppFinishResource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

type AppFinishResourceModel struct {
	AppSlug          string            `tfsdk:"app_slug"`
	ProjectType      string            `tfsdk:"project_type"`
	StackID          string            `tfsdk:"stack_id"`
	Config           string            `tfsdk:"config"`
	Mode             string            `tfsdk:"mode"`
	Envs             map[string]string `tfsdk:"envs"`
	OrganizationSlug string            `tfsdk:"organization_slug"`
}

type PayloadFinish struct {
	AppSlug          string            `json:"app_slug"`
	ProjectType      string            `json:"project_type"`
	StackID          string            `json:"stack_id"`
	Config           string            `json:"config"`
	Mode             string            `json:"mode"`
	Envs             map[string]string `json:"envs"`
	OrganizationSlug string            `json:"organization_slug"`
}

func NewAppFinishResource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AppFinishResource {
	return &AppFinishResource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

func (r *AppFinishResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_finish"
}

func (r *AppFinishResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *AppFinishResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		tflog.Debug(ctx, "MODULEDEBUG: Provider data is missing")
		return
	}
	clientCreator, ok := req.ProviderData.(func(endpoint, token string) *http.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.clientCreator = clientCreator
	tflog.Debug(ctx, "MODULEDEBUG: Provider configuration successful")
}

func (r *AppFinishResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Initialize data to store API response
	var data AppFinishResourceModel

	// Populate data from Terraform plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Check for errors in diagnostics and return if found
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Starting AppFinishResource Create")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Construct the URL for creating the request
	appSlug := data.AppSlug
	completeURL := fmt.Sprintf("%s/v0.1/apps/%s/finish", r.endpoint, appSlug)

	// Construct the payload data using the provided variables
	payload := PayloadFinish{
		ProjectType:      data.ProjectType,
		StackID:          data.StackID,
		Config:           data.Config,
		Mode:             data.Mode,
		Envs:             data.Envs,
		OrganizationSlug: data.OrganizationSlug,
	}

	// Marshal the payload struct into a JSON string
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		tflog.Error(ctx, "Error marshaling JSON payload", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Request payload", map[string]interface{}{"payload_json": string(payloadJSON)})

	// Create an HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", completeURL, strings.NewReader(string(payloadJSON)))
	if err != nil {
		tflog.Error(ctx, "Error creating HTTP request", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Dump the HTTP request details
	dump, _ := httputil.DumpRequest(httpReq, true)
	tflog.Debug(ctx, "MODULEDEBUG: HTTP Request", map[string]interface{}{"request": string(dump)})

	// Send the HTTP request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		tflog.Error(ctx, "Error sending HTTP request", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}
	defer httpResp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		tflog.Error(ctx, "Error reading response body", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}
	tflog.Debug(ctx, "MODULEDEBUG: Response body", map[string]interface{}{"body": string(responseBody)})

	// Debugging: Print response status and headers
	printResponseInfo(httpResp)

	if httpResp.StatusCode != http.StatusOK {
		tflog.Error(ctx, "MODULEDEBUG: Request did not succeed", map[string]interface{}{
			"status": httpResp.Status,
			"headers": httpResp.Header,
		})
		resp.Diagnostics.AddError("MODULEDEBUG: API Request Error", fmt.Sprintf("Request did not succeed: %s", httpResp.Status))
		return
	}

	tflog.Info(ctx, "MODULEDEBUG: App registration completed successfully")

	// Update resource state with populated data
	resp.State.Set(ctx, &data)
}

func (r *AppFinishResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppFinishResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Starting AppFinishResource Read")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Get the app slug from state
	appSlug := data.AppSlug
	if appSlug == "" {
		tflog.Warn(ctx, "AppSlug is empty, skipping Read")
		resp.State.Set(ctx, &data)
		return
	}

	// Construct the URL to get the app details
	completeURL := fmt.Sprintf("%s/v0.1/apps/%s", r.endpoint, appSlug)

	tflog.Debug(ctx, "MODULEDEBUG: Fetching app details for Finish resource", map[string]interface{}{"url": completeURL})

	// Create an HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", completeURL, nil)
	if err != nil {
		tflog.Error(ctx, "Error creating HTTP request", map[string]interface{}{"error": err.Error()})
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request: %s", err))
		return
	}

	// Send the HTTP request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		tflog.Error(ctx, "Error sending HTTP request", map[string]interface{}{"error": err.Error()})
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read app: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// If the app was deleted (404), remove it from state
	if httpResp.StatusCode == http.StatusNotFound {
		tflog.Info(ctx, "App not found, removing Finish resource from state", map[string]interface{}{"app_slug": appSlug})
		resp.State.RemoveResource(ctx)
		return
	}

	if httpResp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(httpResp.Body)
		tflog.Error(ctx, "Request did not succeed", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read app, got status: %s", httpResp.Status))
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: App still exists, keeping Finish resource in state")

	// Save updated data into Terraform state
	resp.State.Set(ctx, &data)
}

func (r *AppFinishResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppFinishResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Starting AppFinishResource Update")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Construct the URL for updating the request
	appSlug := data.AppSlug
	completeURL := fmt.Sprintf("%s/v0.1/apps/%s/finish", r.endpoint, appSlug)

	// Construct the payload data using the provided variables
	payload := PayloadFinish{
		ProjectType:      data.ProjectType,
		StackID:          data.StackID,
		Config:           data.Config,
		Mode:             data.Mode,
		Envs:             data.Envs,
		OrganizationSlug: data.OrganizationSlug,
	}

	// Marshal the payload struct into a JSON string
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		tflog.Error(ctx, "Error marshaling JSON payload", map[string]interface{}{"error": err.Error()})
		resp.Diagnostics.AddError("JSON Error", fmt.Sprintf("Unable to marshal payload: %s", err))
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Update request payload", map[string]interface{}{"payload_json": string(payloadJSON)})

	// Create an HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", completeURL, strings.NewReader(string(payloadJSON)))
	if err != nil {
		tflog.Error(ctx, "Error creating HTTP request", map[string]interface{}{"error": err.Error()})
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request: %s", err))
		return
	}

	// Dump the HTTP request details
	dump, _ := httputil.DumpRequest(httpReq, true)
	tflog.Debug(ctx, "MODULEDEBUG: HTTP Request", map[string]interface{}{"request": string(dump)})

	// Send the HTTP request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		tflog.Error(ctx, "Error sending HTTP request", map[string]interface{}{"error": err.Error()})
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update app finish: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		tflog.Error(ctx, "Error reading response body", map[string]interface{}{"error": err.Error()})
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read response: %s", err))
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Response Body", map[string]interface{}{"body": string(responseBody)})

	// Check response status
	if httpResp.StatusCode != http.StatusOK {
		tflog.Error(ctx, "Request did not succeed", map[string]interface{}{
			"status": httpResp.Status,
			"body":   string(responseBody),
		})
		resp.Diagnostics.AddError("API Request Error", fmt.Sprintf("Request did not succeed: %s - %s", httpResp.Status, string(responseBody)))
		return
	}

	tflog.Info(ctx, "App finish configuration updated successfully")

	// Update resource state with populated data
	resp.State.Set(ctx, &data)
}

func (r *AppFinishResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppFinishResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "MODULEDEBUG: Finish resource deleted from state (no API call needed)")
	// The finish endpoint doesn't require cleanup, just remove from state
	resp.State.RemoveResource(ctx)
}

func (r *AppFinishResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("app_slug"), req, resp)
}
