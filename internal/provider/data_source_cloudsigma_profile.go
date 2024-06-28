package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

var (
	_ datasource.DataSource              = (*profileDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*profileDataSource)(nil)
)

// profileDataSource is the profile data source implementation.
type profileDataSource struct {
	client *cloudsigma.Client
}

// profileDataSourceModel maps the profile data source schema data.
type profileDataSourceModel struct {
	Address   types.String `tfsdk:"address"`
	Company   types.String `tfsdk:"company"`
	FirstName types.String `tfsdk:"first_name"`
	ID        types.String `tfsdk:"id"`
	LastName  types.String `tfsdk:"last_name"`
	Title     types.String `tfsdk:"title"`
	UUID      types.String `tfsdk:"uuid"`
}

func NewProfileDataSource() datasource.DataSource {
	return &profileDataSource{}
}

func (d *profileDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_profile"
}

func (d *profileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "The profile data source provides information about an existing CloudSigma user profile.",
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "The address of the user.",
				Computed:            true,
			},
			"company": schema.StringAttribute{
				MarkdownDescription: "The company name of the user.",
				Computed:            true,
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "The first name of the user.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the profile.",
				Computed:            true,
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "The last name of the user.",
				Computed:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "The title of the user.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current user profile, equal to ID.",
				Computed:            true,
			},
		},
	}
}

func (d *profileDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.client = request.ProviderData.(*cloudsigma.Client)
}

func (d *profileDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	var data profileDataSourceModel

	tflog.Debug(ctx, "Fetching user profile")
	profile, _, err := d.client.Profile.Get(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch user profile", err.Error())
		return
	}
	tflog.Debug(ctx, "Fetched user profile", map[string]interface{}{
		"data": profile,
	})

	data.Address = types.StringValue(profile.Address)
	data.Company = types.StringValue(profile.Company)
	data.FirstName = types.StringValue(profile.FirstName)
	data.ID = types.StringValue(profile.UUID)
	data.LastName = types.StringValue(profile.LastName)
	data.Title = types.StringValue(profile.Title)
	data.UUID = types.StringValue(profile.UUID)

	diags := response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
