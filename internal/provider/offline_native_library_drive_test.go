package provider

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
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
func libraryFixture(t *testing.T, code int, body string, calls *int, path, query string) *cloudsigmaapi.Client {
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
			u, p, ok := r.BasicAuth()
			if !ok || u != "fixture" || p != "fixture" || r.Method != "GET" || r.Header.Get("Accept-Encoding") != "identity" || r.URL.Path != path || r.URL.RawQuery != query {
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
func TestOfflineNativeLibraryDrive(t *testing.T) {
	object := `{"uuid":"selected","name":"image","arch":"64","description":"desc","image_type":"preinst","media":"disk","os":"linux","size":9007199254740993,"status":"unmounted","storage_type":null,"category":["linux"],"paid":true}`
	filter := []migration.FilterModel{{Name: types.StringValue("category"), Values: []types.String{types.StringValue("linux")}}}
	for _, tc := range []struct {
		name         string
		uuid, search types.String
		filters      []migration.FilterModel
		body         string
		code         int
		want         string
	}{
		{"filter-arch", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("arch"), Values: []types.String{types.StringValue("64")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-description", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("description"), Values: []types.String{types.StringValue("desc")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-imagetype", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("imagetype"), Values: []types.String{types.StringValue("preinst")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-media", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("media"), Values: []types.String{types.StringValue("disk")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-name", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("name"), Values: []types.String{types.StringValue("image")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-os", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("os"), Values: []types.String{types.StringValue("linux")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-size", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("size"), Values: []types.String{types.StringValue("9007199254740993")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-status", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("status"), Values: []types.String{types.StringValue("unmounted")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-storagetype", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("storagetype"), Values: []types.String{types.StringValue("")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-uuid", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringValue("selected")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-paid", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("paid"), Values: []types.String{types.StringValue("true")}}}, `{"objects":[` + object + `]}`, 200, ""},
		{"alias", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("image_type")}}, `{"objects":[` + object + `]}`, 200, "Missing field"},
		{"unknownfilter", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringUnknown()}}, `{"objects":[` + object + `]}`, 200, "Missing field"},
		{"nullvalues", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid")}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"unknownvalues", types.StringNull(), types.StringNull(), []migration.FilterModel{{Name: types.StringValue("uuid"), Values: []types.String{types.StringUnknown()}}}, `{"objects":[` + object + `]}`, 200, "No search results"},
		{"singleton", types.StringValue("selected"), types.StringNull(), nil, object, 200, ""},
		{"name", types.StringNull(), types.StringValue("image"), nil, `{"objects":[` + object + `]}`, 200, ""},
		{"filter-precedence", types.StringValue("ignored"), types.StringValue("ignored"), filter, `{"objects":[` + object + `]}`, 200, ""},
		{"mismatch", types.StringValue("selected"), types.StringValue("wrong"), nil, object, 200, "Ambiguous search result"},
		{"404", types.StringValue("selected"), types.StringNull(), nil, `PRIVATE`, 404, "No search results"},
		{"500", types.StringValue("selected"), types.StringNull(), nil, `PRIVATE`, 500, "Unable to get library drive"},
		{"invalid", types.StringValue("selected"), types.StringNull(), nil, `{"size":"bad"}`, 200, "Unable to get library drive"},
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
				path := "/api/2.0/libdrives/"
				query := "limit=0"
				if len(tc.filters) == 0 {
					if tc.uuid.ValueString() != "" {
						path += tc.uuid.ValueString() + "/"
						query = ""
					} else if tc.search.ValueString() != "" {
						query += "&name=" + tc.search.ValueString()
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
				d := &libraryDriveDataSource{}
				var conf datasource.ConfigureResponse
				d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: auth, Native: native, Legacy: legacy}}, &conf)
				if conf.Diagnostics.HasError() {
					t.Fatal(conf.Diagnostics)
				}
				var schema datasource.SchemaResponse
				d.Schema(ctx, datasource.SchemaRequest{}, &schema)
				state := tfsdk.State{Schema: schema.Schema}
				prior := libraryDriveDataSourceModel{ID: types.StringValue("old"), UUID: tc.uuid, Name: tc.search, Filters: tc.filters}
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
					var got libraryDriveDataSourceModel
					if x := out.State.Get(ctx, &got); x.HasError() {
						t.Fatal(x)
					}
					if out.Diagnostics.HasError() || got.ID.ValueString() != "selected" || got.UUID.ValueString() != "selected" || got.Size.ValueInt64() != 9007199254740993 || got.Architecture.ValueString() != "64" || got.Description.ValueString() != "desc" || got.Name.ValueString() != "image" || got.ImageType.ValueString() != "preinst" || got.Media.ValueString() != "disk" || got.OS.ValueString() != "linux" || got.Status.ValueString() != "unmounted" || got.StorageType.ValueString() != "" {
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

func TestOfflineLibraryCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	d := &libraryDriveDataSource{clients: &ProviderClients{Auth: authBasic, Native: native}}
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	m := libraryDriveDataSourceModel{UUID: types.StringValue("selected"), ID: types.StringValue("old")}
	if x := state.Set(context.Background(), &m); x.HasError() {
		t.Fatal(x)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: schema.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out.Diagnostics)
	}
}
