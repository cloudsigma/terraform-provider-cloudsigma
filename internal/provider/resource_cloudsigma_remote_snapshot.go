package provider

import (
	"context"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*remoteSnapshotResource)(nil)
var _ resource.ResourceWithConfigure = (*remoteSnapshotResource)(nil)
var _ resource.ResourceWithImportState = (*remoteSnapshotResource)(nil)

// remoteSnapshotResource is retained as an unregistered draft until the API
// documents read, delete, and import semantics. The provider must not expose
// a managed resource whose lifecycle methods fabricate those operations.
type remoteSnapshotResource struct{ native *cloudsigmaapi.Client }
type remoteSnapshotModel struct {
	Drive       types.String `tfsdk:"drive"`
	Location    types.String `tfsdk:"location"`
	ID          types.String `tfsdk:"id"`
	ResourceURI types.String `tfsdk:"resource_uri"`
	Status      types.String `tfsdk:"status"`
}

func NewRemoteSnapshotResource() resource.Resource { return &remoteSnapshotResource{} }
func (r *remoteSnapshotResource) Metadata(context.Context, resource.MetadataRequest, *resource.MetadataResponse) {
}
func (r *remoteSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, out *resource.SchemaResponse) {
	out.Schema = schema.Schema{MarkdownDescription: "Creates a remote snapshot. The API contract currently provides create acknowledgement only; read/update/delete are not admitted until documented.", Attributes: map[string]schema.Attribute{
		"drive": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "location": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "id": schema.StringAttribute{Computed: true}, "resource_uri": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true}}}
}
func (r *remoteSnapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*ProviderClients)
	if !ok || c == nil || c.Native == nil || c.Auth != authBasic {
		resp.Diagnostics.AddError("Native client unavailable", "remote snapshots require provider basic-auth native transport")
		return
	}
	r.native = c.Native
}
func (r *remoteSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var d remoteSnapshotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &d)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.native == nil {
		resp.Diagnostics.AddError("Native client unavailable", "remote snapshot create requires native transport")
		return
	}
	v, err := r.native.CreateRemoteSnapshot(ctx, cloudsigmaapi.RemoteSnapshotCreate{Drive: d.Drive.ValueString(), Location: d.Location.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create remote snapshot", err.Error())
		return
	}
	d.ID = types.StringValue(v.UUID)
	if v.ResourceURI != "" {
		d.ResourceURI = types.StringValue(v.ResourceURI)
	}
	if v.Location != "" {
		d.Location = types.StringValue(v.Location)
	}
	if v.Status != "" {
		d.Status = types.StringValue(v.Status)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &d)...)
}
func (*remoteSnapshotResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {}
func (*remoteSnapshotResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Remote snapshot update unsupported", "the API contract does not define update semantics")
}
func (*remoteSnapshotResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Remote snapshot delete unsupported", "the API contract does not define delete semantics")
}
func (*remoteSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
