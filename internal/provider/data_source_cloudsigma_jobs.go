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
	_ datasource.DataSource              = (*jobsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*jobsDataSource)(nil)
)

type jobsDataSource struct {
	clients *ProviderClients
}

type jobsDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func NewJobsDataSource() datasource.DataSource {
	return &jobsDataSource{}
}

func (d *jobsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_jobs"
}

func (d *jobsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the CloudSigma asynchronous Jobs collection as dynamic JSON. Jobs are read-only and may include recursive child jobs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Stable identity for the current Jobs collection.",
				Computed:            true,
			},
			"json": schema.StringAttribute{
				MarkdownDescription: "The complete Jobs collection as canonical JSON.",
				Computed:            true,
			},
		},
	}
}

func (d *jobsDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *jobsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	tflog.Debug(ctx, "Fetching CloudSigma jobs")
	jobs, err := d.clients.Native.Jobs(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma jobs", err.Error())
		return
	}
	encoded, err := json.Marshal(jobs)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma jobs", err.Error())
		return
	}
	data := jobsDataSourceModel{ID: types.StringValue("jobs"), JSON: types.StringValue(string(encoded))}
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}
