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

func TestOfflineNativeDrive(t *testing.T) {
	object := `{"uuid":"selected","name":"image","arch":"64","description":"desc","image_type":"preinst","media":"disk","os":"linux","size":9007199254740993,"status":"unmounted","storage_type":null,"allow_multimount":true}`
	filter := []migration.FilterModel{{Name: types.StringValue("allowmultimount"), Values: []types.String{types.StringValue("true")}}}
	for _, tc := range []struct {
		name         string
		uuid, search types.String
		filters      []migration.FilterModel
		body         string
		code         int
		want         string
	}{
		{"filter-media", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("media"), Values: []types.String{types.StringValue("disk")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-name", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("name"), Values: []types.String{types.StringValue("image")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-size", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("size"), Values: []types.String{types.StringValue("9007199254740993")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-status", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("status"), Values: []types.String{types.StringValue("unmounted")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-storagetype", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("storagetype"), Values: []types.String{types.StringValue("")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-uuid", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringValue("selected")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"alias", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("image_type")}}, `{"objects":[` + object + `]}`, 200, "Missing field"},
		{"unknownfilter", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringUnknown()}}, `{"objects":[` + object + `]}`, 200, "Missing field"},
		{"nullvalues", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid")}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"unknownvalues", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringUnknown()}}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"singleton", types.StringValue("selected"), types.StringNull(), nil, object, 200, ""},
		{"name", types.StringNull(), types.StringValue("image"), nil, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-precedence", types.StringValue("ignored"), types.StringValue("ignored"), filter, `{"objects":[` + object + `]}`, 200, ""},
		{"mismatch", types.StringValue("selected"), types.StringValue("wrong"), nil, object, 200, "Ambiguous search result"},
		{"404", types.StringValue("selected"), types.StringNull(), nil, `PRIVATE`, 404, "No search results"},
		{"500", types.StringValue("selected"), types.StringNull(), nil, `PRIVATE`, 500, "Unable to get drive"},
		{"invalid", types.StringValue("selected"), types.StringNull(), nil, `{"size":"bad"}`, 200, "Unable to get drive"},
		{"empty", types.StringNull(), types.StringValue("image"), nil, `{"objects":[]}`, 200, "No search results"},
		{"null", types.StringNull(), types.StringValue("image"), nil, `{"objects":null}`, 200, "No search results"},
		{"many", types.StringNull(), types.StringValue("image"), nil, `{"objects":[{},{}]}`, 200, "Too many search results"},
		{"missing", types.StringNull(), types.StringNull(), nil, `{}`, 200, "Missing required attributes"},
		{"unknown", types.StringUnknown(), types.StringUnknown(), nil, `{}`, 200, "Missing required attributes"},
	} {
		for _, token := range []bool{false, true} {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				calls := 0
				path := "/api/2.0/drives/detail/"
				query := "limit=0"
				if len(tc.filters) == 0 {
					if tc.uuid.ValueString() != "" {
						path = "/api/2.0/drives/" + tc.uuid.ValueString() + "/"
						query = ""
					} else if tc.search.ValueString() != "" {
						query += "&name=" + tc.search.ValueString()
					}
				}
				native := libraryFixtureAuth(t, tc.code, tc.body, &calls, path, query, token)
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
				d := &driveDataSource{}
				var conf datasource.ConfigureResponse
				d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: auth, Native: native, Legacy: legacy}}, &conf)
				if conf.Diagnostics.HasError() {
					t.Fatal(conf.Diagnostics)
				}
				var schema datasource.SchemaResponse
				d.Schema(ctx, datasource.SchemaRequest{}, &schema)
				state := tfsdk.State{Schema: schema.Schema}
				prior := driveDataSourceModel{ID: types.StringValue("old"), UUID: tc.uuid, Name: tc.search, Filters: tc.filters}
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
					var got driveDataSourceModel
					if x := out.State.Get(ctx, &got); x.HasError() {
						t.Fatal(x)
					}
					if out.Diagnostics.HasError() || got.ID.ValueString() != "selected" || got.UUID.ValueString() != "selected" || got.Size.ValueInt64() != 9007199254740993 || got.Name.ValueString() != "image" || got.Status.ValueString() != "unmounted" || got.StorageType.ValueString() != "" {
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

func TestOfflineDriveCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	d := &driveDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	m := driveDataSourceModel{UUID: types.StringValue("selected"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}

func TestOfflineTokenDriveCancelPreservesState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "")
	if err != nil {
		t.Fatal(err)
	}
	d := &driveDataSource{clients: &ProviderClients{Auth: authToken, Native: native}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	prior := driveDataSourceModel{UUID: types.StringValue("selected"), ID: types.StringValue("old"), Name: types.StringValue("old-name")}
	if diags := state.Set(context.Background(), &prior); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}

func TestOfflineDriveConfigureRequiresNative(t *testing.T) {
	legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"))
	d := &driveDataSource{}
	var response datasource.ConfigureResponse
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Legacy: legacy}}, &response)
	if d.clients != nil || !response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
}
