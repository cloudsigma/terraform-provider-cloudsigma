package provider

import (
	"context"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

var (
	_ resource.Resource                = (*sshKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*sshKeyResource)(nil)
	_ resource.ResourceWithImportState = (*sshKeyResource)(nil)
)

// sshKeyResource is the SSH key resource implementation.
type sshKeyResource struct {
	client *cloudsigma.Client
}

// sshKeyResourceModel maps the SSH key resource schema data.
type sshKeyResourceModel struct {
	Name       types.String `tfsdk:"name"`
	ID         types.String `tfsdk:"id"`
	PrivateKey types.String `tfsdk:"private_key"`
	PublicKey  types.String `tfsdk:"public_key"`
	UUID       types.String `tfsdk:"uuid"`
}

func NewSSHKeyResource() resource.Resource {
	return &sshKeyResource{}
}

func (r *sshKeyResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cloudsigma_ssh_key"
}

func (r *sshKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The SSH key resource allows you to manage CloudSigma SSH keys.

Keys created with this resource can be referenced in your server
configuration via their IDs.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The SSH key name.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the SSH key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "The private SSH key material.",
				Computed:            true,
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "The public SSH key material.",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the SSH key, equal to ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *sshKeyResource) Configure(_ context.Context, request resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	r.client = request.ProviderData.(*cloudsigma.Client)
}

func (r *sshKeyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data sshKeyResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &cloudsigma.KeypairCreateRequest{
		Keypairs: []cloudsigma.Keypair{{
			Name: data.Name.ValueString(),
		}},
	}
	privateKey := data.PrivateKey.ValueString()
	if privateKey != "" {
		createRequest.Keypairs[0].PrivateKey = privateKey
	}
	publicKey := strings.Trim(data.PublicKey.ValueString(), "\n")
	if publicKey != "" {
		createRequest.Keypairs[0].PublicKey = publicKey
	}
	tflog.Trace(ctx, "Creating SSH key", map[string]interface{}{"payload": createRequest})
	keypairs, _, err := r.client.Keypairs.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create SSH key", err.Error())
		return
	}
	keypair := keypairs[0]
	tflog.Trace(ctx, "Created SSH key", map[string]interface{}{"data": keypair})

	// map response body to attributes
	data.Name = types.StringValue(keypair.Name)
	data.ID = types.StringValue(keypair.UUID)
	data.PrivateKey = types.StringValue(keypair.PrivateKey)
	data.PublicKey = types.StringValue(keypair.PublicKey)
	data.UUID = types.StringValue(keypair.UUID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *sshKeyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data sshKeyResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	keypairUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Getting SSH key", map[string]interface{}{"ssh_key_uuid": keypairUUID})
	keypair, resp, err := r.client.Keypairs.Get(ctx, keypairUUID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the tag is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get SSH key", err.Error())
		return
	}
	tflog.Trace(ctx, "Got SSH key", map[string]interface{}{"data": keypair})

	// map response body to attributes
	data.Name = types.StringValue(keypair.Name)
	data.ID = types.StringValue(keypair.UUID)
	data.PrivateKey = types.StringValue(keypair.PrivateKey)
	data.PublicKey = types.StringValue(keypair.PublicKey)
	data.UUID = types.StringValue(keypair.UUID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *sshKeyResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan sshKeyResourceModel
	var oldPrivateKey, oldPublicKey types.String

	// read plan and state data into the model
	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	response.Diagnostics.Append(request.State.GetAttribute(ctx, path.Root("private_key"), &oldPrivateKey)...)
	response.Diagnostics.Append(request.State.GetAttribute(ctx, path.Root("public_key"), &oldPublicKey)...)
	if response.Diagnostics.HasError() {
		return
	}

	keypairUUID := plan.ID.ValueString()
	keypair := &cloudsigma.Keypair{
		Name: plan.Name.ValueString(),
		UUID: keypairUUID,
	}
	if !plan.PrivateKey.Equal(oldPrivateKey) {
		keypair.PrivateKey = plan.PrivateKey.ValueString()
	}
	if !plan.PublicKey.Equal(oldPublicKey) {
		keypair.PublicKey = plan.PublicKey.ValueString()
	}
	updateRequest := &cloudsigma.KeypairUpdateRequest{Keypair: keypair}
	tflog.Trace(ctx, "Updating SSH key", map[string]interface{}{
		"payload":      updateRequest,
		"ssh_key_uuid": keypairUUID,
	})
	updatedKeypair, _, err := r.client.Keypairs.Update(ctx, keypairUUID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update SSH key", err.Error())
		return
	}
	tflog.Trace(ctx, "Updated SSH key", map[string]interface{}{"data": updatedKeypair})

	// map response body to attributes
	plan.Name = types.StringValue(updatedKeypair.Name)
	plan.ID = types.StringValue(updatedKeypair.UUID)
	plan.PrivateKey = types.StringValue(updatedKeypair.PrivateKey)
	plan.PublicKey = types.StringValue(updatedKeypair.PublicKey)
	plan.UUID = types.StringValue(updatedKeypair.UUID)

	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

func (r *sshKeyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data sshKeyResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	keypairUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Deleting SSH key", map[string]interface{}{"ssh_key_uuid": keypairUUID})
	_, err := r.client.Keypairs.Delete(ctx, keypairUUID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete SSH key", err.Error())
		return
	}
	tflog.Trace(ctx, "Deleted SSH key", map[string]interface{}{"ssh_key_uuid": keypairUUID})
}

func (r *sshKeyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
