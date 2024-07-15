package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
)

var (
	_ datasource.DataSource              = (*locationDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*locationDataSource)(nil)
)

// locationDataSource is the location data source implementation.
type locationDataSource struct {
	client *cloudsigma.Client
}

type locationDataSourceModel struct {
	APIEndpoint types.String            `tfsdk:"api_endpoint"`
	CountryCode types.String            `tfsdk:"country_code"`
	DisplayName types.String            `tfsdk:"display_name"`
	Filters     []migration.FilterModel `tfsdk:"filter"`
	ID          types.String            `tfsdk:"id"`
	UUID        types.String            `tfsdk:"uuid"`
}

func NewLocationDataSource() datasource.DataSource {
	return &locationDataSource{}
}

func (d *locationDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_location"
}

func (d *locationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The location data source provides information about an existing CloudSigma location.
`,
		Attributes: map[string]schema.Attribute{
			"api_endpoint": schema.StringAttribute{
				MarkdownDescription: "The API endpoint of the location.",
				Computed:            true,
			},
			"country_code": schema.StringAttribute{
				MarkdownDescription: "The location country code.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The human readable name of the location.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the location.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current location, equal to ID.",
				Computed:            true,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SetNestedBlock{
				MarkdownDescription: "One or more name/value pairs to filter off of.",
				DeprecationMessage:  `Configure "uuid" instead. The "filter" block will be removed in a future version of the provider.`,
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

func (d *locationDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *locationDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data locationDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Filters != nil && len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block", map[string]interface{}{"filters_count": len(data.Filters)})

		tflog.Trace(ctx, "Getting locations")
		locations, _, err := d.client.Locations.List(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get locations", err.Error())
			return
		}
		tflog.Trace(ctx, "Got locations", map[string]interface{}{"locations_count": len(locations)})

		tflog.Debug(ctx, "Converting locations for filtering")
		locationsForFilters := migration.ToAnySlice(locations)
		tflog.Debug(ctx, "Converted locations for filtering", map[string]interface{}{"locations": len(locationsForFilters)})
		filteredLocations, diags := migration.ApplyFilter(data.Filters, locationsForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredLocations) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v locations.", len(filteredLocations)),
			)
			return
		}
		if len(filteredLocations) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		location := filteredLocations[0].(cloudsigma.Location)

		data.APIEndpoint = types.StringValue(location.APIEndpoint)
		data.CountryCode = types.StringValue(location.CountryCode)
		data.DisplayName = types.StringValue(location.DisplayName)
		data.ID = types.StringValue(location.ID)
		data.UUID = types.StringValue(location.ID)
	} else {
		locationUUID := data.UUID.ValueString()

		if locationUUID == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "uuid" must be defined.`,
			)
			return
		}

		tflog.Trace(ctx, "Getting locations")
		locations, _, err := d.client.Locations.List(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get locations", err.Error())
			return
		}
		tflog.Trace(ctx, "Got locations", map[string]interface{}{"data": locations})

		locationFound := false
		tflog.Debug(ctx, "Searching for location UUID", map[string]interface{}{"location_uuid": locationUUID})
		for _, location := range locations {
			if locationUUID == location.ID {
				data.APIEndpoint = types.StringValue(location.APIEndpoint)
				data.CountryCode = types.StringValue(location.CountryCode)
				data.DisplayName = types.StringValue(location.DisplayName)
				data.ID = types.StringValue(location.ID)
				data.UUID = types.StringValue(location.ID)

				locationFound = true
				break
			}
		}

		if !locationFound {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
