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

func TestOfflineBackupSchedulersDataSource(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.RequestURI() != "/api/2.0/backupschedulers/?limit=0" {
			t.Fatalf("route=%s %s", req.Method, req.URL.RequestURI())
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture" || password != "fixture" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		_, _ = io.WriteString(w, `{"meta":{"offset":0,"total_count":1},"objects":[{"uuid":"scheduler-1","name":"daily","type":"snapshot","is_enabled":true}]}`)
	}))
	defer server.Close()
	old := http.DefaultTransport
	http.DefaultTransport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
	}}
	defer func() { http.DefaultTransport = old }()
	native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "fixture-agent")
	if err != nil {
		t.Fatal(err)
	}
	d := NewBackupSchedulersDataSource()
	var configured datasource.ConfigureResponse
	d.(datasource.DataSourceWithConfigure).Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authBasic, Native: native}}, &configured)
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
	var got backupSchedulersDataSourceModel
	if diags := out.State.Get(context.Background(), &got); diags.HasError() {
		t.Fatal(diags)
	}
	var objects []map[string]any
	if err := json.Unmarshal([]byte(got.JSON.ValueString()), &objects); err != nil || len(objects) != 1 || objects[0]["uuid"] != "scheduler-1" {
		t.Fatalf("json=%s err=%v", got.JSON.ValueString(), err)
	}
}
