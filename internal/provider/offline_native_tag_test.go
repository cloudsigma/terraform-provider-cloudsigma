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

func TestOfflineNativeTag(t *testing.T) {
	object := `{"uuid":"tag-id","name":"blue","resource_uri":"/api/2.0/tags/tag-id/","resources":[{"uuid":"r","owner":{"uuid":"o"}}],"owner":null}`
	filter := []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringValue("tag-id")}}}
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
		{"nested-filter", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("resources"), Values: []types.String{types.StringValue("9007199254740993")}}}, `{"objects":[` + object + `]}`, 200, "Invalid field type"},
		{"unknownfilter", types.StringNull(), []migration.FilterModel{{Name: types.StringUnknown()}}, `{"objects":[` + object + `]}`, 200, "Missing field"},
		{"nullvalues", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid")}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"unknownvalues", types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringUnknown()}}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"name", types.StringNull(), nil, `{"objects":[` + object + `]}`, 200, ""},
		{"name-first", types.StringNull(), nil, `{"objects":[` + object + `,{"uuid":"second","name":"blue"}]}`, 200, ""},
		{"uuid-name", types.StringValue("tag-id"), nil, object, 200, ""},
		{"ambiguous", types.StringValue("tag-id"), nil, object, 200, "Ambiguous search result"},
		{"name-miss", types.StringNull(), nil, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"name404", types.StringNull(), nil, `PRIVATE`, 404, "Unable to get tags"},
		{"unknown-name", types.StringNull(), nil, `{}`, 200, "Missing required attributes"},
		{"singleton", types.StringValue("tag-id"), nil, object, 200, ""},
		{"filter-precedence", types.StringValue("ignored"), filter, `{"objects":[` + object + `]}`, 200, ""},
		{"404", types.StringValue("tag-id"), nil, `PRIVATE`, 404, "Unable to get tag"},
		{"list404", types.StringNull(), filter, `PRIVATE`, 404, "Unable to get tags"},
		{"listdecode", types.StringNull(), filter, `{"objects":{}}`, 200, "Unable to get tags"},
		{"500", types.StringValue("tag-id"), nil, `PRIVATE`, 500, "Unable to get tag"},
		{"invalid", types.StringValue("tag-id"), nil, `{"resources":{}}`, 200, "Unable to get tag"},
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
				path := "/api/2.0/tags/"
				query := ""
				if len(tc.filters) == 0 {
					if tc.uuid.ValueString() != "" {
						path = "/api/2.0/tags/" + tc.uuid.ValueString() + "/"
						query = ""
					}
				}
				native := libraryFixture(t, tc.code, tc.body, &calls, path, query, token)
				legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: fixtureRT(func(r *http.Request) (*http.Response, error) {
					if !token && path == "/api/2.0/tags/" {
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
				d := &tagDataSource{}
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
				prior := tagDataSourceModel{Name: lookupName, ID: types.StringValue("old"), UUID: tc.uuid, Filters: tc.filters}
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
					var got tagDataSourceModel
					if x := out.State.Get(ctx, &got); x.HasError() {
						t.Fatal(x)
					}
					if out.Diagnostics.HasError() || got.ID.ValueString() != "tag-id" || got.UUID.ValueString() != "tag-id" || got.Name.ValueString() != "blue" || got.ResourceURI.ValueString() != "/api/2.0/tags/tag-id/" {
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

func TestOfflineTagCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	d := &tagDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	m := tagDataSourceModel{Name: types.StringValue("blue"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}

func TestOfflineTokenTagListCancelPreservesState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "")
	if err != nil {
		t.Fatal(err)
	}
	d := &tagDataSource{clients: &ProviderClients{Auth: authToken, Native: native, Legacy: cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture-token"))}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	prior := tagDataSourceModel{Filters: []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringValue("tag-id")}}}, ID: types.StringValue("old")}
	if diags := state.Set(context.Background(), &prior); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}

func TestOfflineNativeTagConfigure(t *testing.T) {
	ctx := context.Background()
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"))
	good := &ProviderClients{Auth: authBasic, Native: native, Legacy: legacy}
	for _, input := range []any{nil, "wrong", (*ProviderClients)(nil), &ProviderClients{Auth: authBasic, Native: native}, &ProviderClients{Auth: authBasic, Legacy: legacy}, &ProviderClients{Auth: authToken}, &ProviderClients{Auth: 0, Native: native, Legacy: legacy}} {
		d := &tagDataSource{clients: good}
		var response datasource.ConfigureResponse
		d.Configure(ctx, datasource.ConfigureRequest{ProviderData: input}, &response)
		if d.clients != nil || (input != nil && !response.Diagnostics.HasError()) {
			t.Fatal(input, response.Diagnostics)
		}
	}
}
