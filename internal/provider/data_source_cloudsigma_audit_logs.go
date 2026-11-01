package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*auditLogsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*auditLogsDataSource)(nil)
)

type auditLogsDataSource struct{ clients *ProviderClients }

type auditLogsDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	JSON  types.String `tfsdk:"json"`
	Limit types.Int64  `tfsdk:"limit"`
}

func NewAuditLogsDataSource() datasource.DataSource { return &auditLogsDataSource{} }

func (d *auditLogsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_audit_logs"
}

func (d *auditLogsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the CloudSigma audit log collection as dynamic JSON.",
		Attributes: map[string]schema.Attribute{
			"id":    schema.StringAttribute{Computed: true},
			"json":  schema.StringAttribute{Computed: true, Sensitive: true},
			"limit": schema.Int64Attribute{Optional: true},
		},
	}
}

func (d *auditLogsDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *auditLogsDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data auditLogsDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
	limit := int64(0)
	if !data.Limit.IsNull() && !data.Limit.IsUnknown() {
		limit = data.Limit.ValueInt64()
	}
	document, err := d.clients.Native.AuditLogs(ctx, limit)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma audit logs", err.Error())
		return
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma audit logs", err.Error())
		return
	}
	data.ID = types.StringValue("audit_logs")
	data.JSON = types.StringValue(string(encoded))
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
