package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
	AppSlug                          types.String `tfsdk:"app_slug"`
	AuthSSHPrivateKey                types.String `tfsdk:"auth_ssh_private_key"`
	AuthSSHPublicKey                 types.String `tfsdk:"auth_ssh_public_key"`
	IsRegisterKeyIntoProviderService types.Bool   `tfsdk:"is_register_key_into_provider_service"`
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

// Create registers the SSH key.
func (r *AppSSHResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppSSHResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.register(ctx, data); err != nil {
		appCallDiags("Unable to create request: %s", "Unable to register SSH key: %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

	if !readAppExists(ctx, r.client, data.AppSlug.ValueString(), resp) {
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

	if err := r.register(ctx, data); err != nil {
		appCallDiags("Unable to create request: %s", "Unable to update SSH key: %s").add(&resp.Diagnostics, err)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ImportState imports the resource by app slug.
func (r *AppSSHResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("app_slug"), req, resp)
}

func (r *AppSSHResource) register(ctx context.Context, data AppSSHResourceModel) error {
	appSlug := data.AppSlug.ValueString()
	return r.client.Do(ctx, bitriseapi.Request{
		Op:     "register SSH key of app " + appSlug,
		Method: http.MethodPost,
		Path:   bitriseapi.AppSSHKeyPath(appSlug),
		Body: Payload{
			AuthSSHPrivateKey:                data.AuthSSHPrivateKey.ValueString(),
			AuthSSHPublicKey:                 data.AuthSSHPublicKey.ValueString(),
			IsRegisterKeyIntoProviderService: data.IsRegisterKeyIntoProviderService.ValueBool(),
		},
	}, &bitriseapi.RawResponse{})
}
