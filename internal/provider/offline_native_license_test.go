package provider

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// In-memory TLS dial seam: no listener/socket or public constructor hook.
func licenseFixture(t *testing.T, code int, body string, calls *int) *cloudsigmaapi.Client {
	t.Helper()
	old := http.DefaultTransport
	tr := old.(*http.Transport).Clone()
	tr.Proxy = nil
	tr.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			r, e := http.ReadRequest(bufio.NewReader(b))
			if e != nil {
				return
			}
			*calls++
			if r.URL.Path != "/api/2.0/licenses/" || r.URL.RawQuery != "" {
				return
			}
			_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", code, len(body), body)
		}()
		return a, nil
	}
	tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	http.DefaultTransport = tr
	c, e := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	http.DefaultTransport = old
	if e != nil {
		t.Fatal(e)
	}
	return c
}

func TestOfflineTokenLicenseNumericQuantity(t *testing.T) {
	const body = `{"objects":[{"name":"win","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`
	var calls int
	legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture-token"), cloudsigma.WithLocation("zrh"), cloudsigma.WithHTTPClient(&http.Client{Transport: fixtureRT(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/api/2.0/licenses/" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("unexpected token license request: %s %s auth=%q", r.Method, r.URL, r.Header.Get("Authorization"))
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}))
	d := &licenseDataSource{}
	ctx := context.Background()
	var conf datasource.ConfigureResponse
	d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Legacy: legacy}}, &conf)
	if conf.Diagnostics.HasError() {
		t.Fatal(conf.Diagnostics)
	}
	var schemaResponse datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResponse)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	prior := licenseDataSourceModel{ID: types.StringValue("old"), Filters: []migration.FilterModel{{Name: types.StringValue("free"), Values: []types.String{types.StringValue("9007199254740993")}}}}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResponse.Schema, Raw: state.Raw}}, &out)
	if out.Diagnostics.HasError() || calls != 1 {
		t.Fatalf("token numeric license read failed: calls=%d diagnostics=%v", calls, out.Diagnostics)
	}
	var got licenseDataSourceModel
	if diags := out.State.Get(ctx, &got); diags.HasError() || got.Name.ValueString() != "win" {
		t.Fatalf("token numeric license state: %+v diagnostics=%v", got, diags)
	}
}

