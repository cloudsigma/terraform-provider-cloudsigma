package provider

import (
	"context"
	"encoding/json"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*usageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*usageDataSource)(nil)
)

type usageDataSource struct {
	clients *ProviderClients
	kind    string
}

type usageDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	JSON       types.String `tfsdk:"json"`
	PollTimeGT types.String `tfsdk:"poll_time_gt"`
	PollTimeLT types.String `tfsdk:"poll_time_lt"`
	Limit      types.Int64  `tfsdk:"limit"`
}

type balanceDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

type burstUsageDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	JSON    types.String `tfsdk:"json"`
	DateGT  types.String `tfsdk:"date_gt"`
	DateGTE types.String `tfsdk:"date_gte"`
	DateLT  types.String `tfsdk:"date_lt"`
	DateLTE types.String `tfsdk:"date_lte"`
}

type pricingDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	JSON  types.String `tfsdk:"json"`
	Limit types.Int64  `tfsdk:"limit"`
}

type discountDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func NewUsageDataSource() datasource.DataSource      { return &usageDataSource{kind: "usage"} }
func NewBalanceDataSource() datasource.DataSource    { return &usageDataSource{kind: "balance"} }
func NewBurstUsageDataSource() datasource.DataSource { return &usageDataSource{kind: "burstusage"} }
func NewDailyBurstUsageDataSource() datasource.DataSource {
	return &usageDataSource{kind: "dailyburstusage"}
}
func NewPricingDataSource() datasource.DataSource  { return &usageDataSource{kind: "pricing"} }
func NewDiscountDataSource() datasource.DataSource { return &usageDataSource{kind: "discount"} }
func NewLedgerDataSource() datasource.DataSource   { return &usageDataSource{kind: "ledger"} }

func (d *usageDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_" + d.kind
}

func (d *usageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"id":   schema.StringAttribute{Computed: true},
		"json": schema.StringAttribute{Computed: true},
	}
	if d.kind == "usage" {
		attrs["poll_time_gt"] = schema.StringAttribute{Optional: true}
		attrs["poll_time_lt"] = schema.StringAttribute{Optional: true}
		attrs["limit"] = schema.Int64Attribute{Optional: true}
	}
	if d.kind == "burstusage" || d.kind == "dailyburstusage" {
		attrs["date_gt"] = schema.StringAttribute{Optional: true}
		attrs["date_gte"] = schema.StringAttribute{Optional: true}
		attrs["date_lt"] = schema.StringAttribute{Optional: true}
		attrs["date_lte"] = schema.StringAttribute{Optional: true}
	}
	if d.kind == "pricing" {
		attrs["limit"] = schema.Int64Attribute{Optional: true}
	}
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the CloudSigma " + d.kind + " API document as dynamic JSON.",
		Attributes:          attrs,
	}
}

