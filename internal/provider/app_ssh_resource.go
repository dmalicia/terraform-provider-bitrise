package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/zclconf/go-cty/cty"
)

type Payload struct {
	AuthSSHPrivateKey                string `json:"auth_ssh_private_key"`
	AuthSSHPublicKey                 string `json:"auth_ssh_public_key"`
	IsRegisterKeyIntoProviderService bool   `json:"is_register_key_into_provider_service"`
}

type AppSSHResource struct {
	clientCreator func(endpoint, token string) *http.Client
	endpoint      string
	token         string
}

func NewAppSSHResource(clientCreator func(endpoint, token string) *http.Client, endpoint, token string) *AppSSHResource {
	return &AppSSHResource{
		clientCreator: clientCreator,
		endpoint:      endpoint,
		token:         token,
	}
}

type AppSSHResourceModel struct {
	AppSlug                          string `tfsdk:"app_slug"`
	AuthSSHPrivateKey                string `tfsdk:"auth_ssh_private_key"`
	AuthSSHPublicKey                 string `tfsdk:"auth_ssh_public_key"`
	IsRegisterKeyIntoProviderService bool   `tfsdk:"is_register_key_into_provider_service"`
}

func (r *AppSSHResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_ssh"
}

func (r *AppSSHResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Bitrise App SSH resource",
		Attributes: map[string]schema.Attribute{
			"app_slug": schema.StringAttribute{
				MarkdownDescription: "App Slug",
				Required:            true,
			},
			"auth_ssh_private_key": schema.StringAttribute{
				MarkdownDescription: "Private SSH key for authentication",
				Required:            true,
			},
			"auth_ssh_public_key": schema.StringAttribute{
				MarkdownDescription: "Public SSH key for authentication",
				Required:            true,
			},
			"is_register_key_into_provider_service": schema.BoolAttribute{
				MarkdownDescription: "Whether to register the public key into the provider service",
				Optional:            true,
			},
		},
	}
}

func (r *AppSSHResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppSSHResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Initialize data to store API response
	var data AppSSHResourceModel

	// Populate data from Terraform plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	// Check for errors in diagnostics and return if found
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Starting AppSSHResource Create")

	// Create an HTTP client using the client creator from the provider
	client := r.clientCreator(r.endpoint, r.token)

	// Construct the URL for creating the request
	appSlug := data.AppSlug
	completeURL := fmt.Sprintf("%s/v0.1/apps/%s/register-ssh-key", r.endpoint, appSlug)

	filePath := "testtfkey"
	fileContent := []byte(data.AuthSSHPrivateKey)

	err := os.WriteFile(filePath, fileContent, 0644)
	if err != nil {
		tflog.Error(ctx, "Error writing to file", map[string]interface{}{"error": err.Error()})
		return
	}

	// Construct the payload data using the provided variables
	privateKeyPath := "testtfkey"
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		tflog.Error(ctx, "Error reading private key file", map[string]interface{}{"error": err.Error()})
		return
	}
	privateKey := string(privateKeyBytes)

	tflog.Debug(ctx, "Variable content written to file", map[string]interface{}{"file": filePath})

	payload := Payload{
		AuthSSHPrivateKey:                privateKey,
		AuthSSHPublicKey:                 data.AuthSSHPublicKey,
		IsRegisterKeyIntoProviderService: data.IsRegisterKeyIntoProviderService,
	}

	// Marshal the payload struct into a JSON string
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		tflog.Error(ctx, "Error marshaling JSON payload", map[string]interface{}{"error": err.Error()})
		handleRequestError(err, resp)
		return
	}

	tflog.Debug(ctx, "SSH key payload prepared", map[string]interface{}{"payload_json": string(payloadJSON)})

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

	tflog.Info(ctx, "SSH key registration completed successfully")

	// Update resource state with populated data
	resp.State.Set(ctx, &data)
	// Delete the tfkey file
        err = os.Remove(filePath)
        if err != nil {
	  tflog.Error(ctx, "Error deleting tfkey file", map[string]interface{}{"error": err.Error()})
	  // Handle the error if needed
        }
}

func (r *AppSSHResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppSSHResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Set(ctx, cty.NilVal)
}

func (r *AppSSHResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppSSHResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read App, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.State.Set(ctx, &data)
}

func (r *AppSSHResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppSSHResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update App, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.State.Set(ctx, &data)
}

func (r *AppSSHResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
