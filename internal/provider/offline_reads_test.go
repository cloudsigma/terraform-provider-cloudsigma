package provider

import (
	"bytes"
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
	"github.com/hashicorp/terraform-plugin-log/tflogtest"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type fixtureRT func(*http.Request) (*http.Response, error)

func (f fixtureRT) RoundTrip(q *http.Request) (*http.Response, error) { return f(q) }
func TestOfflineDatasourceReads(t *testing.T) {
	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)
	defer func() {
		if strings.Contains(logs.String(), "PRIVATE-CANARY") {
			t.Error("provider logs contain sensitive synthetic canary")
		}
	}()
	for _, kind := range []string{"profile", "ip", "vlan", "tag"} {
		for _, code := range []int{200, 404, 500} {
			t.Run(fmt.Sprint(kind, code), func(t *testing.T) {
				var d datasource.DataSource
				path := "/api/2.0/"
				switch kind {
				case "profile":
					d = NewProfileDataSource()
					path += "profile/"
				case "ip":
					d = NewIPDataSource()
					path += "ips/fixture/"
				case "vlan":
					d = NewVLANDataSource()
					path += "vlans/fixture/"
				case "tag":
					d = NewTagDataSource()
					path += "tags/fixture/"
				}
				calls := 0
				var providerData *ProviderClients
				if kind == "profile" {
					// Profile reads are provider-owned native transport on both
					// credential modes; keep this fixture aligned with that boundary
					// instead of expecting a legacy SDK round trip.
					body := `{"uuid":"fixture","name":"PRIVATE-CANARY","address":"PRIVATE-CANARY","company":"PRIVATE-CANARY","first_name":"PRIVATE-CANARY","last_name":"PRIVATE-CANARY","title":"PRIVATE-CANARY"}`
					if code != 200 {
						body = `[]`
					}
					providerData = &ProviderClients{Auth: authToken, Native: profileFixture(t, code, body, &calls, "Bearer fixture-token")}
				} else if kind == "ip" || kind == "vlan" {
					body := `{"uuid":"fixture","name":"PRIVATE-CANARY","address":"PRIVATE-CANARY","company":"PRIVATE-CANARY","meta":{"s":"PRIVATE-CANARY"},"netmask":24}`
					if code != 200 {
						body = `[]`
					}
					providerData = &ProviderClients{Auth: authToken, Native: libraryFixture(t, code, body, &calls, path, "", true)}
				} else {
					c := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithLocation("next"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
						calls++
						if q.Method != "GET" || q.URL.RequestURI() != path {
							t.Fatal("wire", q.URL.RequestURI())
						}
						body := `{"uuid":"fixture","name":"PRIVATE-CANARY","address":"PRIVATE-CANARY","company":"PRIVATE-CANARY","meta":{"s":"PRIVATE-CANARY"},"netmask":24}`
						if code != 200 {
							body = `[]`
						}
						return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
					})}))
					providerData = &ProviderClients{Legacy: c, Auth: authToken}
				}
				var sc datasource.SchemaResponse
				d.Schema(ctx, datasource.SchemaRequest{}, &sc)
				typ := sc.Schema.Type().TerraformType(ctx)
				attrs := map[string]tftypes.Value{}
				for k, v := range typ.(tftypes.Object).AttributeTypes {
					attrs[k] = tftypes.NewValue(v, nil)
				}
				if kind != "profile" {
					attrs["uuid"] = tftypes.NewValue(tftypes.String, "fixture")
				}
				raw := tftypes.NewValue(typ, attrs)
				var conf datasource.ConfigureResponse
				d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &conf)
				resp := datasource.ReadResponse{State: tfsdk.State{Schema: sc.Schema, Raw: raw}}
				d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: sc.Schema, Raw: raw}}, &resp)
				if calls != 1 {
					t.Fatal("calls", calls)
				}
				if code == 200 && resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}
				if code != 200 && !resp.Diagnostics.HasError() {
					t.Fatal("missing diagnostic")
				}
				if code == 200 {
					var id types.String
					di := resp.State.GetAttribute(ctx, tfpath.Root("uuid"), &id)
					if di.HasError() || id.ValueString() != "fixture" {
						t.Fatal("state uuid")
					}
				} else {
					var id types.String
					di := resp.State.GetAttribute(ctx, tfpath.Root("uuid"), &id)
					if di.HasError() {
						t.Fatal("state decode")
					}
					if kind != "profile" && id.ValueString() != "fixture" {
						t.Fatal("datasource error must not remove config/state id")
					}
				}
			})
		}
	}
}
