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

func TestOfflineUsageDataSources(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/api/2.0/usage/":
			if req.URL.Query().Get("poll_time_gt") != "2024-04-01" || req.URL.Query().Get("poll_time_lt") != "2024-04-02" || req.URL.Query().Get("limit") != "1500" {
				t.Fatalf("usage query=%s", req.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"meta":{"total_count":1},"objects":[{"resource":"intel_cpu"}]}`)
		case "/api/2.0/balance/":
			_, _ = io.WriteString(w, `{"balance":"1","currency":"EUR"}`)
		case "/api/2.0/burstusage/", "/api/2.0/dailyburstusage/":
			_, _ = io.WriteString(w, `{"meta":{"total_count":0},"objects":[]}`)
		case "/api/2.0/pricing/":
			if req.URL.Query().Get("limit") != "5" {
				t.Fatalf("pricing query=%s", req.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"current":{"cpu":1},"next":{"cpu":2},"meta":{"total_count":1},"objects":[]}`)
		case "/api/2.0/discount/":
			_, _ = io.WriteString(w, `{"meta":{"total_count":1},"objects":[{"period":"3 months","value":"0.03"}]}`)
		default:
			http.NotFound(w, req)
		}
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
		name string
		new  func() datasource.DataSource
		want string
	}{
		{name: "usage", new: NewUsageDataSource, want: "usage"},
		{name: "balance", new: NewBalanceDataSource, want: "balance"},
		{name: "burst", new: NewBurstUsageDataSource, want: "burstusage"},
		{name: "daily", new: NewDailyBurstUsageDataSource, want: "dailyburstusage"},
		{name: "pricing", new: NewPricingDataSource, want: "pricing"},
		{name: "discount", new: NewDiscountDataSource, want: "discount"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.new()
			var configured datasource.ConfigureResponse
			d.(datasource.DataSourceWithConfigure).Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native}}, &configured)
			if configured.Diagnostics.HasError() {
				t.Fatal(configured.Diagnostics)
			}
			var schemaResponse datasource.SchemaResponse
			d.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
			out := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			switch tc.name {
			case "usage":
				config := usageDataSourceModel{PollTimeGT: types.StringValue("2024-04-01"), PollTimeLT: types.StringValue("2024-04-02"), Limit: types.Int64Value(1500)}
				if diags := out.State.Set(context.Background(), &config); diags.HasError() {
					t.Fatal(diags)
				}
			case "balance":
				if diags := out.State.Set(context.Background(), &balanceDataSourceModel{}); diags.HasError() {
					t.Fatal(diags)
				}
			case "pricing":
				config := pricingDataSourceModel{Limit: types.Int64Value(5)}
				if diags := out.State.Set(context.Background(), &config); diags.HasError() {
					t.Fatal(diags)
				}
			case "discount":
				if diags := out.State.Set(context.Background(), &discountDataSourceModel{}); diags.HasError() {
					t.Fatal(diags)
				}
			default:
				if diags := out.State.Set(context.Background(), &burstUsageDataSourceModel{}); diags.HasError() {
					t.Fatal(diags)
				}
			}
			var requestConfig tfsdk.Config
			requestConfig.Schema = schemaResponse.Schema
			requestConfig.Raw = out.State.Raw
			d.Read(context.Background(), datasource.ReadRequest{Config: requestConfig}, &out)
			if out.Diagnostics.HasError() {
				t.Fatal(out.Diagnostics)
			}
			var id, encoded string
			switch tc.name {
			case "usage":
				var got usageDataSourceModel
				if diags := out.State.Get(context.Background(), &got); diags.HasError() {
					t.Fatal(diags)
				}
				id, encoded = got.ID.ValueString(), got.JSON.ValueString()
			case "balance":
				var got balanceDataSourceModel
				if diags := out.State.Get(context.Background(), &got); diags.HasError() {
					t.Fatal(diags)
				}
				id, encoded = got.ID.ValueString(), got.JSON.ValueString()
			case "pricing":
				var got pricingDataSourceModel
				if diags := out.State.Get(context.Background(), &got); diags.HasError() {
					t.Fatal(diags)
				}
				id, encoded = got.ID.ValueString(), got.JSON.ValueString()
			case "discount":
				var got discountDataSourceModel
				if diags := out.State.Get(context.Background(), &got); diags.HasError() {
					t.Fatal(diags)
				}
				id, encoded = got.ID.ValueString(), got.JSON.ValueString()
			default:
				var got burstUsageDataSourceModel
				if diags := out.State.Get(context.Background(), &got); diags.HasError() {
					t.Fatal(diags)
				}
				id, encoded = got.ID.ValueString(), got.JSON.ValueString()
			}
			var document map[string]json.RawMessage
			if err := json.Unmarshal([]byte(encoded), &document); err != nil || id != tc.want {
				t.Fatalf("id=%s json=%s err=%v", id, encoded, err)
			}
		})
	}
}
