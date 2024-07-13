package provider

import (
	"context"
	"net/http"

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
	_ resource.Resource                = (*tagResource)(nil)
	_ resource.ResourceWithConfigure   = (*tagResource)(nil)
	_ resource.ResourceWithImportState = (*tagResource)(nil)
)

// tagResource is the tag resource implementation.
type tagResource struct {
	client *cloudsigma.Client
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

func (r *tagResource) Configure(_ context.Context, request resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	r.client = request.ProviderData.(*cloudsigma.Client)
}

func (r *tagResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data tagResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &cloudsigma.TagCreateRequest{
		Tags: []cloudsigma.Tag{{
			Name: data.Name.ValueString(),
		}},
	}
	tflog.Trace(ctx, "Creating tag", map[string]interface{}{"payload": createRequest})
	tags, _, err := r.client.Tags.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create tag", err.Error())
		return
	}
	tag := tags[0]
	tflog.Trace(ctx, "Created tag", map[string]interface{}{"data": tag})

	// map response body to attributes
	data.ID = types.StringValue(tag.UUID)
	data.Name = types.StringValue(tag.Name)
	data.ResourceURI = types.StringValue(tag.ResourceURI)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *tagResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data tagResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tagUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Getting tag", map[string]interface{}{"tag_uuid": tagUUID})
	tag, resp, err := r.client.Tags.Get(ctx, tagUUID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the tag is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get tag", err.Error())
		return
	}
	tflog.Trace(ctx, "Got tag", map[string]interface{}{"data": tag})

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
	updateRequest := &cloudsigma.TagUpdateRequest{
		Tag: &cloudsigma.Tag{
			Name: data.Name.ValueString(),
		},
	}
	tflog.Trace(ctx, "Updating tag", map[string]interface{}{
		"payload":  updateRequest,
		"tag_uuid": tagUUID},
	)
	tag, _, err := r.client.Tags.Update(ctx, tagUUID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update tag", err.Error())
		return
	}
	tflog.Trace(ctx, "Updated tag", map[string]interface{}{"data": tag})

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
	tflog.Trace(ctx, "Deleting tag", map[string]interface{}{"tag_uuid": tagUUID})
	_, err := r.client.Tags.Delete(ctx, tagUUID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete tag", err.Error())
		return
	}
	tflog.Trace(ctx, "Deleted tag", map[string]interface{}{"tag_uuid": tagUUID})
}

func (r *tagResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
