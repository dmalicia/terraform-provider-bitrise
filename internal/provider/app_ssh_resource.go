package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-bitrise/internal/bitriseapi"
)

var (
	_ resource.ResourceWithConfigure   = &AppSSHResource{}
	_ resource.ResourceWithImportState = &AppSSHResource{}
)

// Payload is the request body of the SSH key registration endpoint.
type Payload struct {
	AuthSSHPrivateKey                string `json:"auth_ssh_private_key"`
	AuthSSHPublicKey                 string `json:"auth_ssh_public_key"`
	IsRegisterKeyIntoProviderService bool   `json:"is_register_key_into_provider_service"`
}

// AppSSHResource registers the SSH key a Bitrise app uses to clone its repository.
type AppSSHResource struct {
	resourceWithClient
}

// NewAppSSHResource returns a new bitrise_app_ssh resource.
func NewAppSSHResource() resource.Resource {
	return &AppSSHResource{}
}

// AppSSHResourceModel describes the bitrise_app_ssh resource data model.
type AppSSHResourceModel struct {
	AppSlug                          string `tfsdk:"app_slug"`
	AuthSSHPrivateKey                string `tfsdk:"auth_ssh_private_key"`
	AuthSSHPublicKey                 string `tfsdk:"auth_ssh_public_key"`
	IsRegisterKeyIntoProviderService bool   `tfsdk:"is_register_key_into_provider_service"`
}

// Metadata returns the resource type name.
func (r *AppSSHResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_ssh"
}

// Schema returns the resource schema.
func (r *AppSSHResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

// sshKeyFile is the scratch file the private key round-trips through.
const sshKeyFile = "testtfkey"

// Create registers the SSH key.
func (r *AppSSHResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppSSHResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.WriteFile(sshKeyFile, []byte(data.AuthSSHPrivateKey), 0o644); err != nil {
		tflog.Error(ctx, "Error writing to file", map[string]interface{}{"error": err.Error()})
		return
	}
	privateKey, err := os.ReadFile(sshKeyFile)
	if err != nil {
		tflog.Error(ctx, "Error reading private key file", map[string]interface{}{"error": err.Error()})
		return
	}

	err = r.register(ctx, data, string(privateKey))
	if err != nil {
		// Pre-existing behaviour: only rejected statuses are reported.
		var callErr *bitriseapi.CallError
		var statusErr *bitriseapi.StatusError
		if !errors.As(err, &callErr) && errors.As(err, &statusErr) {
			resp.Diagnostics.AddError("API Request Error", fmt.Sprintf("Request did not succeed: %s", statusErr.Status()))
		}
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if err := os.Remove(sshKeyFile); err != nil {
		tflog.Error(ctx, "Error deleting tfkey file", map[string]interface{}{"error": err.Error()})
	}
}

// Delete only removes the resource from state; the API has no endpoint to unregister the key.
func (r *AppSSHResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppSSHResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.RemoveResource(ctx)
}

// Read keeps the resource in state while the app exists.
func (r *AppSSHResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppSSHResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !readAppExists(ctx, r.client, data.AppSlug, resp) {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update registers the planned SSH key.
func (r *AppSSHResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppSSHResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.WriteFile(sshKeyFile, []byte(data.AuthSSHPrivateKey), 0o644); err != nil {
		resp.Diagnostics.AddError("File Error", fmt.Sprintf("Unable to write private key: %s", err))
		return
	}
	defer func() { _ = os.Remove(sshKeyFile) }()
	privateKey, err := os.ReadFile(sshKeyFile)
	if err != nil {
		resp.Diagnostics.AddError("File Error", fmt.Sprintf("Unable to read private key: %s", err))
		return
	}

	if err := r.register(ctx, data, string(privateKey)); err != nil {
		appCallDiags("Unable to create request: %s", "Unable to update SSH key: %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ImportState imports the resource by ID.
func (r *AppSSHResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AppSSHResource) register(ctx context.Context, data AppSSHResourceModel, privateKey string) error {
	payload := Payload{
		AuthSSHPrivateKey:                privateKey,
		AuthSSHPublicKey:                 data.AuthSSHPublicKey,
		IsRegisterKeyIntoProviderService: data.IsRegisterKeyIntoProviderService,
	}
	payloadJSON, _ := json.Marshal(payload)
	tflog.Debug(ctx, "MODULEDEBUG: SSH key payload prepared", map[string]interface{}{"payload_json": string(payloadJSON)})

	return r.client.Do(ctx, bitriseapi.Request{
		Op:     "register SSH key of app " + data.AppSlug,
		Method: http.MethodPost,
		Path:   bitriseapi.AppSSHKeyPath(data.AppSlug),
		Body:   payload,
	}, &bitriseapi.RawResponse{})
}
