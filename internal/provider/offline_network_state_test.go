package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestOfflineNetworkProjectedState qualifies only the existing projected schema.
// Wire owner/subscription/nameservers are documented but deliberately NOT schema fields.
func TestOfflineNetworkProjectedState(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name, route, uuid, fixture string
		factory                    func() datasource.DataSource
	}{
		{"ip", "ips/detail/", "185.12.6.183", `{"gateway":"185.12.6.1","grantees":[],"meta":{},"nameservers":["69.194.139.62","178.22.66.167","178.22.71.56"],"netmask":24,"owner":{"resource_uri":"/api/2.0/user/5b4a69a3-8e78-4c45-a8ba-8b13f0895e23/","uuid":"5b4a69a3-8e78-4c45-a8ba-8b13f0895e23"},"permissions":[],"resource_uri":"/api/2.0/ips/185.12.6.183/","server":null,"subscription":{"id":7273,"resource_uri":"/api/2.0/subscriptions/7273/"},"tags":[],"uuid":"185.12.6.183"}`, NewIPDataSource},
		{"vlan", "vlans/detail/", "10619300-edda-42ba-91e0-7e3df0689d00", `{"bottom_vlan_tag":null,"grantees":[],"meta":{},"owner":{"resource_uri":"/api/2.0/user/c2fc9982-cf2e-434a-bf63-e22a27b39f00/","uuid":"c2fc9982-cf2e-434a-bf63-e22a27b39f00"},"permissions":[],"resource_uri":"/api/2.0/vlans/10619300-edda-42ba-91e0-7e3df0689d00/","servers":[],"subscription":{"id":12,"resource_uri":"/api/2.0/subscriptions/12/"},"tags":[],"uuid":"10619300-edda-42ba-91e0-7e3df0689d00"}`, NewVLANDataSource},
	}
	for _, tc := range cases {
		for _, mode := range []string{"known", "computed-unknown", "filter-null", "filter-unknown", "nested-unknown", "wire-type-error"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				d := tc.factory()
				var sr datasource.SchemaResponse
				d.Schema(ctx, datasource.SchemaRequest{}, &sr)
				typ := sr.Schema.Type().TerraformType(ctx)
				ats := typ.(tftypes.Object).AttributeTypes
				wantCount := 5
				if tc.name == "ip" {
					wantCount = 6
				}
				if len(ats) != wantCount {
					t.Fatalf("schema field count %d", len(ats))
				}
				for _, unsupported := range []string{"owner", "subscription", "nameservers", "servers", "tags", "meta"} {
					if _, ok := ats[unsupported]; ok {
						t.Fatalf("unexpected projected field %s", unsupported)
					}
				}
				attrs := map[string]tftypes.Value{}
				for k, v := range ats {
					attrs[k] = tftypes.NewValue(v, nil)
				}
				ft := ats["filter"].(tftypes.Set)
				values := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, tc.uuid), tftypes.NewValue(tftypes.String, "nonmatching")})
				if mode == "nested-unknown" {
					values = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, tftypes.UnknownValue)
				}
				attrs["filter"] = tftypes.NewValue(ft, []tftypes.Value{tftypes.NewValue(ft.ElementType, map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "uuid"), "values": values})})
				if mode == "computed-unknown" {
					for k, v := range ats {
						if k != "filter" {
							attrs[k] = tftypes.NewValue(v, tftypes.UnknownValue)
						}
					}
				}
				if mode == "filter-null" {
					attrs["filter"] = tftypes.NewValue(ft, nil)
				}
				if mode == "filter-unknown" {
					attrs["filter"] = tftypes.NewValue(ft, tftypes.UnknownValue)
				}
				raw := tftypes.NewValue(typ, attrs)
				calls := 0
				client := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithLocation("next"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
					calls++
					if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/"+tc.route {
						t.Fatalf("request %s %s", q.Method, q.URL.RequestURI())
					}
					body := `{"objects":[` + tc.fixture + `]}`
					if mode == "wire-type-error" {
						body = `{"objects":[{"uuid":{}}]}`
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
				})}))
				var cr datasource.ConfigureResponse
				d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Legacy: client, Auth: authToken}}, &cr)
				if len(cr.Diagnostics) != 0 {
					t.Fatal(cr.Diagnostics)
				}
				out := datasource.ReadResponse{State: tfsdk.State{Schema: sr.Schema, Raw: raw}}
				d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: sr.Schema, Raw: raw}}, &out)
				failure := mode == "filter-null" || mode == "filter-unknown" || mode == "nested-unknown" || mode == "wire-type-error"
				if failure {
					expectedCalls := 0
					if mode == "wire-type-error" {
						expectedCalls = 1
					}
					if calls != expectedCalls || !out.Diagnostics.HasError() || !out.State.Raw.Equal(raw) {
						t.Fatalf("failure state/calls diagnostics: %d %v", calls, out.Diagnostics)
					}
					summary := "Value Conversion Error"
					if mode == "filter-null" {
						summary = "Missing required attributes"
					}
					if mode == "wire-type-error" {
						summary = "Unable to get IPs"
						if tc.name == "vlan" {
							summary = "Unable to get VLANs"
						}
					}
					if out.Diagnostics[0].Summary() != summary {
						t.Fatalf("summary %q", out.Diagnostics[0].Summary())
					}
					if mode == "wire-type-error" && !strings.Contains(out.Diagnostics[0].Detail(), "cannot unmarshal object") {
						t.Fatal(out.Diagnostics)
					}
					return
				}
				if calls != 1 || len(out.Diagnostics) != 0 {
					t.Fatalf("calls %d diag %v", calls, out.Diagnostics)
				}
				// Independently specified full expected typed state, including exact nested filter values.
				expected := map[string]tftypes.Value{"filter": attrs["filter"], "id": tftypes.NewValue(tftypes.String, tc.uuid), "uuid": tftypes.NewValue(tftypes.String, tc.uuid), "resource_uri": tftypes.NewValue(tftypes.String, "/api/2.0/"+tc.name+"s/"+tc.uuid+"/")}
				if tc.name == "ip" {
					expected["gateway"] = tftypes.NewValue(tftypes.String, "185.12.6.1")
					expected["netmask"] = tftypes.NewValue(tftypes.Number, 24)
				} else {
					expected["name"] = tftypes.NewValue(tftypes.String, "")
				}
				want := tftypes.NewValue(typ, expected)
				if !out.State.Raw.Equal(want) {
					t.Fatalf("state got %s want %s", out.State.Raw, want)
				}
				// Decode actual provider model then reconstruct a NEW Framework state from it.
				rebuilt := tfsdk.State{Schema: sr.Schema}
				if tc.name == "ip" {
					var model ipDataSourceModel
					if di := out.State.Get(ctx, &model); di.HasError() {
						t.Fatal(di)
					}
					if di := rebuilt.Set(ctx, &model); di.HasError() {
						t.Fatal(di)
					}
				} else {
					var model vlanDataSourceModel
					if di := out.State.Get(ctx, &model); di.HasError() {
						t.Fatal(di)
					}
					if di := rebuilt.Set(ctx, &model); di.HasError() {
						t.Fatal(di)
					}
				}
				if !rebuilt.Raw.Equal(want) {
					t.Fatal("model round-trip changed full state")
				}
				gotJSON, err := tfprotov6.NewDynamicValue(typ, rebuilt.Raw)
				if err != nil || len(gotJSON.MsgPack) == 0 {
					t.Fatalf("serialization %v", err)
				}
				decoded, err := gotJSON.Unmarshal(typ)
				if err != nil || !decoded.Equal(want) {
					t.Fatalf("serialized state %v", err)
				}
			})
		}
	}
}
