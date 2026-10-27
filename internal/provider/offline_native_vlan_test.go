package provider

import (
	"context"
	"fmt"
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

func TestOfflineNativeVLAN(t *testing.T) {
	object := `{"uuid":"vlan-id","meta":{"name":"blue"},"resource_uri":"/api/2.0/vlans/vlan-id/","subscription":{"id":9007199254740993},"owner":null}`
	filter := []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringValue("vlan-id")}}}
	for _, tc := range []struct {
		name    string
		uuid    types.String
		filters []migration.FilterModel
		body    string
		code    int
		want    string
	}{
		{"uuid-filter", types.StringNull(), filter, `{"objects":[` + object + `]}`, 200, ""},
		{"nil-owner", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("owner"), Values: []types.String{types.StringValue("")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"nested-filter", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("subscription"), Values: []types.String{types.StringValue("9007199254740993")}}}, `{"objects":[` + object + `]}`, 200, "Invalid field type"},
		{"unknownfilter", types.StringNull(), []migration.FilterModel{{Name: types.StringUnknown()}}, `{"objects":[` + object + `]}`, 200, "Missing field"},
		{"nullvalues", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid")}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"unknownvalues", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringUnknown()}}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"name", types.StringNull(), nil, `{"objects":[` + object + `]}`, 200, ""},
		{"name-first", types.StringNull(), nil, `{"objects":[` + object + `,{"uuid":"second","meta":{"name":"blue"}}]}`, 200, ""},
		{"uuid-name", types.StringValue("vlan-id"), nil, object, 200, ""},
		{"ambiguous", types.StringValue("vlan-id"), nil, object, 200, "Ambiguous search result"},
		{"name-miss", types.StringNull(), nil, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"name404", types.StringNull(), nil, `PRIVATE`, 404, "Unable to get VLANs"},
		{"unknown-name", types.StringNull(), nil, `{}`, 200, "Missing required attributes"},
		{"singleton", types.StringValue("vlan-id"), nil, object, 200, ""},
		{"filter-precedence", types.StringValue("ignored"), filter, `{"objects":[` + object + `]}`, 200, ""},
		{"404", types.StringValue("vlan-id"), nil, `PRIVATE`, 404, "Unable to get VLAN"},
		{"list404", types.StringNull(), filter, `PRIVATE`, 404, "Unable to get VLANs"},
		{"listdecode", types.StringNull(), filter, `{"objects":{}}`, 200, "Unable to get VLANs"},
		{"500", types.StringValue("vlan-id"), nil, `PRIVATE`, 500, "Unable to get VLAN"},
		{"invalid", types.StringValue("vlan-id"), nil, `{"subscription":{"id":"bad"}}`, 200, "Unable to get VLAN"},
		{"empty", types.StringNull(), filter, `{"objects":[]}`, 200, "No search results"},
		{"null", types.StringNull(), filter, `{"objects":null}`, 200, "No search results"},
		{"many", types.StringNull(), filter, `{"objects":[` + object + `,` + object + `]}`, 200, "Too many search results"},
		{"missing", types.StringNull(), nil, `{}`, 200, "Missing required attributes"},
		{"unknown", types.StringUnknown(), nil, `{}`, 200, "Missing required attributes"},
	} {
		for _, token := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/token=%v", tc.name, token), func(t *testing.T) {
				ctx := context.Background()
				calls := 0
				path := "/api/2.0/vlans/detail/"
				query := ""
				if len(tc.filters) == 0 {
					if tc.uuid.ValueString() != "" {
						path = "/api/2.0/vlans/" + tc.uuid.ValueString() + "/"
						query = ""
					}
				}
				native := libraryFixture(t, tc.code, tc.body, &calls, path, query, token)
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
				d := &vlanDataSource{}
				var conf datasource.ConfigureResponse
				d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: auth, Native: native, Legacy: legacy}}, &conf)
				if conf.Diagnostics.HasError() {
					t.Fatal(conf.Diagnostics)
				}
				var schema datasource.SchemaResponse
				d.Schema(ctx, datasource.SchemaRequest{}, &schema)
				state := tfsdk.State{Schema: schema.Schema}
				lookupName := types.StringNull()
				switch tc.name {
				case "name", "name-first", "uuid-name", "name404":
					lookupName = types.StringValue("blue")
				case "name-miss", "ambiguous":
					lookupName = types.StringValue("red")
				case "unknown-name":
					lookupName = types.StringUnknown()
				}
				prior := vlanDataSourceModel{Name: lookupName, ID: types.StringValue("old"), UUID: tc.uuid, Filters: tc.filters}
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
					var got vlanDataSourceModel
					if x := out.State.Get(ctx, &got); x.HasError() {
						t.Fatal(x)
					}
					if out.Diagnostics.HasError() || got.ID.ValueString() != "vlan-id" || got.UUID.ValueString() != "vlan-id" || got.Name.ValueString() != "blue" || got.ResourceURI.ValueString() != "/api/2.0/vlans/vlan-id/" {
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

func TestOfflineVLANCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	d := &vlanDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	m := vlanDataSourceModel{UUID: types.StringValue("vlan-id"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}

func TestOfflineTokenVLANCancelPreservesState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "")
	if err != nil {
		t.Fatal(err)
	}
	d := &vlanDataSource{clients: &ProviderClients{Auth: authToken, Native: native}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	prior := vlanDataSourceModel{UUID: types.StringValue("vlan-id"), ID: types.StringValue("old"), Name: types.StringValue("old-name")}
	if diags := state.Set(context.Background(), &prior); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}

func TestOfflineVLANConfigureRequiresNative(t *testing.T) {
	legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"))
	d := &vlanDataSource{}
	var response datasource.ConfigureResponse
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Legacy: legacy}}, &response)
	if d.clients != nil || !response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
}
