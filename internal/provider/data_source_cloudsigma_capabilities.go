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
	_ datasource.DataSource              = (*capabilitiesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*capabilitiesDataSource)(nil)
)

type capabilitiesDataSource struct {
	clients *ProviderClients
}

type capabilitiesDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func NewCapabilitiesDataSource() datasource.DataSource {
	return &capabilitiesDataSource{}
}

func (d *capabilitiesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_capabilities"
}

func (d *capabilitiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the deployment-specific CloudSigma capabilities document. The JSON field is dynamic and account/location scoped.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Stable identity for the current capabilities document.",
				Computed:            true,
			},
			"json": schema.StringAttribute{
				MarkdownDescription: "The complete dynamic capabilities document as canonical JSON.",
				Computed:            true,
			},
		},
	}
}

func (d *capabilitiesDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *capabilitiesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	tflog.Debug(ctx, "Fetching CloudSigma capabilities")
	capabilities, err := d.clients.Native.Capabilities(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma capabilities", err.Error())
		return
	}
	encoded, err := json.Marshal(capabilities)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma capabilities", err.Error())
		return
	}
	data := capabilitiesDataSourceModel{ID: types.StringValue("capabilities"), JSON: types.StringValue(string(encoded))}
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
