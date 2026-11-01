package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*backupSchedulersDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*backupSchedulersDataSource)(nil)
)

type backupSchedulersDataSource struct{ clients *ProviderClients }

type backupSchedulersDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func NewBackupSchedulersDataSource() datasource.DataSource { return &backupSchedulersDataSource{} }

func (d *backupSchedulersDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_backup_schedulers"
}

func (d *backupSchedulersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the GET-only CloudSigma backup scheduler collection as canonical JSON. This data source does not manage scheduler lifecycle or drive assignments.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true},
			"json": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *backupSchedulersDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.clients = nil
	if request.ProviderData == nil {
		return
	}
	clients, ok := request.ProviderData.(*ProviderClients)
	if !ok || clients == nil || clients.Native == nil || (clients.Auth != authBasic && clients.Auth != authToken) {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	d.clients = clients
}

func (d *backupSchedulersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	tflog.Debug(ctx, "Fetching CloudSigma backup schedulers")
	schedulers, err := d.clients.Native.BackupSchedulers(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma backup schedulers", err.Error())
		return
	}
	encoded, err := json.Marshal(schedulers)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma backup schedulers", err.Error())
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &backupSchedulersDataSourceModel{ID: types.StringValue("backup-schedulers"), JSON: types.StringValue(string(encoded))})...)
}
