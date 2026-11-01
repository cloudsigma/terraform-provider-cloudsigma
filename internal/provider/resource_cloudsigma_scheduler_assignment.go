package provider

import (
	"context"
	"errors"
	"net/http"
	"sync"

	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

var (
	_ resource.Resource                = (*schedulerAssignmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*schedulerAssignmentResource)(nil)
	_ resource.ResourceWithImportState = (*schedulerAssignmentResource)(nil)
)

type schedulerAssignmentResource struct{ native *cloudsigmaapi.Client }

type schedulerAssignmentModel struct {
	ID            types.String `tfsdk:"id"`
	DriveUUID     types.String `tfsdk:"drive_uuid"`
	SchedulerUUID types.String `tfsdk:"scheduler_uuid"`
}

var schedulerAssignmentMu sync.Mutex

func NewSchedulerAssignmentResource() resource.Resource { return &schedulerAssignmentResource{} }

func (r *schedulerAssignmentResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cloudsigma_scheduler_assignment"
}
func (r *schedulerAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{MarkdownDescription: "Owns only the drive-to-scheduler association. Scheduler and drive resources do not write this association.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true},
		"drive_uuid": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		}},
		"scheduler_uuid": schema.StringAttribute{Required: true},
	}}
}
func (r *schedulerAssignmentResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func driveSchedulerUUID(drive *cloudsigmaapi.Drive) (string, error) {
	if drive == nil {
		return "", &cloudsigmaapi.Error{Category: "decode_error"}
	}
	return cloudsigmaapi.SchedulerAssociationUUID(drive.BackupScheduler)
}

func (r *schedulerAssignmentResource) reconcile(ctx context.Context, driveUUID, schedulerUUID string) error {
	schedulerAssignmentMu.Lock()
	defer schedulerAssignmentMu.Unlock()
	drive, err := r.native.Drive(ctx, driveUUID)
	if err != nil {
		return err
	}
	current, err := driveSchedulerUUID(drive)
	if err != nil {
		return err
	}
	if current == schedulerUUID {
		return nil
	}
	_, actionErr := r.native.AssignDriveScheduler(ctx, driveUUID, schedulerUUID)
	if actionErr != nil {
		// The request outcome is uncertain. Read before any retry; never replay
		// the 202 action blindly.
		observed, readErr := r.native.Drive(ctx, driveUUID)
		if readErr == nil {
			association, projectionErr := driveSchedulerUUID(observed)
			if projectionErr == nil && association == schedulerUUID {
				return nil
			}
		}
		return actionErr
	}
	observed, err := r.native.Drive(ctx, driveUUID)
	if err != nil {
		return err
	}
	association, err := driveSchedulerUUID(observed)
	if err != nil {
		return err
	}
	if association != schedulerUUID {
		return &cloudsigmaapi.Error{Category: "reconciliation"}
	}
	return nil
}

func (r *schedulerAssignmentResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data schedulerAssignmentModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	if err := r.reconcile(ctx, data.DriveUUID.ValueString(), data.SchedulerUUID.ValueString()); err != nil {
		response.Diagnostics.AddError("Unable to assign scheduler", err.Error())
		return
	}
	data.ID = types.StringValue(data.DriveUUID.ValueString())
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
func (r *schedulerAssignmentResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data schedulerAssignmentModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	drive, err := r.native.Drive(ctx, data.DriveUUID.ValueString())
	if err != nil {
		var apiErr *cloudsigmaapi.Error
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read scheduler assignment", err.Error())
		return
	}
	observed, err := driveSchedulerUUID(drive)
	if err != nil {
		response.Diagnostics.AddError("Unable to read scheduler assignment", err.Error())
		return
	}
	if observed == "" {
		response.State.RemoveResource(ctx)
		return
	}
	if data.SchedulerUUID.IsNull() || data.SchedulerUUID.IsUnknown() {
		data.SchedulerUUID = types.StringValue(observed)
		data.ID = types.StringValue(data.DriveUUID.ValueString())
		response.Diagnostics.Append(response.State.Set(ctx, &data)...)
		return
	}
	// Preserve the observed association in state so Terraform can plan the
	// configured replacement; refresh never writes a competing association.
	data.SchedulerUUID = types.StringValue(observed)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
func (r *schedulerAssignmentResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data, prior schedulerAssignmentModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	response.Diagnostics.Append(request.State.Get(ctx, &prior)...)
	if response.Diagnostics.HasError() {
		return
	}
	if data.DriveUUID.IsNull() || data.DriveUUID.IsUnknown() || !data.DriveUUID.Equal(prior.DriveUUID) {
		response.Diagnostics.AddError("Drive change requires replacement", "Replace the scheduler assignment to change drive_uuid so the previous drive association is removed.")
		return
	}
	r.Create(ctx, resource.CreateRequest{Plan: request.Plan}, (*resource.CreateResponse)(response))
}
func (r *schedulerAssignmentResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data schedulerAssignmentModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	schedulerAssignmentMu.Lock()
	defer schedulerAssignmentMu.Unlock()
	drive, err := r.native.Drive(ctx, data.DriveUUID.ValueString())
	if err != nil {
		var apiErr *cloudsigmaapi.Error
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return
		}
		response.Diagnostics.AddError("Unable to read scheduler assignment", err.Error())
		return
	}
	observed, err := driveSchedulerUUID(drive)
	if err != nil {
		response.Diagnostics.AddError("Unable to read scheduler assignment", err.Error())
		return
	}
	if observed == "" || observed != data.SchedulerUUID.ValueString() {
		return
	}
	if _, err = r.native.UnassignDriveScheduler(ctx, data.DriveUUID.ValueString(), data.SchedulerUUID.ValueString()); err != nil {
		// An uncertain outcome or 404 is only success after a fresh authorized
		// read verifies that this association is gone. Never remove a replacement.
		var apiErr *cloudsigmaapi.Error
		if errors.As(err, &apiErr) && (apiErr.Status == http.StatusNotFound || apiErr.Category == "transport" || apiErr.Category == "timeout" || apiErr.Category == "cancelled") {
			final, readErr := r.native.Drive(ctx, data.DriveUUID.ValueString())
			if readErr == nil {
				finalAssociation, projectionErr := driveSchedulerUUID(final)
				if projectionErr == nil && finalAssociation != data.SchedulerUUID.ValueString() {
					return
				}
			}
			if readErr != nil {
				var finalErr *cloudsigmaapi.Error
				if errors.As(readErr, &finalErr) && finalErr.Status == http.StatusNotFound {
					return
				}
			}
		}
		response.Diagnostics.AddError("Unable to unassign scheduler", err.Error())
		return
	}
	final, err := r.native.Drive(ctx, data.DriveUUID.ValueString())
	if err != nil {
		var apiErr *cloudsigmaapi.Error
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			return
		}
		response.Diagnostics.AddError("Unable to verify scheduler unassignment", err.Error())
		return
	}
	finalAssociation, err := driveSchedulerUUID(final)
	if err != nil {
		response.Diagnostics.AddError("Unable to verify scheduler unassignment", err.Error())
		return
	}
	if finalAssociation == data.SchedulerUUID.ValueString() {
		response.Diagnostics.AddError("Unable to verify scheduler unassignment", "The drive still has a scheduler association.")
	}
}
func (r *schedulerAssignmentResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	response.Diagnostics.Append(response.State.SetAttribute(ctx, tfpath.Root("id"), request.ID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, tfpath.Root("drive_uuid"), request.ID)...)
}
