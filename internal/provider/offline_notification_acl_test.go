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

func TestOfflineNotificationACLDataSources(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.RawQuery != "limit=0" || req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("request=%s %s", req.Method, req.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"meta":{"offset":0,"total_count":1},"objects":[{"uuid":"fixture-id","dynamic":"value"}]}`)
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
	for _, tc := range []struct {
		kind string
		new  func() datasource.DataSource
	}{
		{kind: "notification_contacts", new: NewNotificationContactsDataSource},
		{kind: "notification_preferences", new: NewNotificationPreferencesDataSource},
		{kind: "acls", new: NewACLsDataSource},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			d := tc.new()
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
			var got accountCollectionDataSourceModel
			if diags := out.State.Get(context.Background(), &got); diags.HasError() {
				t.Fatal(diags)
			}
			var document map[string]json.RawMessage
			if err := json.Unmarshal([]byte(got.JSON.ValueString()), &document); err != nil || got.ID.ValueString() != tc.kind || string(document["objects"]) != `[{"uuid":"fixture-id","dynamic":"value"}]` {
				t.Fatalf("state=%+v err=%v", got, err)
			}
		})
	}
}
