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
	_ datasource.DataSource              = (*virtualRoutersDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*virtualRoutersDataSource)(nil)
)

type virtualRoutersDataSource struct {
	clients *ProviderClients
}

type virtualRoutersDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func NewVirtualRoutersDataSource() datasource.DataSource {
	return &virtualRoutersDataSource{}
}

func (d *virtualRoutersDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_virtual_routers"
}

func (d *virtualRoutersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the CloudSigma virtual-router collection as dynamic JSON. This data source does not manage router lifecycle or configuration actions.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{MarkdownDescription: "Stable identity for the current virtual-router collection.", Computed: true},
			"json": schema.StringAttribute{MarkdownDescription: "The complete dynamic virtual-router collection as canonical JSON.", Computed: true},
		},
	}
}

func (d *virtualRoutersDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *virtualRoutersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	tflog.Debug(ctx, "Fetching CloudSigma virtual routers")
	routers, err := d.clients.Native.VirtualRouters(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma virtual routers", err.Error())
		return
	}
	encoded, err := json.Marshal(routers)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma virtual routers", err.Error())
		return
	}
	response.Diagnostics.Append(response.State.Set(ctx, &virtualRoutersDataSourceModel{ID: types.StringValue("virtual-routers"), JSON: types.StringValue(string(encoded))})...)
}
