package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func (r *AppFinishResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Initialize data to store API response
	var data AppFinishResourceModel

	// Populate data from Terraform plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Check for errors in diagnostics and return if found
	if resp.Diagnostics.HasError() {
		return
	}

	fmt.Println("Starting AppFinishResource Create...")

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
		fmt.Println("Error marshaling JSON payload:", err)
		handleRequestError(err, resp)
		return
	}

	fmt.Println("DMALICIADEBUGFINISH: Payload:", payload)
	fmt.Println("DMALICIADEBUGFINISH: PayloadJSON:", string(payloadJSON))

	// Create an HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", completeURL, strings.NewReader(string(payloadJSON)))
	if err != nil {
		fmt.Println("DMALICIADEBUGFINISH: Error creating HTTP request:", err)
		handleRequestError(err, resp)
		return
	}
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Dump the HTTP request details
	dump, _ := httputil.DumpRequest(httpReq, true)
	fmt.Println("DMALICIADEBUGFINISH: HTTP Request Dump:")
	fmt.Println(string(dump))

	// Send the HTTP request
	httpResp, err := client.Do(httpReq)
	if err != nil {
		fmt.Println("DMALICIADEBUGFINISH: Error sending HTTP request:", err)
		handleRequestError(err, resp)
		return
	}
	defer httpResp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		fmt.Println("DMALICIADEBUGFINISH: Error reading response body:", err)
		handleRequestError(err, resp)
		return
	}
	fmt.Println("DMALICIADEBUGFINISH: Response Body:", string(responseBody))

	// Debugging: Print response status and headers
	printResponseInfo(httpResp)

	if httpResp.StatusCode != http.StatusOK {
		fmt.Println("DMALICIADEBUGFINISH: Request did not succeed:", httpResp.Status)
		fmt.Println("DMALICIADEBUGFINISH: Response Headers:")
		for key, values := range httpResp.Header {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
		resp.Diagnostics.AddError("API Request Error", fmt.Sprintf("DMALICIADEBUGFINISH: Request did not succeed: %s", httpResp.Status))
		return
	}

	fmt.Println("DMALICIADEBUGFINISH: App registration completed successfully")

	// Update resource state with populated data
	resp.State.Set(ctx, &data)
}

func (r *AppFinishResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Implement the read logic for the "finish" step
}

func (r *AppFinishResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Implement the update logic for the "finish" step
}

func (r *AppFinishResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Implement the delete logic for the "finish" step
}

func (r *AppFinishResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Implement the import state logic for the "finish" step
}
