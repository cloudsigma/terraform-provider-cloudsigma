package provider

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
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
func subscriptionFixture(t *testing.T, code int, body string, calls *int, tokenAuth ...bool) *cloudsigmaapi.Client {
	t.Helper()
	useToken := len(tokenAuth) > 0 && tokenAuth[0]
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
			if r.Method != "GET" || r.Header.Get("Accept-Encoding") != "identity" || r.URL.Path != "/api/2.0/subscriptions/" || r.URL.RawQuery != "" {
				return
			}
			if useToken {
				if r.Header.Get("Authorization") != "Bearer fixture-token" {
					return
				}
			} else {
				u, p, ok := r.BasicAuth()
				if !ok || u != "fixture" || p != "fixture" {
					return
				}
			}
			if r.Header.Get("Authorization") == "" {
				return
			}
			_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", code, len(body), body)
		}()
		return a, nil
	}
	tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	http.DefaultTransport = tr
	var c *cloudsigmaapi.Client
	var e error
	if useToken {
		c, e = cloudsigmaapi.NewToken("zrh", "fixture-token", "")
	} else {
		c, e = cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	}
	http.DefaultTransport = old
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestOfflineNativeSubscription(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		uuidValue types.String
		filters   []migration.FilterModel
		want      string
	}{
		{"success", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("selected"), nil, ""},
		{"field-amount", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("amount"), Values: []types.String{types.StringValue("9007199254740993")}}}, ""},
		{"field-autorenew", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("autorenew"), Values: []types.String{types.StringValue("true")}}}, ""},
		{"field-endtime", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("endtime"), Values: []types.String{types.StringValue("end")}}}, ""},
		{"field-freetier", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("freetier"), Values: []types.String{types.StringValue("false")}}}, ""},
		{"field-id", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("id"), Values: []types.String{types.StringValue("14")}}}, ""},
		{"field-period", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("period"), Values: []types.String{types.StringValue("month")}}}, ""},
		{"field-price", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("price"), Values: []types.String{types.StringValue("0.000003839842975139617919921875")}}}, ""},
		{"field-remaining", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("remaining"), Values: []types.String{types.StringValue("0E-20")}}}, ""},
		{"field-resource", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("resource"), Values: []types.String{types.StringValue("cpu")}}}, ""},
		{"field-resourceuri", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("resourceuri"), Values: []types.String{types.StringValue("uri")}}}, ""},
		{"field-starttime", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("starttime"), Values: []types.String{types.StringValue("start")}}}, ""},
		{"field-status", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("status"), Values: []types.String{types.StringValue("active")}}}, ""},
		{"field-subscribedobject", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("subscribedobject"), Values: []types.String{types.StringValue("target")}}}, ""},
		{"field-uuid", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("ignored-filter-precedence"), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringValue("selected")}}}, ""},
		{"first-match", 200, `{"objects": [{"amount": "9007199254740993", "auto_renew": true, "end_time": "end", "free_tier": false, "id": "14", "period": "month", "price": "0.000003839842975139617919921875", "remaining": "0E-20", "resource": "cpu", "resource_uri": "uri", "start_time": "start", "status": "active", "subscribed_object": "target", "uuid": "selected"}, {"uuid": "selected", "id": "WRONG"}]}`, types.StringValue("selected"), nil, ""},
		{"invalid-field", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("selected"), []migration.FilterModel{{Name: types.StringValue("auto_renew"), Values: []types.String{types.StringValue("true")}}}, "Missing field"},
		{"unknown-filter", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("selected"), []migration.FilterModel{{Name: types.StringUnknown()}}, "Missing field"},
		{"null-values", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("selected"), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: nil}}, "No search results"},
		{"unknown-values", 200, `{"objects":[{"amount":"9007199254740993","auto_renew":true,"end_time":"end","free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","start_time":"start","status":"active","subscribed_object":"target","uuid":"selected"}],"meta":{"limit":20,"total_count":99,"next":"https://untrusted.invalid"}}`, types.StringValue("selected"), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringUnknown()}}}, "No search results"},
		{"emptyuuid", 200, `{}`, types.StringValue(""), nil, "Missing required attributes"},
		{"nulluuid", 200, `{}`, types.StringNull(), nil, "Missing required attributes"},
		{"unknownuuid", 200, `{}`, types.StringUnknown(), nil, "Missing required attributes"},
		{"numeric-invalid", 200, `{"objects":[{"amount":123}]}`, types.StringValue("selected"), nil, "Unable to get subscriptions"},
		{"decimal-number-invalid", 200, `{"objects":[{"price":1.23}]}`, types.StringValue("selected"), nil, "Unable to get subscriptions"},
		{"malformed", 200, `{"objects":{}}`, types.StringValue("selected"), nil, "Unable to get subscriptions"},
		{"empty", 200, `{"objects":[]}`, types.StringValue("selected"), nil, "No search results"},
		{"nullobjects", 200, `{"objects":null}`, types.StringValue("selected"), nil, "No search results"},
		{"404", 404, `PRIVATE`, types.StringValue("selected"), nil, "Unable to get subscriptions"},
		{"500", 500, `PRIVATE`, types.StringValue("selected"), nil, "Unable to get subscriptions"},
		{"multiple", 200, `{"objects":[{"uuid":"selected"},{"uuid":"selected"}]}`, types.StringValue("selected"), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringValue("selected")}}}, "Too many search results"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			calls := 0
			native := subscriptionFixture(t, tc.code, tc.body, &calls)
			legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: fixtureRT(func(*http.Request) (*http.Response, error) {
				t.Error("legacy fallback")
				return nil, errors.New("unexpected fallback")
			})}))
			d := &subscriptionDataSource{}
			var conf datasource.ConfigureResponse
			d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authBasic, Native: native, Legacy: legacy}}, &conf)
			if conf.Diagnostics.HasError() {
				t.Fatal(conf.Diagnostics)
			}
			var s datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &s)
			state := tfsdk.State{Schema: s.Schema}
			prior := subscriptionDataSourceModel{ID: types.StringValue("old"), UUID: tc.uuidValue, Filters: tc.filters}
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
				var got subscriptionDataSourceModel
				if x := out.State.Get(ctx, &got); x.HasError() {
					t.Fatal(x)
				}
				if !reflect.DeepEqual(got.Filters, tc.filters) {
					t.Fatal("filters changed")
				}
				if out.Diagnostics.HasError() || got.Amount.ValueString() != "9007199254740993" || got.AutoRenew.ValueBool() != true || got.FreeTier.ValueBool() != false || got.ID.ValueString() != "14" || got.Period.ValueString() != "month" || got.Price.ValueString() != "0.000003839842975139617919921875" || got.Remaining.ValueString() != "0E-20" || got.Resource.ValueString() != "cpu" || got.ResourceURI.ValueString() != "uri" || got.Status.ValueString() != "active" || got.UUID.ValueString() != "selected" {
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
	d := &subscriptionDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var s datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &s)
	state := tfsdk.State{Schema: s.Schema}
	m := subscriptionDataSourceModel{UUID: types.StringValue("selected"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") {
		t.Fatal(errors.New("cancel state"))
	}
}

