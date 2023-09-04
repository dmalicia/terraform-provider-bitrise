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
	fmt.Println("Printing attributes and types in Schema method:")
	for attrName, attr := range resp.Schema.Attributes {
		fmt.Printf("AAAAAAAttribute: %s, Type: %T\n", attrName, attr)
	}
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
		fmt.Println("Provider data is missing.")
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
	fmt.Println("Provider configuration successful.")
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

	fmt.Println("Starting AppResource Create...")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Construct the cleaned URL for creating the request
	cleanedURL := strings.Trim(strings.TrimPrefix(r.endpoint, "https://"), "\"")
	// Append the API path to the cleaned URL
	apiPath := "/v0.1/apps/register"

	completeURL := cleanedURL + apiPath
	// Retrieve values from Terraform variables
	repo := data.Repo.String()
	isPublic := data.IsPublic.ValueBool()
	organizationSlug := data.OrganizationSlug.String()
	repoURL := data.RepoURL.String()
	typeValue := data.Type.String()
	gitRepoSlug := data.GitRepoSlug.String()
	gitOwner := data.GitOwner.String()

	// // Construct the payload data using the provided variables
	// payloadData := map[string]interface{}{
	// 	"provider":          repo,
	// 	"is_public":         isPublic,
	// 	"organization_slug": organizationSlug,
	// 	"repo_url":          repoURL,
	// 	"type":              typeValue,
	// 	"git_repo_slug":     gitRepoSlug,
	// 	"git_owner":         gitOwner,
	// }

	// // Remove the extra quotation marks from string values
	// for key, value := range payloadData {
	// 	if stringValue, ok := value.(string); ok {
	// 		payloadData[key] = strings.Trim(stringValue, "\"")
	// 	}
	// }

	payload := fmt.Sprintf(`{
"provider": %s,
"is_public": %t,
"organization_slug": %s,
"repo_url": %s,
"type": %s,
"git_repo_slug": %s,
"git_owner": %s
}`, repo, isPublic, organizationSlug, repoURL, typeValue, gitRepoSlug, gitOwner)

	fmt.Println("Payload:", payload)

	// Create an HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", completeURL, strings.NewReader(payload))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		handleRequestError(err, resp)
		return
	}
	// Dump the HTTP request details
	dump, _ := httputil.DumpRequest(httpReq, true)
	fmt.Println("HTTP Request Dump:")
	fmt.Println(string(dump))

	// Send the HTTP request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		handleRequestError(err, resp)
		return
	}
	defer httpResp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		handleRequestError(err, resp)
		return
	}
	fmt.Println("Response Body:", string(responseBody))

	// Parse the response JSON
	// Unmarshal the JSON response into the CreateResponse struct
	var jsonResponse CreateResponse
	err = json.Unmarshal(responseBody, &jsonResponse)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		handleRequestError(err, resp)
		return
	}

	// Print the captured slug
	fmt.Println("Captured Slug:", jsonResponse.Slug)
	data.AppSlug = types.StringValue(jsonResponse.Slug)

	// Debugging: Print response status and headers
	printResponseInfo(httpResp)

	if httpResp.StatusCode != http.StatusOK {
		fmt.Println("Request did not succeed:", httpResp.Status)
		fmt.Println("Response Headers:")
		for key, values := range httpResp.Header {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		resp.Diagnostics.AddError("API Request Error", fmt.Sprintf("Request did not succeed: %s", httpResp.Status))
		return
	}

	// Set example ID in data
	data.Id = types.StringValue("example-id")

	fmt.Println("Resource created successfully")

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

	fmt.Println("Starting AppResource Delete...")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Construct the cleaned URL for deleting the request
	cleanedURL := strings.Trim(strings.TrimPrefix(r.endpoint, "https://"), "\"")

	// Remove the surrounding double quotes from the appSlug
	appSlug := strings.Trim(data.AppSlug.String(), `"`)
	completeURL := fmt.Sprintf("%s/v0.1/apps/%s", cleanedURL, appSlug)

	fmt.Println("URL to delete:", completeURL)

	// Create an HTTP request with DELETE method
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", completeURL, nil)
	if err != nil {
		fmt.Println("Error creating HTTP request to delete APP:", err)
		return
	}

	// Send the HTTP request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return
	}
	defer httpResp.Body.Close()

	// // Debugging: Print response status and headers
	printResponseInfo(httpResp)

	if httpResp.StatusCode != http.StatusOK {
		fmt.Println("Request did not succeed:", httpResp.Status)
		fmt.Println("Response Headers:")
		for key, values := range httpResp.Header {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		// Optionally, you can add diagnostics here if needed
		return
	}

	fmt.Println("Resource deleted successfully")

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

func handleResponseStatus(httpResp *http.Response, resp *resource.CreateResponse) {
	// Implementation for handling response status
}
