package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
)

var (
	_ datasource.DataSource              = (*driveDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*driveDataSource)(nil)
)

// driveDataSource is the drive data source implementation.
type driveDataSource struct {
	clients *ProviderClients
}

// driveDataSourceModel maps the drive data source schema data.
type driveDataSourceModel struct {
	Filters     []migration.FilterModel `tfsdk:"filter"`
	ID          types.String            `tfsdk:"id"`
	Name        types.String            `tfsdk:"name"`
	Size        types.Int64             `tfsdk:"size"`
	Status      types.String            `tfsdk:"status"`
	StorageType types.String            `tfsdk:"storage_type"`
	UUID        types.String            `tfsdk:"uuid"`
}

func NewDriveDataSource() datasource.DataSource {
	return &driveDataSource{}
}

func (d *driveDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_drive"
}

func (d *driveDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The drive data source provides information about an existing CloudSigma drive.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the drive.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The human readable name of the drive.",
				Computed:            true,
				Optional:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "The size of the drive in bytes.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the drive.",
				Computed:            true,
			},
			"storage_type": schema.StringAttribute{
				MarkdownDescription: "The storage type of the drive.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current drive, equal to ID.",
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

func (d *driveDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *driveDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data driveDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Filters != nil && len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block")

		opts := &cloudsigma.DriveListOptions{
			ListOptions: cloudsigma.ListOptions{Limit: 0},
		}
		tflog.Trace(ctx, "Getting drives")
		drives, err := d.drives(ctx, opts)
		if err != nil {
			response.Diagnostics.AddError("Unable to get drives", err.Error())
			return
		}
		tflog.Trace(ctx, "Got drives")

		tflog.Debug(ctx, "Converting drives for filtering")
		drivesForFilters := migration.ToAnySlice(drives)
		tflog.Debug(ctx, "Converted drives for filtering")
		filteredDrives, diags := migration.ApplyFilter(data.Filters, drivesForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredDrives) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v drives.", len(filteredDrives)),
			)
			return
		}
		if len(filteredDrives) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		drive := filteredDrives[0].(cloudsigmaapi.Drive)

		data.ID = types.StringValue(drive.UUID)
		data.Name = types.StringValue(drive.Name)
		data.Size = types.Int64Value(int64(drive.Size))
		data.Status = types.StringValue(drive.Status)
		data.StorageType = types.StringValue(drive.StorageType)
		data.UUID = types.StringValue(drive.UUID)
	} else {
		driveName := data.Name.ValueString()
		driveUUID := data.UUID.ValueString()

		if driveName == "" && driveUUID == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "name" or "uuid" must be defined.`,
			)
			return
		}

		if driveUUID != "" {
			tflog.Trace(ctx, "Getting drive using UUID")
			drive, status, err := d.drive(ctx, driveUUID)
			if err != nil {
				if status == http.StatusNotFound {
					response.Diagnostics.AddError("No search results", "Please refine your search.")
					return
				}
				response.Diagnostics.AddError("Unable to get drive", err.Error())
				return
			}
			tflog.Trace(ctx, "Got drive")

			// if name is defined check that it's equal
			if driveName != "" && driveName != drive.Name {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual drive name are different. Expected '%s', got '%s'", driveName, drive.Name),
				)
				return
			}

			data.ID = types.StringValue(drive.UUID)
			data.Name = types.StringValue(drive.Name)
			data.Size = types.Int64Value(int64(drive.Size))
			data.Status = types.StringValue(drive.Status)
			data.StorageType = types.StringValue(drive.StorageType)
			data.UUID = types.StringValue(drive.UUID)
		} else {
			opts := &cloudsigma.DriveListOptions{
				ListOptions: cloudsigma.ListOptions{Limit: 0},
				Names:       []string{driveName},
			}
			tflog.Trace(ctx, "Getting drives")
			drives, err := d.drives(ctx, opts)
			if err != nil {
				response.Diagnostics.AddError("Unable to get drives", err.Error())
				return
			}
			tflog.Trace(ctx, "Got drives")

			if len(drives) > 1 {
				response.Diagnostics.AddError(
					"Too many search results",
					fmt.Sprintf("Please refine your search to be more specific. Found %v drives.", len(drives)),
				)
				return
			}
			if len(drives) < 1 {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}

			drive := drives[0]

			data.ID = types.StringValue(drive.UUID)
			data.Name = types.StringValue(drive.Name)
			data.Size = types.Int64Value(int64(drive.Size))
			data.Status = types.StringValue(drive.Status)
			data.StorageType = types.StringValue(drive.StorageType)
			data.UUID = types.StringValue(drive.UUID)
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

// Authentication mode is selected at Configure, never by retrying an error.
func (d *driveDataSource) drives(ctx context.Context, opts *cloudsigma.DriveListOptions) ([]cloudsigmaapi.Drive, error) {
	if d.clients.Auth == authBasic {
		name := ""
		if len(opts.Names) > 0 {
			name = opts.Names[0]
		}
		return d.clients.Native.Drives(ctx, name)
	}
	v, _, e := d.clients.Legacy.Drives.List(ctx, opts)
	if e != nil {
		return nil, e
	}
	var out []cloudsigmaapi.Drive
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	e = json.Unmarshal(b, &out)
	return out, e
}
func (d *driveDataSource) drive(ctx context.Context, uuid string) (*cloudsigmaapi.Drive, int, error) {
	if d.clients.Auth == authBasic {
		v, e := d.clients.Native.Drive(ctx, uuid)
		var apiErr *cloudsigmaapi.Error
		if errors.As(e, &apiErr) {
			return v, apiErr.Status, e
		}
		return v, 0, e
	}
	v, r, e := d.clients.Legacy.Drives.Get(ctx, uuid)
	status := 0
	if r != nil {
		status = r.StatusCode
	}
	if e != nil {
		return nil, status, e
	}
	var out cloudsigmaapi.Drive
	b, e := json.Marshal(v)
	if e != nil {
		return nil, status, e
	}
	e = json.Unmarshal(b, &out)
	return &out, status, e
}
