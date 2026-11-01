package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOfflineAuditLogsDataSource(t *testing.T) {
	calls := 0
	native := libraryFixture(t, http.StatusOK, `{"meta":{"limit":5,"total_count":1},"objects":[{"action":"create","success":true}]}`, &calls, "/api/2.0/logs/", "limit=5", true)
	d := NewAuditLogsDataSource()
	var configured datasource.ConfigureResponse
	d.(datasource.DataSourceWithConfigure).Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native}}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	var schemaResponse datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
	if !schemaResponse.Schema.Attributes["json"].IsSensitive() {
		t.Fatal("audit-log JSON must be sensitive")
	}
	config := auditLogsDataSourceModel{Limit: types.Int64Value(5)}
	out := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	if diags := out.State.Set(context.Background(), &config); diags.HasError() {
		t.Fatal(diags)
	}
	requestConfig := tfsdk.Config{Schema: schemaResponse.Schema, Raw: out.State.Raw}
	d.Read(context.Background(), datasource.ReadRequest{Config: requestConfig}, &out)
	if out.Diagnostics.HasError() {
		t.Fatal(out.Diagnostics)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	var got auditLogsDataSourceModel
	if diags := out.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatal(diags)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal([]byte(got.JSON.ValueString()), &document); err != nil || got.ID.ValueString() != "audit_logs" || string(document["meta"]) != `{"limit":5,"total_count":1}` {
		t.Fatalf("state=%+v document=%s err=%v", got, got.JSON.ValueString(), err)
	}
}
