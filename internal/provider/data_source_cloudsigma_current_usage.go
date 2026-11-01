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
	_ datasource.DataSource              = (*currentUsageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*currentUsageDataSource)(nil)
)

type currentUsageDataSource struct {
	clients *ProviderClients
}

type currentUsageDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func NewCurrentUsageDataSource() datasource.DataSource {
	return &currentUsageDataSource{}
}

func (d *currentUsageDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_current_usage"
}

func (d *currentUsageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the account's dynamic current usage and balance document as JSON.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Stable identity for the current usage document.",
				Computed:            true,
			},
			"json": schema.StringAttribute{
				MarkdownDescription: "The complete current usage document as canonical JSON.",
				Computed:            true,
			},
		},
	}
}

func (d *currentUsageDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *currentUsageDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	tflog.Debug(ctx, "Fetching CloudSigma current usage")
	usage, err := d.clients.Native.CurrentUsage(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma current usage", err.Error())
		return
	}
	encoded, err := json.Marshal(usage)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma current usage", err.Error())
		return
	}
	data := currentUsageDataSourceModel{ID: types.StringValue("currentusage"), JSON: types.StringValue(string(encoded))}
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
