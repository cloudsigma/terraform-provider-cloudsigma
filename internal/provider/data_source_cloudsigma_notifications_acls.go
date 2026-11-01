package provider

import (
	"context"
	"encoding/json"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*accountCollectionDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*accountCollectionDataSource)(nil)
)

type accountCollectionDataSource struct {
	clients *ProviderClients
	kind    string
}

type accountCollectionDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func NewNotificationContactsDataSource() datasource.DataSource {
	return &accountCollectionDataSource{kind: "notification_contacts"}
}

func NewNotificationPreferencesDataSource() datasource.DataSource {
	return &accountCollectionDataSource{kind: "notification_preferences"}
}

func NewACLsDataSource() datasource.DataSource {
	return &accountCollectionDataSource{kind: "acls"}
}

func (d *accountCollectionDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "cloudsigma_" + d.kind
}

func (d *accountCollectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Reads the CloudSigma " + d.kind + " collection as dynamic JSON. The response includes pagination metadata; this data source does not manage account notification or sharing settings.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{MarkdownDescription: "Stable identity for the requested account collection.", Computed: true},
			"json": schema.StringAttribute{MarkdownDescription: "The returned collection response as canonical JSON, including pagination metadata.", Computed: true},
		},
	}
}

func (d *accountCollectionDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *accountCollectionDataSource) Read(ctx context.Context, _ datasource.ReadRequest, response *datasource.ReadResponse) {
	if d.clients == nil {
		response.Diagnostics.AddError("Unconfigured CloudSigma client", "Please report this issue to the provider developers.")
		return
	}
	var document cloudsigmaapi.DynamicDocument
	var err error
	switch d.kind {
	case "notification_contacts":
		document, err = d.clients.Native.NotificationContacts(ctx)
	case "notification_preferences":
		document, err = d.clients.Native.NotificationPreferences(ctx)
	case "acls":
		document, err = d.clients.Native.ACLs(ctx)
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
	response.Diagnostics.Append(response.State.Set(ctx, &accountCollectionDataSourceModel{ID: types.StringValue(d.kind), JSON: types.StringValue(string(encoded))})...)
}
