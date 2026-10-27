package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
)

var (
	_ datasource.DataSource              = (*vlanDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vlanDataSource)(nil)
)

// vlanDataSource is the VLAN data source implementation.
type vlanDataSource struct {
	clients *ProviderClients
}

// vlanDataSourceModel maps the VLAN data source schema data.
type vlanDataSourceModel struct {
	Filters     []migration.FilterModel `tfsdk:"filter"`
	ID          types.String            `tfsdk:"id"`
	Name        types.String            `tfsdk:"name"`
	ResourceURI types.String            `tfsdk:"resource_uri"`
	UUID        types.String            `tfsdk:"uuid"`
}

func NewVLANDataSource() datasource.DataSource {
	return &vlanDataSource{}
}

func (d *vlanDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_vlan"
}

func (d *vlanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The vlan data source provides information about an existing CloudSigma VLAN.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the VLAN.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VLAN.",
				Computed:            true,
				Optional:            true,
			},
			"resource_uri": schema.StringAttribute{
				MarkdownDescription: "The unique resource identifier of the VLAN.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current VLAN, equal to ID.",
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

func (d *vlanDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	d.clients = nil
	if request.ProviderData == nil {
		return
	}
	clients, ok := request.ProviderData.(*ProviderClients)
	if !ok || clients == nil || (clients.Auth != authBasic && clients.Auth != authToken) || (clients.Auth == authBasic && clients.Native == nil) || (clients.Auth == authToken && clients.Legacy == nil) {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	d.clients = clients
}

func (d *vlanDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data vlanDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block")

		tflog.Trace(ctx, "Getting VLANs")
		vlans, err := d.vlans(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get VLANs", err.Error())
			return
		}
		tflog.Trace(ctx, "Got VLANs")

		tflog.Debug(ctx, "Converting VLANs for filtering")
		vlansForFilters := migration.ToAnySlice(vlans)
		tflog.Debug(ctx, "Converted VLANs")
		filteredVLANs, diags := migration.ApplyFilter(data.Filters, vlansForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredVLANs) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v VLANs.", len(filteredVLANs)),
			)
			return
		}
		if len(filteredVLANs) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		vlan := filteredVLANs[0].(cloudsigmaapi.VLAN)

		data.ID = types.StringValue(vlan.UUID)
		data.Name = types.StringValue(getVLANName(vlan))
		data.ResourceURI = types.StringValue(vlan.ResourceURI)
		data.UUID = types.StringValue(vlan.UUID)
	} else {
		vlanUUID := data.UUID.ValueString()
		vlanName := data.Name.ValueString()

		if vlanUUID == "" && vlanName == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "name" or "uuid" must be defined.`,
			)
			return
		}

		if vlanUUID != "" {
			tflog.Trace(ctx, "Getting VLAN using UUID")
			vlan, err := d.vlan(ctx, vlanUUID)
			if err != nil {
				response.Diagnostics.AddError("Unable to get VLAN", err.Error())
				return
			}
			tflog.Trace(ctx, "Got VLAN")

			// if name is defined check that it's equal
			if vlanName != "" && vlanName != getVLANName(*vlan) {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual VLAN name are different. Expected '%s', got '%s'", vlanName, getVLANName(*vlan)),
				)
				return
			}

			data.ID = types.StringValue(vlan.UUID)
			data.Name = types.StringValue(getVLANName(*vlan))
			data.ResourceURI = types.StringValue(vlan.ResourceURI)
			data.UUID = types.StringValue(vlan.UUID)
		} else {
			tflog.Trace(ctx, "Getting VLANs for filtering")
			vlans, err := d.vlans(ctx)
			if err != nil {
				response.Diagnostics.AddError("Unable to get VLANs", err.Error())
				return
			}
			tflog.Trace(ctx, "Got VLANs")

			vlanFound := false
			for _, vlan := range vlans {
				if vlanName == getVLANName(vlan) {
					data.ID = types.StringValue(vlan.UUID)
					data.Name = types.StringValue(getVLANName(vlan))
					data.ResourceURI = types.StringValue(vlan.ResourceURI)
					data.UUID = types.StringValue(vlan.UUID)

					vlanFound = true
					break
				}
			}

			if !vlanFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func getVLANName(vlan cloudsigmaapi.VLAN) string {
	name, ok := vlan.Meta["name"]
	if ok {
		return fmt.Sprintf("%s", name)
	} else {
		return ""
	}
}

// Authentication mode is selected at Configure, never by retrying an error.
func (d *vlanDataSource) vlans(ctx context.Context) ([]cloudsigmaapi.VLAN, error) {
	if d.clients.Auth == authBasic {
		return d.clients.Native.VLANs(ctx)
	}
	v, _, e := d.clients.Legacy.VLANs.List(ctx)
	if e != nil {
		return nil, e
	}
	var out []cloudsigmaapi.VLAN
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(b, &out)
	return out, e
}
func (d *vlanDataSource) vlan(ctx context.Context, id string) (*cloudsigmaapi.VLAN, error) {
	if d.clients.Auth == authBasic {
		return d.clients.Native.VLAN(ctx, id)
	}
	v, _, e := d.clients.Legacy.VLANs.Get(ctx, id)
	if e != nil {
		return nil, e
	}
	var out cloudsigmaapi.VLAN
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(b, &out)
	return &out, e
}
