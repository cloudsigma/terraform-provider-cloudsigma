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
	_ datasource.DataSource              = (*licenseDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*licenseDataSource)(nil)
)

// licenseDataSource is the license data source implementation.
type licenseDataSource struct {
	client *cloudsigma.Client
}

// licenseDataSourceModel maps the license data source schema data.
type licenseDataSourceModel struct {
	Burstable   types.Bool              `tfsdk:"burstable"`
	Filters     []migration.FilterModel `tfsdk:"filter"`
	ID          types.String            `tfsdk:"id"`
	LongName    types.String            `tfsdk:"long_name"`
	Name        types.String            `tfsdk:"name"`
	ResourceURI types.String            `tfsdk:"resource_uri"`
	Type        types.String            `tfsdk:"type"`
	UserMetric  types.String            `tfsdk:"user_metric"`
	UUID        types.String            `tfsdk:"uuid"`
}

func NewLicenseDataSource() datasource.DataSource {
	return &licenseDataSource{}
}

func (d *licenseDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_license"
}

func (d *licenseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The license data source provides information about an existing CloudSigma license.
`,
		Attributes: map[string]schema.Attribute{
			"burstable": schema.BoolAttribute{
				MarkdownDescription: "`true`, if the license can be used on burst, otherwise `false`.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the license.",
				Computed:            true,
			},
			"long_name": schema.StringAttribute{
				MarkdownDescription: "The human-readable name of the license.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name that should be used when purchasing the license.",
				Computed:            true,
				Optional:            true,
			},
			"resource_uri": schema.StringAttribute{
				MarkdownDescription: "The unique resource identifier of the license.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of billing of the license.",
				Computed:            true,
			},
			"user_metric": schema.StringAttribute{
				MarkdownDescription: "The metric that the user is charged for.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current license, equal to ID.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SetNestedBlock{
				MarkdownDescription: "One or more name/value pairs to filter off of.",
				DeprecationMessage:  `Configure "name" instead. The "filter" block will be removed in a future version of the provider.`,
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

func (d *licenseDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *licenseDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data licenseDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Filters != nil && len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block", map[string]interface{}{"filters_count": len(data.Filters)})

		tflog.Trace(ctx, "Getting licenses")
		licenses, _, err := d.client.Licenses.List(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get licenses", err.Error())
			return
		}
		tflog.Trace(ctx, "Got licenses", map[string]interface{}{"licenses_count": len(licenses)})

		tflog.Debug(ctx, "Converting licenses for filtering")
		licensesForFilters := migration.ToAnySlice(licenses)
		tflog.Debug(ctx, "Converted licenses for filtering", map[string]interface{}{"licenses": licensesForFilters})
		filteredLicenses, diags := migration.ApplyFilter(data.Filters, licensesForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredLicenses) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v licenses.", len(filteredLicenses)),
			)
			return
		}
		if len(filteredLicenses) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		license := filteredLicenses[0].(cloudsigma.License)

		data.Burstable = types.BoolValue(license.Burstable)
		data.ID = types.StringValue(license.Name)
		data.LongName = types.StringValue(license.LongName)
		data.Name = types.StringValue(license.Name)
		data.ResourceURI = types.StringValue(license.ResourceURI)
		data.Type = types.StringValue(license.Type)
		data.UserMetric = types.StringValue(license.UserMetric)
		data.UUID = types.StringValue(license.Name)
	} else {
		licenseName := data.Name.ValueString()

		if licenseName == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "name" must be defined.`,
			)
			return
		}

		tflog.Trace(ctx, "Getting licenses for filtering", map[string]interface{}{"license_name": licenseName})
		licenses, _, err := d.client.Licenses.List(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get licenses", err.Error())
			return
		}
		tflog.Trace(ctx, "Got licenses", map[string]interface{}{"data": licenses})

		licenseFound := false
		for _, license := range licenses {
			if licenseName == license.Name {
				data.Burstable = types.BoolValue(license.Burstable)
				data.ID = types.StringValue(license.Name)
				data.LongName = types.StringValue(license.LongName)
				data.Name = types.StringValue(license.Name)
				data.ResourceURI = types.StringValue(license.ResourceURI)
				data.Type = types.StringValue(license.Type)
				data.UserMetric = types.StringValue(license.UserMetric)
				data.UUID = types.StringValue(license.Name)

				licenseFound = true
				break
			}
		}

		if !licenseFound {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
