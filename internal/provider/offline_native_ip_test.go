package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/migration"
)

func TestOfflineNativeIP(t *testing.T) {
	object := `{"uuid":"185.12.6.183","gateway":"185.12.6.1","netmask":9007199254740993,"resource_uri":"/api/2.0/ips/185.12.6.183/","nameservers":["1.2.3.4"],"server":null}`
	filter := []migration.FilterModel{{Name: types.StringValue("nameservers"), Values: []types.String{types.StringValue("1.2.3.4")}}}
	for _, tc := range []struct {
		name    string
		uuid    types.String
		filters []migration.FilterModel
		body    string
		code    int
		want    string
	}{
		{"nameservers", types.StringNull(), filter, `{"objects":[` + object + `]}`, 200, ""},
		{"precision", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("netmask"), Values: []types.String{types.StringValue("9007199254740993")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"nil-server", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("server"), Values: []types.String{types.StringValue("")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"unknownfilter", types.StringNull(), []migration.FilterModel{{Name: types.StringUnknown()}}, `{"objects":[` + object + `]}`, 200, "Missing field"},
		{"nullvalues", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid")}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"unknownvalues", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringUnknown()}}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"singleton", types.StringValue("185.12.6.183"), nil, object, 200, ""},
		{"filter-precedence", types.StringValue("ignored"), filter, `{"objects":[` + object + `]}`, 200, ""},
		{"404", types.StringValue("185.12.6.183"), nil, `PRIVATE`, 404, "Unable to get IP"},
		{"list404", types.StringNull(), filter, `PRIVATE`, 404, "Unable to get IPs"},
		{"listdecode", types.StringNull(), filter, `{"objects":{}}`, 200, "Unable to get IPs"},
		{"500", types.StringValue("185.12.6.183"), nil, `PRIVATE`, 500, "Unable to get IP"},
		{"invalid", types.StringValue("185.12.6.183"), nil, `{"netmask":"bad"}`, 200, "Unable to get IP"},
		{"empty", types.StringNull(), filter, `{"objects":[]}`, 200, "No search results"},
		{"null", types.StringNull(), filter, `{"objects":null}`, 200, "No search results"},
		{"many", types.StringNull(), filter, `{"objects":[` + object + `,` + object + `]}`, 200, "Too many search results"},
		{"missing", types.StringNull(), nil, `{}`, 200, "Missing required attributes"},
		{"unknown", types.StringUnknown(), nil, `{}`, 200, "Missing required attributes"},
	} {
		for _, token := range []bool{false, true} {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				calls := 0
				path := "/api/2.0/ips/detail/"
				query := ""
				if len(tc.filters) == 0 {
					if tc.uuid.ValueString() != "" {
						path = "/api/2.0/ips/" + tc.uuid.ValueString() + "/"
						query = ""
					}
				}
				native := libraryFixture(t, tc.code, tc.body, &calls, path, query)
				legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: fixtureRT(func(r *http.Request) (*http.Response, error) {
					if !token {
						t.Fatal("fallback")
					}
					calls++
					if r.URL.Path != path || r.URL.RawQuery != query {
						t.Fatal(r.URL)
					}
					return &http.Response{StatusCode: tc.code, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
				})}))
				auth := authBasic
				if token {
					auth = authToken
				}
				d := &ipDataSource{}
				var conf datasource.ConfigureResponse
				d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: auth, Native: native, Legacy: legacy}}, &conf)
				if conf.Diagnostics.HasError() {
					t.Fatal(conf.Diagnostics)
				}
				var schema datasource.SchemaResponse
				d.Schema(ctx, datasource.SchemaRequest{}, &schema)
				state := tfsdk.State{Schema: schema.Schema}
				prior := ipDataSourceModel{ID: types.StringValue("old"), UUID: tc.uuid, Filters: tc.filters}
				if x := state.Set(ctx, &prior); x.HasError() {
					t.Fatal(x)
				}
				out := datasource.ReadResponse{State: state}
				d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
				if tc.want != "" {
					if !out.Diagnostics.HasError() || out.Diagnostics[0].Summary() != tc.want || !out.State.Raw.Equal(state.Raw) {
						t.Fatal(out.Diagnostics)
					}
				} else {
					var got ipDataSourceModel
					if x := out.State.Get(ctx, &got); x.HasError() {
						t.Fatal(x)
					}
					if out.Diagnostics.HasError() || got.ID.ValueString() != "185.12.6.183" || got.UUID.ValueString() != "185.12.6.183" || got.Netmask.ValueInt64() != 9007199254740993 || got.Gateway.ValueString() != "185.12.6.1" || got.ResourceURI.ValueString() != "/api/2.0/ips/185.12.6.183/" {
						t.Fatal(got, out.Diagnostics)
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
	}
}

func TestOfflineIPCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	d := &ipDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	m := ipDataSourceModel{UUID: types.StringValue("185.12.6.183"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}
