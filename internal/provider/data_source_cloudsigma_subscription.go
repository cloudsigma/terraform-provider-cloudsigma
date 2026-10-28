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
	_ datasource.DataSource              = (*subscriptionDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*subscriptionDataSource)(nil)
)

// subscriptionDataSource is the subscription data source implementation.
type subscriptionDataSource struct {
	clients *ProviderClients
}

// subscriptionDataSourceModel maps the subscription data source schema data.
type subscriptionDataSourceModel struct {
	Amount      types.String            `tfsdk:"amount"`
	AutoRenew   types.Bool              `tfsdk:"auto_renew"`
	Filters     []migration.FilterModel `tfsdk:"filter"`
	FreeTier    types.Bool              `tfsdk:"free_tier"`
	ID          types.String            `tfsdk:"id"`
	Period      types.String            `tfsdk:"period"`
	Price       types.String            `tfsdk:"price"`
	Remaining   types.String            `tfsdk:"remaining"`
	Resource    types.String            `tfsdk:"resource"`
	ResourceURI types.String            `tfsdk:"resource_uri"`
	Status      types.String            `tfsdk:"status"`
	UUID        types.String            `tfsdk:"uuid"`
}

func NewSubscriptionDataSource() datasource.DataSource {
	return &subscriptionDataSource{}
}

func (d *subscriptionDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_subscription"
}

func (d *subscriptionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The subscription data source provides information about an existing CloudSigma subscription.
`,
		Attributes: map[string]schema.Attribute{
			"amount": schema.StringAttribute{
				MarkdownDescription: "The amount of the subscription.",
				Computed:            true,
			},
			"auto_renew": schema.BoolAttribute{
				MarkdownDescription: "`true`, if the subscription will auto renew on expire, otherwise `false`.",
				Computed:            true,
			},
			"free_tier": schema.BoolAttribute{
				MarkdownDescription: "`true`, if the subscription is in free tier, otherwise `false`.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the subscription.",
				Computed:            true,
			},
			"period": schema.StringAttribute{
				MarkdownDescription: "The duration of the subscription.",
				Computed:            true,
			},
			"price": schema.StringAttribute{
				MarkdownDescription: "The price of the subscription.",
				Computed:            true,
			},
			"remaining": schema.StringAttribute{
				MarkdownDescription: "The amount remaining.",
				Computed:            true,
			},
			"resource": schema.StringAttribute{
				MarkdownDescription: "The name of resource associated with the subscription.",
				Computed:            true,
			},
			"resource_uri": schema.StringAttribute{
				MarkdownDescription: "The unique resource identifier of the subscription.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the subscription.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique universal identifier of the current subscription.",
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

func (d *subscriptionDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *subscriptionDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var data subscriptionDataSourceModel

	// read state data into the model
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(data.Filters) > 0 {
		// this logic belongs to deprecated filter block and should be removed after breaking change release
		tflog.Warn(ctx, "Using legacy filter block")

		tflog.Trace(ctx, "Getting subscriptions")
		subscriptions, err := d.subscriptions(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get subscriptions", err.Error())
			return
		}
		tflog.Trace(ctx, "Got subscriptions")

		tflog.Debug(ctx, "Converting subscriptions for filtering")
		subscriptionsForFilters := migration.ToAnySlice(subscriptions)
		tflog.Debug(ctx, "Converted subscriptions for filtering")
		filteredSubscriptions, diags := migration.ApplyFilter(data.Filters, subscriptionsForFilters)
		if diags != nil {
			response.Diagnostics.Append(diags)
			return
		}

		if len(filteredSubscriptions) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v subscriptions.", len(filteredSubscriptions)),
			)
			return
		}
		if len(filteredSubscriptions) < 1 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}

		subscription := filteredSubscriptions[0].(cloudsigmaapi.Subscription)

		data.Amount = types.StringValue(subscription.Amount)
		data.AutoRenew = types.BoolValue(subscription.AutoRenew)
		data.FreeTier = types.BoolValue(subscription.FreeTier)
		data.ID = types.StringValue(subscription.ID)
		data.Period = types.StringValue(subscription.Period)
		data.Price = types.StringValue(subscription.Price)
		data.Remaining = types.StringValue(subscription.Remaining)
		data.Resource = types.StringValue(subscription.Resource)
		data.ResourceURI = types.StringValue(subscription.ResourceURI)
		data.Status = types.StringValue(subscription.Status)
		data.UUID = types.StringValue(subscription.UUID)
	} else {
		subscriptionUUID := data.UUID.ValueString()

		if subscriptionUUID == "" {
			response.Diagnostics.AddError(
				"Missing required attributes",
				`The attribute "uuid" must be defined.`,
			)
			return
		}

		tflog.Trace(ctx, "Getting subscriptions for filtering")
		subscriptions, err := d.subscriptions(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get subscriptions", err.Error())
			return
		}
		tflog.Trace(ctx, "Got subscriptions")

		subscriptionFound := false
		for _, subscription := range subscriptions {
			if subscriptionUUID == subscription.UUID {
				data.Amount = types.StringValue(subscription.Amount)
				data.AutoRenew = types.BoolValue(subscription.AutoRenew)
				data.FreeTier = types.BoolValue(subscription.FreeTier)
				data.ID = types.StringValue(subscription.ID)
				data.Period = types.StringValue(subscription.Period)
				data.Price = types.StringValue(subscription.Price)
				data.Remaining = types.StringValue(subscription.Remaining)
				data.Resource = types.StringValue(subscription.Resource)
				data.ResourceURI = types.StringValue(subscription.ResourceURI)
				data.Status = types.StringValue(subscription.Status)
				data.UUID = types.StringValue(subscription.UUID)

				subscriptionFound = true
				break
			}
		}

		if !subscriptionFound {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

// subscriptions selects auth mode at configuration, never as an error fallback.
func (d *subscriptionDataSource) subscriptions(ctx context.Context) ([]cloudsigmaapi.Subscription, error) {
	return d.clients.Native.Subscriptions(ctx)
}
