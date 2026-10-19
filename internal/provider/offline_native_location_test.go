package provider

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// In-memory TLS dial seam: no listener/socket or public constructor hook.
func locationFixture(t *testing.T, code int, body string, calls *int) *cloudsigmaapi.Client {
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
			if r.URL.Path != "/api/2.0/locations/" || r.URL.RawQuery != "" {
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
func TestOfflineNativeLocation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		body    string
		uuid    types.String
		filters []migration.FilterModel
		want    string
	}{
		{"success", 200, `{"objects":[{"id":"ZRH","api_endpoint":"api","country_code":"CH","display_name":"Zurich"}]}`, types.StringValue("ZRH"), nil, ""},
		{"filter", 200, `{"objects":[{"id":"ZRH","api_endpoint":"api","country_code":"CH","display_name":"Zurich","upload_url":"upload"}]}`, types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uploadurl"), Values: []types.String{types.StringValue("upload")}}}, ""},
		{"404", 404, `PRIVATE`, types.StringValue("ZRH"), nil, "Unable to get locations"},
		{"500", 500, `PRIVATE`, types.StringValue("ZRH"), nil, "Unable to get locations"},
		{"malformed", 200, `{"objects":{}}`, types.StringValue("ZRH"), nil, "Unable to get locations"},
		{"empty", 200, `{"objects":[]}`, types.StringValue("ZRH"), nil, "No search results"},
		{"nulluuid", 200, `{}`, types.StringNull(), nil, "Missing required attributes"},
		{"unknownuuid", 200, `{}`, types.StringUnknown(), nil, "Missing required attributes"},
		{"multiple", 200, `{"objects":[{"id":"ZRH"},{"id":"ZRH"}]}`, types.StringNull(), []migration.FilterModel{{Name: types.StringValue("id"), Values: []types.String{types.StringValue("ZRH")}}}, "Too many search results"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			calls := 0
			native := locationFixture(t, tc.code, tc.body, &calls)
			d := &locationDataSource{}
			var conf datasource.ConfigureResponse
			d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authBasic, Native: native}}, &conf)
			if conf.Diagnostics.HasError() {
				t.Fatal(conf.Diagnostics)
			}
			var s datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &s)
			state := tfsdk.State{Schema: s.Schema}
			prior := locationDataSourceModel{ID: types.StringValue("old"), UUID: tc.uuid, Filters: tc.filters}
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
				var got locationDataSourceModel
				if x := out.State.Get(ctx, &got); x.HasError() {
					t.Fatal(x)
				}
				if out.Diagnostics.HasError() || got.ID.ValueString() != "ZRH" || got.UUID.ValueString() != "ZRH" || got.CountryCode.ValueString() != "CH" || got.APIEndpoint.ValueString() != "api" || got.DisplayName.ValueString() != "Zurich" {
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
	d := &locationDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var s datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &s)
	state := tfsdk.State{Schema: s.Schema}
	m := locationDataSourceModel{UUID: types.StringValue("ZRH"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") {
		t.Fatal(errors.New("cancel state"))
	}
}
