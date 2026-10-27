package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
)

var (
	_ datasource.DataSource              = (*ipDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*ipDataSource)(nil)
)

// ipDataSource is the IP data source implementation.
type ipDataSource struct {
	clients *ProviderClients
}

// ipDataSourceModel maps the IP data source schema data.
type ipDataSourceModel struct {
	Filters     []migration.FilterModel `tfsdk:"filter"`
	Gateway     types.String            `tfsdk:"gateway"`
	ID          types.String            `tfsdk:"id"`
	Netmask     types.Int64             `tfsdk:"netmask"`
	ResourceURI types.String            `tfsdk:"resource_uri"`
	UUID        types.String            `tfsdk:"uuid"`
}

func NewIPDataSource() datasource.DataSource {
	return &ipDataSource{}
}

func (d *ipDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_ip"
}

func (d *ipDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The ip data source provides information about an existing CloudSigma IP address.
`,
		Attributes: map[string]schema.Attribute{
			"gateway": schema.StringAttribute{
				MarkdownDescription: "Default gateway for the IP address.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the IP address.",
				Computed:            true,
			},
			"netmask": schema.Int64Attribute{
				MarkdownDescription: "Netmask value in CIDR notation.",
				Computed:            true,
			},
			"resource_uri": schema.StringAttribute{
				MarkdownDescription: "The unique resource identifier of the IP address.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current IP address, equal to ID.",
				Computed:            true,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SetNestedBlock{
				MarkdownDescription: "One or more name/value pairs to filter off of.",
				DeprecationMessage:  `Configure "uuid" or "name" instead. The "filter" block will be removed in a future version of the provider.`,
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the attribute to filter.",
							Optional:            true,
						},
						"values": schema.ListAttribute{
							MarkdownDescription: "The value of the attribute to filter.",
							ElementType:         types.StringType,
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ipDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.clients = nil
	if request.ProviderData == nil {
		return
	}
	clients, ok := request.ProviderData.(*ProviderClients)
	if !ok || clients == nil || (clients.Auth != authBasic && clients.Auth != authToken) || clients.Native == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	d.clients = clients
}

func (d *ipDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data ipDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block")

		tflog.Trace(ctx, "Getting IPs")
		ips, err := d.ips(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get IPs", err.Error())
			return
		}
		tflog.Trace(ctx, "Got IPs")

		tflog.Debug(ctx, "Converting IPs for filtering")
		ipsForFilters := migration.ToAnySlice(ips)
		tflog.Debug(ctx, "Converted IPs for filtering")
		filteredIPs, diags := migration.ApplyFilter(data.Filters, ipsForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredIPs) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v IPs.", len(filteredIPs)),
			)
			return
		}
		if len(filteredIPs) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		ip := filteredIPs[0].(cloudsigmaapi.IP)

		data.Gateway = types.StringValue(ip.Gateway)
		data.ID = types.StringValue(ip.UUID)
		data.Netmask = types.Int64Value(int64(ip.Netmask))
		data.ResourceURI = types.StringValue(ip.ResourceURI)
		data.UUID = types.StringValue(ip.UUID)
	} else {
		ipUUID := data.UUID.ValueString()

		if ipUUID == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "uuid" must be defined.`,
			)
			return
		}

		tflog.Trace(ctx, "Getting IP using UUID")
		ip, err := d.ip(ctx, ipUUID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get IP", err.Error())
			return
		}
		tflog.Trace(ctx, "Got IP")

		data.Gateway = types.StringValue(ip.Gateway)
		data.ID = types.StringValue(ip.UUID)
		data.Netmask = types.Int64Value(int64(ip.Netmask))
		data.ResourceURI = types.StringValue(ip.ResourceURI)
		data.UUID = types.StringValue(ip.UUID)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

// Authentication mode is selected at Configure, never by retrying an error.
func (d *ipDataSource) ips(ctx context.Context) ([]cloudsigmaapi.IP, error) {
	return d.clients.Native.IPs(ctx)
}
func (d *ipDataSource) ip(ctx context.Context, id string) (*cloudsigmaapi.IP, error) {
	return d.clients.Native.IP(ctx, id)
}
