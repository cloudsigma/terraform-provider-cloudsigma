package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
)

var (
	_ datasource.DataSource              = (*libraryDriveDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*libraryDriveDataSource)(nil)
)

// libraryDriveDataSource is the library drive data source implementation.
type libraryDriveDataSource struct {
	client *cloudsigma.Client
}

// driveDataSourceModel maps the library drive data source schema data.
type libraryDriveDataSourceModel struct {
	Architecture types.String            `tfsdk:"arch"`
	Description  types.String            `tfsdk:"description"`
	Filters      []migration.FilterModel `tfsdk:"filter"`
	ID           types.String            `tfsdk:"id"`
	ImageType    types.String            `tfsdk:"image_type"`
	Media        types.String            `tfsdk:"media"`
	Name         types.String            `tfsdk:"name"`
	OS           types.String            `tfsdk:"os"`
	Size         types.Int64             `tfsdk:"size"`
	Status       types.String            `tfsdk:"status"`
	StorageType  types.String            `tfsdk:"storage_type"`
	UUID         types.String            `tfsdk:"uuid"`
}

func NewLibraryDriveDataSource() datasource.DataSource {
	return &libraryDriveDataSource{}
}

func (d *libraryDriveDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_library_drive"
}

func (d *libraryDriveDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
`,
		Attributes: map[string]schema.Attribute{
			"arch": schema.StringAttribute{
				MarkdownDescription: "The operating system bit architecture of the library drive.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the library drive.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the library drive.",
				Computed:            true,
			},
			"image_type": schema.StringAttribute{
				MarkdownDescription: "The image type of the library drive.",
				Computed:            true,
			},
			"media": schema.StringAttribute{
				MarkdownDescription: "The media representation type. It can be `cdrom` or `disk`.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The human-readable name of the library drive.",
				Computed:            true,
				Optional:            true,
			},
			"os": schema.StringAttribute{
				MarkdownDescription: "The operating system of the library drive.",
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "The size of the library drive in bytes.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the library drive.",
				Computed:            true,
			},
			"storage_type": schema.StringAttribute{
				MarkdownDescription: "The storage type of the library drive.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current library drive, equal to ID.",
				Computed:            true,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SetNestedBlock{
				MarkdownDescription: "One or more name/value pairs to filter off of.",
				DeprecationMessage:  `Configure "name" or "uuid" instead. The "filter" block will be removed in a future version of the provider.`,
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

func (d *libraryDriveDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *libraryDriveDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data libraryDriveDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Filters != nil && len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block", map[string]interface{}{"filters_count": len(data.Filters)})

		opts := &cloudsigma.LibraryDriveListOptions{
			ListOptions: cloudsigma.ListOptions{Limit: 0},
		}
		tflog.Trace(ctx, "Getting library drives")
		libraryDrives, _, err := d.client.LibraryDrives.List(ctx, opts)
		if err != nil {
			response.Diagnostics.AddError("Unable to get library drives", err.Error())
			return
		}
		tflog.Trace(ctx, "Got library drives", map[string]interface{}{"drives_count": len(libraryDrives)})

		tflog.Debug(ctx, "Converting library drives for filtering")
		libraryDrivesForFilters := migration.ToAnySlice(libraryDrives)
		tflog.Debug(ctx, "Converted library drives for filtering", map[string]interface{}{"library_drives": libraryDrivesForFilters})
		filteredLibraryDrives, diags := migration.ApplyFilter(data.Filters, libraryDrivesForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredLibraryDrives) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v library drives.", len(filteredLibraryDrives)),
			)
			return
		}
		if len(filteredLibraryDrives) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		libraryDrive := filteredLibraryDrives[0].(cloudsigma.LibraryDrive)

		data.Architecture = types.StringValue(libraryDrive.Arch)
		data.Description = types.StringValue(libraryDrive.Description)
		data.ID = types.StringValue(libraryDrive.UUID)
		data.ImageType = types.StringValue(libraryDrive.ImageType)
		data.Media = types.StringValue(libraryDrive.Media)
		data.Name = types.StringValue(libraryDrive.Name)
		data.OS = types.StringValue(libraryDrive.OS)
		data.Size = types.Int64Value(int64(libraryDrive.Size))
		data.Status = types.StringValue(libraryDrive.Status)
		data.StorageType = types.StringValue(libraryDrive.StorageType)
		data.UUID = types.StringValue(libraryDrive.UUID)
	} else {
		libraryDriveName := data.Name.ValueString()
		libraryDriveUUID := data.UUID.ValueString()

		if libraryDriveName == "" && libraryDriveUUID == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "name" or "uuid" must be defined.`,
			)
			return
		}

		if libraryDriveUUID != "" {
			tflog.Trace(ctx, "Getting library drive using UUID", map[string]interface{}{"library_drive_uuid": libraryDriveUUID})
			libraryDrive, resp, err := d.client.LibraryDrives.Get(ctx, libraryDriveUUID)
			if err != nil {
				if resp != nil && resp.StatusCode == http.StatusNotFound {
					response.Diagnostics.AddError("No search results", "Please refine your search.")
					return
				}
				response.Diagnostics.AddError("Unable to get library drive", err.Error())
				return
			}
			tflog.Trace(ctx, "Got library drive", map[string]interface{}{"data": libraryDrive})

			// if name is defined check that it's equal
			if libraryDriveName != "" && libraryDriveName != libraryDrive.Name {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual library drive name are different. Expected '%s', got '%s'", libraryDriveName, libraryDrive.Name),
				)
				return
			}

			data.Architecture = types.StringValue(libraryDrive.Arch)
			data.Description = types.StringValue(libraryDrive.Description)
			data.ID = types.StringValue(libraryDrive.UUID)
			data.ImageType = types.StringValue(libraryDrive.ImageType)
			data.Media = types.StringValue(libraryDrive.Media)
			data.Name = types.StringValue(libraryDrive.Name)
			data.OS = types.StringValue(libraryDrive.OS)
			data.Size = types.Int64Value(int64(libraryDrive.Size))
			data.Status = types.StringValue(libraryDrive.Status)
			data.StorageType = types.StringValue(libraryDrive.StorageType)
			data.UUID = types.StringValue(libraryDrive.UUID)
		} else {
			opts := &cloudsigma.LibraryDriveListOptions{
				ListOptions: cloudsigma.ListOptions{Limit: 0},
				Names:       []string{libraryDriveName},
			}
			tflog.Trace(ctx, "Getting library drives", map[string]interface{}{"opts": opts})
			libraryDrives, _, err := d.client.LibraryDrives.List(ctx, opts)
			if err != nil {
				response.Diagnostics.AddError("Unable to get library drives", err.Error())
				return
			}
			tflog.Trace(ctx, "Got library drives", map[string]interface{}{"data": libraryDrives})

			if len(libraryDrives) > 1 {
				response.Diagnostics.AddError(
					"Too many search results",
					fmt.Sprintf("Please refine your search to be more specific. Found %v library drives.", len(libraryDrives)),
				)
				return
			}
			if len(libraryDrives) < 1 {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}

			libraryDrive := libraryDrives[0]

			data.Architecture = types.StringValue(libraryDrive.Arch)
			data.Description = types.StringValue(libraryDrive.Description)
			data.ID = types.StringValue(libraryDrive.UUID)
			data.ImageType = types.StringValue(libraryDrive.ImageType)
			data.Media = types.StringValue(libraryDrive.Media)
			data.Name = types.StringValue(libraryDrive.Name)
			data.OS = types.StringValue(libraryDrive.OS)
			data.Size = types.Int64Value(int64(libraryDrive.Size))
			data.Status = types.StringValue(libraryDrive.Status)
			data.StorageType = types.StringValue(libraryDrive.StorageType)
			data.UUID = types.StringValue(libraryDrive.UUID)
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
