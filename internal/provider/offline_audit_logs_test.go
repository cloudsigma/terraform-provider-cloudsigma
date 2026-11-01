package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOfflineAuditLogsDataSource(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.RequestURI() != "/api/2.0/logs/?limit=5" {
			t.Fatalf("route=%s %s", req.Method, req.URL.RequestURI())
		}
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"meta":{"limit":5,"total_count":1},"objects":[{"action":"create","success":true}]}`)
	}))
	defer server.Close()
	old := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // synthetic test server only
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		},
	}
	defer func() { http.DefaultTransport = old }()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "fixture-agent")
	if err != nil {
		t.Fatal(err)
	}
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
	var got auditLogsDataSourceModel
	if diags := out.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatal(diags)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal([]byte(got.JSON.ValueString()), &document); err != nil || got.ID.ValueString() != "audit_logs" || string(document["meta"]) != `{"limit":5,"total_count":1}` {
		t.Fatalf("state=%+v document=%s err=%v", got, got.JSON.ValueString(), err)
	}
}
