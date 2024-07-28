package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/drive"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/snapshot"
)

var (
	_ resource.Resource                = (*snapshotResource)(nil)
	_ resource.ResourceWithConfigure   = (*snapshotResource)(nil)
	_ resource.ResourceWithImportState = (*snapshotResource)(nil)
)

// snapshotResource is the snapshot resource implementation.
type snapshotResource struct {
	client *cloudsigma.Client
}

// snapshotResourceModel maps the snapshot resource schema data.
type snapshotResourceModel struct {
	Drive       types.String `tfsdk:"drive"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	ResourceURI types.String `tfsdk:"resource_uri"`
	Status      types.String `tfsdk:"status"`
	Timestamp   types.String `tfsdk:"timestamp"`
	UUID        types.String `tfsdk:"uuid"`
}

func NewSnapshotResource() resource.Resource {
	return &snapshotResource{}
}

func (r *snapshotResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cloudsigma_snapshot"
}

func (r *snapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The snapshot resource allows you to manage CloudSigma snapshots.

Snapshots are point-in-time versions of a drive. They can be cloned to a full drive,
which makes it possible to restore an older version of a VM image.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"drive": schema.StringAttribute{
				MarkdownDescription: "The UUID of the drive.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the snapshot.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the snapshot.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_uri": schema.StringAttribute{
				MarkdownDescription: "The unique resource identifier of the snapshot.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the snapshot.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				}},
			"timestamp": schema.StringAttribute{
				MarkdownDescription: "The timestamp of the snapshot creation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the snapshot, equal to ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *snapshotResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*cloudsigma.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unconfigured CloudSigma client",
			"Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

func (r *snapshotResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data snapshotResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Checking for drive to be mounted or unmounted")
	driveUUID := data.Drive.ValueString()
	err := drive.WaitDriveStatusMountedOrUnmounted(ctx, r.client, driveUUID)
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid drive status",
			fmt.Sprintf("Drive status must be 'mounted' or 'unmounted': %v", err),
		)
		return
	}

	createRequest := &cloudsigma.SnapshotCreateRequest{
		Snapshots: []cloudsigma.Snapshot{{
			Drive: &cloudsigma.Drive{UUID: driveUUID},
			Name:  data.Name.ValueString(),
		}},
	}
	tflog.Trace(ctx, "Creating snapshot", map[string]any{"payload": createRequest})
	snapshots, _, err := r.client.Snapshots.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create snapshot", err.Error())
		return
	}
	snap := &snapshots[0]
	tflog.Trace(ctx, "Created snapshot", map[string]any{"data": snap})

	tflog.Info(ctx, "Waiting for snapshot to be available")
	err = snapshot.WaitSnapshotStatusAvailable(ctx, r.client, snap.UUID)
	if err != nil {
		response.Diagnostics.AddError(
			"Invalid snapshot status",
			fmt.Sprintf("Snapshot status must be 'available': %v", err.Error()),
		)
		return
	}

	snap, _, err = r.client.Snapshots.Get(ctx, snap.UUID)
	if err != nil {
		response.Diagnostics.AddError("Unable to create snapshot", err.Error())
		return
	}

	// map response body to attributes
	data.Drive = types.StringValue(snap.Drive.UUID)
	data.Name = types.StringValue(snap.Name)
	data.ID = types.StringValue(snap.UUID)
	data.ResourceURI = types.StringValue(snap.ResourceURI)
	data.Status = types.StringValue(snap.Status)
	data.Timestamp = types.StringValue(snap.Timestamp)
	data.UUID = types.StringValue(snap.UUID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *snapshotResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data snapshotResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	snapshotUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Getting snapshot", map[string]any{"snapshot_uuid": snapshotUUID})
	snap, resp, err := r.client.Snapshots.Get(ctx, snapshotUUID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the tag is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get snapshot", err.Error())
		return
	}
	tflog.Trace(ctx, "Got snapshot", map[string]any{"data": snap})

	// map response body to attributes
	data.Drive = types.StringValue(snap.Drive.UUID)
	data.Name = types.StringValue(snap.Name)
	data.ID = types.StringValue(snap.UUID)
	data.ResourceURI = types.StringValue(snap.ResourceURI)
	data.Status = types.StringValue(snap.Status)
	data.Timestamp = types.StringValue(snap.Timestamp)
	data.UUID = types.StringValue(snap.UUID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *snapshotResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data snapshotResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	snapshotUUID := data.ID.ValueString()
	updateRequest := &cloudsigma.SnapshotUpdateRequest{
		Snapshot: &cloudsigma.Snapshot{
			Drive: &cloudsigma.Drive{UUID: data.Drive.ValueString()},
			Name:  data.Name.ValueString(),
		},
	}
	tflog.Trace(ctx, "Updating snapshot", map[string]any{
		"payload":       updateRequest,
		"snapshot_uuid": snapshotUUID,
	})
	snap, _, err := r.client.Snapshots.Update(ctx, snapshotUUID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update snapshot", err.Error())
		return
	}
	tflog.Trace(ctx, "Updated snapshot", map[string]any{"data": snap})

	// map response body to attributes
	data.Drive = types.StringValue(snap.Drive.UUID)
	data.Name = types.StringValue(snap.Name)
	data.ID = types.StringValue(snap.UUID)
	data.ResourceURI = types.StringValue(snap.ResourceURI)
	data.Status = types.StringValue(snap.Status)
	data.Timestamp = types.StringValue(snap.Timestamp)
	data.UUID = types.StringValue(snap.UUID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *snapshotResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data snapshotResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	snapshotUUID := data.ID.ValueString()
	tflog.Trace(ctx, "Deleting snapshot", map[string]any{"snapshot_uuid": snapshotUUID})
	_, err := r.client.Snapshots.Delete(ctx, snapshotUUID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete snapshot", err.Error())
		return
	}
	tflog.Trace(ctx, "Deleted snapshot", map[string]any{"snapshot_uuid": snapshotUUID})
}

func (r *snapshotResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
