package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

var (
	_ resource.Resource                = (*tagResource)(nil)
	_ resource.ResourceWithConfigure   = (*tagResource)(nil)
	_ resource.ResourceWithImportState = (*tagResource)(nil)
)

// tagResource is the tag resource implementation.
type tagResource struct {
	native *cloudsigmaapi.Client
}

// tagResourceModel maps the tag resource schema data.
type tagResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ResourceURI types.String `tfsdk:"resource_uri"`
}

func NewTagResource() resource.Resource {
	return &tagResource{}
}

func (r *tagResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cloudsigma_tag"
}

func (r *tagResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The tag resource allows you to manage CloudSigma tags.

A tag is a label that can be applied to a CloudSigma resource in order to better organize or
facilitate the lookups and actions on it. Tags created with this resource can be referenced
in your configurations via their IDs.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the tag.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The tag name.",
				Required:            true,
			},
			"resource_uri": schema.StringAttribute{
				MarkdownDescription: "The unique resource identifier of the tag.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *tagResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	r.native = nil
	if request.ProviderData == nil {
		return
	}
	clients, ok := request.ProviderData.(*ProviderClients)
	if !ok || clients == nil || clients.Native == nil || (clients.Auth != authBasic && clients.Auth != authToken) {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	r.native = clients.Native
}

func (r *tagResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data tagResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "Creating tag")
	tag, err := r.native.CreateTag(ctx, cloudsigmaapi.TagCreateRequest{Name: data.Name.ValueString()})
	if err != nil {
		response.Diagnostics.AddError("Unable to create tag", err.Error())
		return
	}
	tflog.Trace(ctx, "Created tag")

	// map response body to attributes
	data.ID = types.StringValue(tag.UUID)
	data.Name = types.StringValue(tag.Name)
	data.ResourceURI = types.StringValue(tag.ResourceURI)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *tagResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data tagResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tagUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Getting tag")
	tag, err := r.native.Tag(ctx, tagUUID)
	if err != nil {
		var apiErr *cloudsigmaapi.Error
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			// if the tag is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get tag", err.Error())
		return
	}
	tflog.Trace(ctx, "Got tag")

	// map response body to attributes
	data.ID = types.StringValue(tag.UUID)
	data.Name = types.StringValue(tag.Name)
	data.ResourceURI = types.StringValue(tag.ResourceURI)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *tagResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data tagResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tagUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Updating tag")
	tag, err := r.native.UpdateTag(ctx, tagUUID, cloudsigmaapi.TagUpdateRequest{Name: data.Name.ValueString()})
	if err != nil {
		response.Diagnostics.AddError("Unable to update tag", err.Error())
		return
	}
	tflog.Trace(ctx, "Updated tag")

	// map response body to attributes
	data.ID = types.StringValue(tag.UUID)
	data.Name = types.StringValue(tag.Name)
	data.ResourceURI = types.StringValue(tag.ResourceURI)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *tagResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data tagResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tagUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Deleting tag")
	err := r.native.DeleteTag(ctx, tagUUID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete tag", err.Error())
		return
	}
	tflog.Trace(ctx, "Deleted tag")
}

func (r *tagResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