func TestOfflineNativeLicense(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		nameValue types.String
		filters   []migration.FilterModel
		want      string
	}{
		{"success", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp"}]}`, types.StringValue("win"), nil, ""},
		{"filter", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993}]}`, types.StringNull(), []migration.FilterModel{{Name: types.StringValue("free"), Values: []types.String{types.StringValue("9007199254740993")}}}, ""},
		{"field-burstable", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("burstable"), Values: []types.String{types.StringValue("true")}}}, ""},
		{"field-longname", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("longname"), Values: []types.String{types.StringValue("Windows")}}}, ""},
		{"field-name", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("name"), Values: []types.String{types.StringValue("win")}}}, ""},
		{"field-resourceuri", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("resourceuri"), Values: []types.String{types.StringValue("uri")}}}, ""},
		{"field-type", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("type"), Values: []types.String{types.StringValue("instance")}}}, ""},
		{"field-usermetric", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("usermetric"), Values: []types.String{types.StringValue("smp")}}}, ""},
		{"field-free", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("free"), Values: []types.String{types.StringValue("9007199254740993")}}}, ""},
		{"field-initialpurchaserequirement", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("initialpurchaserequirement"), Values: []types.String{types.StringValue("1.20e3")}}}, ""},
		{"field-subsequentpurchaserequirement", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("subsequentpurchaserequirement"), Values: []types.String{types.StringValue("-0")}}}, ""},
		{"first-match", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0},{"name":"win","long_name":"WRONG"}]}`, types.StringValue("win"), nil, ""},
		{"invalid-field", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("win"), []migration.FilterModel{{Name: types.StringValue("long_name"), Values: []types.String{types.StringValue("Windows")}}}, "Missing field"},
		{"unknown-filter", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("win"), []migration.FilterModel{{Name: types.StringUnknown()}}, "Missing field"},
		{"null-values", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("win"), []migration.FilterModel{{Name: types.StringValue("name")}}, "No search results"},
		{"unknown-values", 200, `{"objects":[{"name":"win","burstable":true,"long_name":"Windows","resource_uri":"uri","type":"instance","user_metric":"smp","free":9007199254740993,"initial_purchase_requirement":1.20e3,"subsequent_purchase_requirement":-0}]}`, types.StringValue("win"), []migration.FilterModel{{Name: types.StringValue("name"), Values: []types.String{types.StringUnknown()}}}, "No search results"},
		{"emptyname", 200, `{}`, types.StringValue(""), nil, "Missing required attributes"},
		{"numeric-invalid", 200, `{"objects":[{"free":true}]}`, types.StringValue("win"), nil, "Unable to get licenses"},
		{"nullobjects", 200, `{"objects":null}`, types.StringValue("win"), nil, "No search results"},
		{"404", 404, `PRIVATE`, types.StringValue("win"), nil, "Unable to get licenses"},
		{"500", 500, `PRIVATE`, types.StringValue("win"), nil, "Unable to get licenses"},
		{"malformed", 200, `{"objects":{}}`, types.StringValue("win"), nil, "Unable to get licenses"},
		{"empty", 200, `{"objects":[]}`, types.StringValue("win"), nil, "No search results"},
		{"nullname", 200, `{}`, types.StringNull(), nil, "Missing required attributes"},
		{"unknownname", 200, `{}`, types.StringUnknown(), nil, "Missing required attributes"},
		{"multiple", 200, `{"objects":[{"name":"win"},{"name":"win"}]}`, types.StringNull(), []migration.FilterModel{{Name: types.StringValue("name"), Values: []types.String{types.StringValue("win")}}}, "Too many search results"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			calls := 0
			native := licenseFixture(t, tc.code, tc.body, &calls)
			legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: fixtureRT(func(*http.Request) (*http.Response, error) {
				t.Error("legacy fallback")
				return nil, errors.New("unexpected fallback")
			})}))
			d := &licenseDataSource{}
			var conf datasource.ConfigureResponse
			d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authBasic, Native: native, Legacy: legacy}}, &conf)
			if conf.Diagnostics.HasError() {
				t.Fatal(conf.Diagnostics)
			}
			var s datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &s)
			state := tfsdk.State{Schema: s.Schema}
			prior := licenseDataSourceModel{ID: types.StringValue("old"), Name: tc.nameValue, Filters: tc.filters}
			if x := state.Set(ctx, &prior); x.HasError() {
				t.Fatal(x)
			}
			out := datasource.ReadResponse{State: state}
			d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
			if tc.want != "" {
				if !out.Diagnostics.HasError() || out.Diagnostics[0].Summary() != tc.want || !out.State.Raw.Equal(state.Raw) {
					t.Fatal(out.Diagnostics)
				}
			} else {
				var got licenseDataSourceModel
				if x := out.State.Get(ctx, &got); x.HasError() {
					t.Fatal(x)
				}
				if !reflect.DeepEqual(got.Filters, tc.filters) {
					t.Fatal("filters changed")
				}
				if out.Diagnostics.HasError() || got.ID.ValueString() != "win" || got.UUID.ValueString() != "win" || !got.Burstable.ValueBool() || got.LongName.ValueString() != "Windows" || got.ResourceURI.ValueString() != "uri" || got.Type.ValueString() != "instance" || got.UserMetric.ValueString() != "smp" || got.Name.ValueString() != "win" {
					t.Fatalf("%+v %v", got, out.Diagnostics)
				}
			}
			want := 1
			if tc.want == "Missing required attributes" {
				want = 0
			}
			if calls != want {
				t.Fatal(calls)
			}
		})
	}
	// Cancellation with actual provider state and no possible legacy fallback.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	d := &licenseDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var s datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &s)
	state := tfsdk.State{Schema: s.Schema}
	m := licenseDataSourceModel{Name: types.StringValue("win"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") {
		t.Fatal(errors.New("cancel state"))
	}
}
