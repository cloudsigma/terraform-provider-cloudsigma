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
)

func TestOfflineCurrentUsageDataSource(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.RequestURI() != "/api/2.0/currentusage/" {
			t.Fatalf("route=%s %s", req.Method, req.URL.RequestURI())
		}
		if got := req.Header.Get("Authorization"); got != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"balance":{"balance":"1","currency":"EUR"},"usage":{"amd_cpu":{"burst":0}}}`)
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
	d := NewCurrentUsageDataSource()
	var configured datasource.ConfigureResponse
	d.(datasource.DataSourceWithConfigure).Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native}}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	var schemaResponse datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
	out := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	d.Read(context.Background(), datasource.ReadRequest{}, &out)
	if out.Diagnostics.HasError() {
		t.Fatal(out.Diagnostics)
	}
	var got currentUsageDataSourceModel
	if diags := out.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatal(diags)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal([]byte(got.JSON.ValueString()), &document); err != nil || string(document["balance"]) != `{"balance":"1","currency":"EUR"}` {
		t.Fatalf("state=%+v document=%s err=%v", got, got.JSON.ValueString(), err)
	}
	if got.ID.ValueString() != "currentusage" {
		t.Fatalf("id=%s", got.ID.ValueString())
	}
}