func TestOfflineNativeTokenSubscription(t *testing.T) {
	const body = `{"objects":[{"amount":"9007199254740993","auto_renew":true,"free_tier":false,"id":"14","period":"month","price":"0.000003839842975139617919921875","remaining":"0E-20","resource":"cpu","resource_uri":"uri","status":"active","uuid":"selected"}]}`
	calls := 0
	native := subscriptionFixture(t, http.StatusOK, body, &calls, true)
	d := &subscriptionDataSource{}
	ctx := context.Background()
	var conf datasource.ConfigureResponse
	d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native}}, &conf)
	if conf.Diagnostics.HasError() {
		t.Fatal(conf.Diagnostics)
	}
	var s datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &s)
	state := tfsdk.State{Schema: s.Schema}
	prior := subscriptionDataSourceModel{UUID: types.StringValue("selected"), ID: types.StringValue("old")}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
	if out.Diagnostics.HasError() || calls != 1 {
		t.Fatalf("diagnostics=%v calls=%d", out.Diagnostics, calls)
	}
}

func TestOfflineTokenSubscriptionCancelPreservesState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "")
	if err != nil {
		t.Fatal(err)
	}
	d := &subscriptionDataSource{clients: &ProviderClients{Auth: authToken, Native: native}}
	var s datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &s)
	state := tfsdk.State{Schema: s.Schema}
	prior := subscriptionDataSourceModel{UUID: types.StringValue("selected"), ID: types.StringValue("old")}
	if diags := state.Set(context.Background(), &prior); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") {
		t.Fatal(out.Diagnostics)
	}
}

func TestOfflineSubscriptionConfigureRequiresNative(t *testing.T) {
	d := &subscriptionDataSource{}
	var response datasource.ConfigureResponse
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken}}, &response)
	if d.clients != nil || !response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
}
