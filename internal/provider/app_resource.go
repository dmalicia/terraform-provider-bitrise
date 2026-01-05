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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/zclconf/go-cty/cty"
)

type AppResource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

func NewAppResource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AppResource {
	return &AppResource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

// Define a struct that matches the structure of the JSON response
type CreateResponse struct {
	Status     string `json:"status"`
	Slug       string `json:"slug"`
	ProviderID string `json:"provider_id"`
}

type AppResourceModel struct {
	ConfigurableAttribute types.String `tfsdk:"configurable_attribute"`
	Id                    types.String `tfsdk:"id"`
	Repo                  types.String `tfsdk:"repo"`
	IsPublic              types.Bool   `tfsdk:"is_public"`
	OrganizationSlug      types.String `tfsdk:"organization_slug"`
	RepoURL               types.String `tfsdk:"repo_url"`
	Type                  types.String `tfsdk:"type"`
	GitRepoSlug           types.String `tfsdk:"git_repo_slug"`
	GitOwner              types.String `tfsdk:"git_owner"`
	AppSlug               types.String `tfsdk:"app_slug"`
}

func (r *AppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (r *AppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *AppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		tflog.Debug(ctx, "Provider data is missing")
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
	tflog.Debug(ctx, "Provider configuration successful")
}

func (r *AppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Initialize data to store API response
	var data AppResourceModel

	// Populate data from Terraform plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Check for errors in diagnostics and return if found
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Starting AppResource Create")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Construct the URL for creating the request
	apiPath := "/v0.1/apps/register"
	completeURL := r.endpoint + apiPath
	
	// Retrieve values from Terraform variables using ValueString() to get the actual string value
	repo := data.Repo.ValueString()
	isPublic := data.IsPublic.ValueBool()
	organizationSlug := data.OrganizationSlug.ValueString()
	repoURL := data.RepoURL.ValueString()
	typeValue := data.Type.ValueString()
	gitRepoSlug := data.GitRepoSlug.ValueString()
	gitOwner := data.GitOwner.ValueString()

	// Construct the payload data using the provided variables
	payloadData := map[string]interface{}{
		"provider":          repo,
		"is_public":         isPublic,
		"organization_slug": organizationSlug,
		"repo_url":          repoURL,
		"type":              typeValue,
		"git_repo_slug":     gitRepoSlug,
		"git_owner":         gitOwner,
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		tflog.Error(ctx, "Error marshaling payload", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}
	payload := string(payloadBytes)

	tflog.Debug(ctx, "Request payload", map[string]interface{}{"payload": payload})

	// Create an HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", completeURL, strings.NewReader(payload))
	if err != nil {
		tflog.Error(ctx, "Error creating HTTP request", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}
	// Dump the HTTP request details
	dump, _ := httputil.DumpRequest(httpReq, true)
	tflog.Debug(ctx, "HTTP Request", map[string]interface{}{"request": string(dump)})

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
	tflog.Debug(ctx, "Response body", map[string]interface{}{"body": string(responseBody)})

	// Parse the response JSON
	// Unmarshal the JSON response into the CreateResponse struct
	var jsonResponse CreateResponse
	err = json.Unmarshal(responseBody, &jsonResponse)
	if err != nil {
		tflog.Error(ctx, "Error parsing JSON response", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}

	// Print the captured slug
	tflog.Debug(ctx, "Captured app slug", map[string]interface{}{"slug": jsonResponse.Slug})
	data.AppSlug = types.StringValue(jsonResponse.Slug)

	// Debugging: Print response status and headers
	printResponseInfo(httpResp)

	if httpResp.StatusCode != http.StatusOK {
		tflog.Error(ctx, "Request did not succeed", map[string]interface{}{
			"status": httpResp.Status,
			"headers": httpResp.Header,
		})
		resp.Diagnostics.AddError("API Request Error", fmt.Sprintf("Request did not succeed: %s", httpResp.Status))
		return
	}

	// Set example ID in data
	data.Id = types.StringValue("example-id")

	tflog.Info(ctx, "Resource created successfully")

	// Update resource state with populated data
	resp.State.Set(ctx, &data)

}

func (r *AppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppResourceModel

	// Retrieve values from Terraform state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Starting AppResource Delete")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Construct the URL for deleting the request
	// Get the actual app slug value
	appSlug := data.AppSlug.ValueString()
	completeURL := fmt.Sprintf("%s/v0.1/apps/%s", r.endpoint, appSlug)

	tflog.Debug(ctx, "Delete URL", map[string]interface{}{"url": completeURL})

	// Create an HTTP request with DELETE method
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", completeURL, nil)
	if err != nil {
		tflog.Error(ctx, "Error creating HTTP request to delete APP", map[string]interface{}{"error": err.Error()})
		return
	}

	// Send the HTTP request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		tflog.Error(ctx, "Error sending HTTP request", map[string]interface{}{"error": err.Error()})
		return
	}
	defer httpResp.Body.Close()

	// // Debugging: Print response status and headers
	printResponseInfo(httpResp)

	if httpResp.StatusCode != http.StatusOK {
		tflog.Error(ctx, "Delete request did not succeed", map[string]interface{}{
			"status": httpResp.Status,
			"headers": httpResp.Header,
		})
		// Optionally, you can add diagnostics here if needed
		return
	}

	tflog.Info(ctx, "Resource deleted successfully")

	// Update resource state to indicate deletion
	resp.State.Set(ctx, cty.NilVal)
}

func (r *AppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.State.Set(ctx, &data)
}

func (r *AppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.State.Set(ctx, &data)
}

func (r *AppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func handleRequestError(err error, resp *resource.CreateResponse) {
	// Implementation for handling request error
}

func printResponseInfo(httpResp *http.Response) {
	// Implementation for printing response info
}
