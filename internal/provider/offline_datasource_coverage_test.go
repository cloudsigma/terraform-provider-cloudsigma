package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// TestOfflineDatasourceCoverage exercises actual Framework config decoding and state
// serialization, not provider configuration or Terraform acceptance discovery.
func TestOfflineDatasourceCoverage(t *testing.T) {
	cases := []struct {
		name                         string
		newDS                        func() datasource.DataSource
		route, field, item, identity string
	}{
		{"drive", NewDriveDataSource, "drives/detail/?limit=0", "uuid", `{"uuid":"selected","name":"disk","size":4096,"status":"unmounted","storage_type":"ssd"}`, "selected"},
		{"library_drive", NewLibraryDriveDataSource, "libdrives/?limit=0", "uuid", `{"uuid":"selected","name":"image","size":8192,"arch":"x86_64","os":"linux"}`, "selected"},
		{"license", NewLicenseDataSource, "licenses/", "name", `{"name":"selected","long_name":"Fixture License","burstable":true,"type":"software","user_metric":"cpu"}`, "selected"},
		{"location", NewLocationDataSource, "locations/", "id", `{"id":"selected","country_code":"CH","display_name":"Fixture","api_endpoint":"https://example.invalid/api/2.0/"}`, "selected"},
		{"subscription", NewSubscriptionDataSource, "subscriptions/", "uuid", `{"uuid":"selected","id":"subscription-id","amount":"3","auto_renew":true,"free_tier":false,"period":"month","price":"12.50","remaining":"2","resource":"cpu","status":"active"}`, "subscription-id"},
		{"ip", NewIPDataSource, "ips/detail/", "uuid", `{"uuid":"selected","gateway":"192.0.2.1","netmask":24,"resource_uri":"/api/2.0/ips/selected/"}`, "selected"},
		{"vlan", NewVLANDataSource, "vlans/detail/", "uuid", `{"uuid":"selected","meta":{"name":"private-network"},"resource_uri":"/api/2.0/vlans/selected/"}`, "selected"},
		{"tag", NewTagDataSource, "tags/", "uuid", `{"uuid":"selected","name":"label","resource_uri":"/api/2.0/tags/selected/"}`, "selected"},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []string{"one", "none", "many", "invalid-field", "http-error", "malformed"} {
				t.Run(mode, func(t *testing.T) {
					d := tc.newDS()
					var schema datasource.SchemaResponse
					d.Schema(ctx, datasource.SchemaRequest{}, &schema)
					typ := schema.Schema.Type().TerraformType(ctx)
					attrs := map[string]tftypes.Value{}
					for k, v := range typ.(tftypes.Object).AttributeTypes {
						attrs[k] = tftypes.NewValue(v, nil)
					}
					ft := typ.(tftypes.Object).AttributeTypes["filter"].(tftypes.Set)
					field := tc.field
					if mode == "invalid-field" {
						field = "nonexistent"
					}
					attrs["filter"] = tftypes.NewValue(ft, []tftypes.Value{tftypes.NewValue(ft.ElementType, map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, field), "values": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "selected")})})})
					raw := tftypes.NewValue(typ, attrs)
					calls := 0
					body := `{"objects":[` + tc.item + `]}`
					code := 200
					switch mode {
					case "none":
						body = `{"objects":[]}`
					case "many":
						body = `{"objects":[` + tc.item + `,` + tc.item + `]}`
					case "http-error":
						code = 500
						body = `[]`
					case "malformed":
						body = `{`
					}
					route := "/api/2.0/" + tc.route
					path, query := route, ""
					if i := strings.IndexByte(route, '?'); i >= 0 {
						path, query = route[:i], route[i+1:]
					}
					native := libraryFixtureAuth(t, code, body, &calls, path, query, tc.name == "ip" || tc.name == "vlan")
					client := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithLocation("next"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
						calls++
						if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/"+tc.route {
							t.Fatalf("unexpected request %s %s", q.Method, q.URL.RequestURI())
						}
						return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
					})}))
					var conf datasource.ConfigureResponse
					d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Native: native, Legacy: client, Auth: authToken}}, &conf)
					if conf.Diagnostics.HasError() {
						t.Fatal(conf.Diagnostics)
					}
					out := datasource.ReadResponse{State: tfsdk.State{Schema: schema.Schema, Raw: raw}}
					d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: raw}}, &out)
					if calls != 1 {
						t.Fatalf("calls=%d", calls)
					}
					if mode != "one" {
						if !out.Diagnostics.HasError() {
							t.Fatal("missing diagnostic")
						}
						if !out.State.Raw.Equal(raw) {
							t.Fatal("error changed state")
						}
						summaries := map[string]string{"none": "No search results", "many": "Too many search results", "invalid-field": "Missing field"}
						if want := summaries[mode]; want != "" && out.Diagnostics[0].Summary() != want {
							t.Fatalf("diagnostic %v", out.Diagnostics)
						}
						return
					}
					if out.Diagnostics.HasError() {
						t.Fatal(out.Diagnostics)
					}
					var id types.String
					if di := out.State.GetAttribute(ctx, tfpath.Root("id"), &id); di.HasError() || id.ValueString() != tc.identity {
						t.Fatalf("identity %v %v", id, di)
					}
					var uuid types.String
					if di := out.State.GetAttribute(ctx, tfpath.Root("uuid"), &uuid); di.HasError() || uuid.ValueString() != "selected" {
						t.Fatalf("uuid %v %v", uuid, di)
					}
					var filters types.Set
					if di := out.State.GetAttribute(ctx, tfpath.Root("filter"), &filters); di.HasError() || len(filters.Elements()) != 1 {
						t.Fatalf("filter preservation %v", di)
					}
				})
			}
		})
	}
	// Every registered datasource has a nil-provider-data no-op and wrong-type diagnostic.
	factories := []func() datasource.DataSource{NewDriveDataSource, NewLibraryDriveDataSource, NewLicenseDataSource, NewLocationDataSource, NewSubscriptionDataSource, NewIPDataSource, NewVLANDataSource, NewTagDataSource, NewProfileDataSource}
	for i, newDS := range factories {
		t.Run(fmt.Sprintf("configure-%d", i), func(t *testing.T) {
			d := newDS().(datasource.DataSourceWithConfigure)
			var empty, bad datasource.ConfigureResponse
			d.Configure(ctx, datasource.ConfigureRequest{}, &empty)
			if empty.Diagnostics.HasError() {
				t.Fatal(empty.Diagnostics)
			}
			d.Configure(ctx, datasource.ConfigureRequest{ProviderData: "wrong-type"}, &bad)
			if !bad.Diagnostics.HasError() || bad.Diagnostics[0].Summary() != "Unconfigured CloudSigma client" {
				t.Fatal(bad.Diagnostics)
			}
		})
	}
}