func (d *usageDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *usageDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var config usageDataSourceModel
	if d.kind == "pricing" {
		var state pricingDataSourceModel
		response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
		if response.Diagnostics.HasError() {
			return
		}
		limit := int64(0)
		if !state.Limit.IsNull() && !state.Limit.IsUnknown() {
			limit = state.Limit.ValueInt64()
		}
		document, err := d.clients.Native.Pricing(ctx, limit)
		if err != nil {
			response.Diagnostics.AddError("Unable to fetch CloudSigma pricing", err.Error())
			return
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			response.Diagnostics.AddError("Unable to encode CloudSigma pricing", err.Error())
			return
		}
		state.ID, state.JSON = types.StringValue(d.kind), types.StringValue(string(encoded))
		response.Diagnostics.Append(response.State.Set(ctx, &state)...)
		return
	}
	if d.kind == "discount" || d.kind == "ledger" {
		var state discountDataSourceModel
		response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
		if response.Diagnostics.HasError() {
			return
		}
		var document cloudsigmaapi.DynamicDocument
		var err error
		if d.kind == "ledger" {
			document, err = d.clients.Native.Ledger(ctx)
		} else {
			document, err = d.clients.Native.Discount(ctx)
		}
		if err != nil {
			response.Diagnostics.AddError("Unable to fetch CloudSigma "+d.kind, err.Error())
			return
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			response.Diagnostics.AddError("Unable to encode CloudSigma "+d.kind, err.Error())
			return
		}
		state.ID, state.JSON = types.StringValue(d.kind), types.StringValue(string(encoded))
		response.Diagnostics.Append(response.State.Set(ctx, &state)...)
		return
	}
	if d.kind == "balance" {
		var state balanceDataSourceModel
		response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
		if response.Diagnostics.HasError() {
			return
		}
		d.readDocument(ctx, response, nil, &state)
		return
	}
	if d.kind == "burstusage" || d.kind == "dailyburstusage" {
		var state burstUsageDataSourceModel
		response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
		if response.Diagnostics.HasError() {
			return
		}
		d.readBurstDocument(ctx, response, &state)
		return
	}
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	var document cloudsigmaapi.DynamicDocument
	var err error
	switch d.kind {
	case "usage":
		limit := int64(0)
		if !config.Limit.IsNull() && !config.Limit.IsUnknown() {
			limit = config.Limit.ValueInt64()
		}
		gt, lt := "", ""
		if !config.PollTimeGT.IsNull() && !config.PollTimeGT.IsUnknown() {
			gt = config.PollTimeGT.ValueString()
		}
		if !config.PollTimeLT.IsNull() && !config.PollTimeLT.IsUnknown() {
			lt = config.PollTimeLT.ValueString()
		}
		document, err = d.clients.Native.Usage(ctx, cloudsigmaapi.UsageQuery{PollTimeGT: gt, PollTimeLT: lt, Limit: limit})
	}
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma "+d.kind, err.Error())
		return
	}
	tflog.Debug(ctx, "Fetched CloudSigma "+d.kind)
	encoded, err := json.Marshal(document)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma "+d.kind, err.Error())
		return
	}
	config.ID = types.StringValue(d.kind)
	config.JSON = types.StringValue(string(encoded))
	response.Diagnostics.Append(response.State.Set(ctx, &config)...)
}

func (d *usageDataSource) readDocument(ctx context.Context, response *datasource.ReadResponse, _ any, state *balanceDataSourceModel) {
	document, err := d.clients.Native.Balance(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma balance", err.Error())
		return
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma balance", err.Error())
		return
	}
	state.ID, state.JSON = types.StringValue(d.kind), types.StringValue(string(encoded))
	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (d *usageDataSource) readBurstDocument(ctx context.Context, response *datasource.ReadResponse, state *burstUsageDataSourceModel) {
	query := cloudsigmaapi.BurstUsageQuery{}
	if !state.DateGT.IsNull() && !state.DateGT.IsUnknown() {
		query.DateGT = state.DateGT.ValueString()
	}
	if !state.DateGTE.IsNull() && !state.DateGTE.IsUnknown() {
		query.DateGTE = state.DateGTE.ValueString()
	}
	if !state.DateLT.IsNull() && !state.DateLT.IsUnknown() {
		query.DateLT = state.DateLT.ValueString()
	}
	if !state.DateLTE.IsNull() && !state.DateLTE.IsUnknown() {
		query.DateLTE = state.DateLTE.ValueString()
	}
	var document cloudsigmaapi.DynamicDocument
	var err error
	if d.kind == "burstusage" {
		document, err = d.clients.Native.BurstUsage(ctx, query)
	} else {
		document, err = d.clients.Native.DailyBurstUsage(ctx, query)
	}
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch CloudSigma "+d.kind, err.Error())
		return
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		response.Diagnostics.AddError("Unable to encode CloudSigma "+d.kind, err.Error())
		return
	}
	state.ID, state.JSON = types.StringValue(d.kind), types.StringValue(string(encoded))
	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}
