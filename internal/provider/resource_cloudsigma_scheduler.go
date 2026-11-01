package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

var (
	_ resource.Resource                = (*schedulerResource)(nil)
	_ resource.ResourceWithConfigure   = (*schedulerResource)(nil)
	_ resource.ResourceWithImportState = (*schedulerResource)(nil)
)

type schedulerResource struct{ native *cloudsigmaapi.Client }

type schedulerResourceModel struct {
	ID             types.String `tfsdk:"id"`
	DefinitionJSON types.String `tfsdk:"definition_json"`
	JSON           types.String `tfsdk:"json"`
}

func NewSchedulerResource() resource.Resource { return &schedulerResource{} }

func (r *schedulerResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "cloudsigma_scheduler"
}

func (r *schedulerResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Manages a CloudSigma backup scheduler using the pinned native scheduler contract. `definition_json` must be the complete writable scheduler projection.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Computed: true},
			"definition_json": schema.StringAttribute{Optional: true, Computed: true},
			"json":            schema.StringAttribute{Computed: true},
		},
	}
}

func (r *schedulerResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func schedulerDefinition(raw string) (cloudsigmaapi.SchedulerCreateRequest, error) {
	var definition cloudsigmaapi.SchedulerCreateRequest
	if err := json.Unmarshal([]byte(raw), &definition); err != nil {
		return definition, err
	}
	return definition, nil
}

func schedulerProjection(raw json.RawMessage) (string, error) {
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return "", err
	}
	for _, key := range []string{"uuid", "owner", "resource_uri", "permissions", "grantees", "permission", "backup_scheduler"} {
		delete(object, key)
	}
	encoded, err := json.Marshal(object)
	return string(encoded), err
}

func (r *schedulerResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data schedulerResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() || data.DefinitionJSON.IsNull() || data.DefinitionJSON.IsUnknown() {
		response.Diagnostics.AddError("Invalid scheduler definition", "definition_json must contain the complete writable scheduler projection.")
		return
	}
	definition, err := schedulerDefinition(data.DefinitionJSON.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Invalid scheduler definition", err.Error())
		return
	}
	created, err := r.native.CreateBackupScheduler(ctx, definition)
	if err != nil {
		response.Diagnostics.AddError("Unable to create scheduler", err.Error())
		return
	}
	data.ID = types.StringValue(created.UUID)
	data.JSON = types.StringValue(string(mustJSON(created)))
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *schedulerResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data schedulerResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	raw, err := r.native.BackupScheduler(ctx, data.ID.ValueString())
	if err != nil {
		var apiErr *cloudsigmaapi.Error
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read scheduler", err.Error())
		return
	}
	projection, err := schedulerProjection(raw)
	if err != nil {
		response.Diagnostics.AddError("Unable to project scheduler state", err.Error())
		return
	}
	data.JSON = types.StringValue(string(raw))
	data.DefinitionJSON = types.StringValue(projection)
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *schedulerResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data schedulerResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	var prior schedulerResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &prior)...)
	if response.Diagnostics.HasError() || data.DefinitionJSON.IsNull() || data.DefinitionJSON.IsUnknown() {
		response.Diagnostics.AddError("Invalid scheduler definition", "definition_json must contain the complete writable scheduler projection.")
		return
	}
	definition, err := schedulerDefinition(data.DefinitionJSON.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Invalid scheduler definition", err.Error())
		return
	}
	updated, err := r.native.UpdateBackupScheduler(ctx, prior.ID.ValueString(), definition)
	if err != nil {
		response.Diagnostics.AddError("Unable to update scheduler", err.Error())
		return
	}
	data.ID = prior.ID
	data.JSON = types.StringValue(string(mustJSON(updated)))
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *schedulerResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	if r.native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data schedulerResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	if err := r.native.DeleteBackupScheduler(ctx, data.ID.ValueString()); err != nil {
		response.Diagnostics.AddError("Unable to delete scheduler", err.Error())
	}
}

func (r *schedulerResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func mustJSON(value any) []byte { b, _ := json.Marshal(value); return b }
