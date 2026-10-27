package provider

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

func TestOfflineNativeConfigure(t *testing.T) {
	ctx := context.Background()
	native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	if err != nil {
		t.Fatal(err)
	}
	good := &ProviderClients{Native: native, Legacy: &cloudsigma.Client{}, Auth: authBasic}
	bad := []interface{}{nil, "wrong", &cloudsigma.Client{}, (*ProviderClients)(nil), &ProviderClients{}, &ProviderClients{Auth: authMode(99), Legacy: good.Legacy}, &ProviderClients{Auth: authBasic}}
	p := &cloudSigmaProvider{}
	for _, factory := range p.DataSources(ctx) {
		d := factory()
		conf := d.(datasource.DataSourceWithConfigure)
		for _, input := range bad {
			var ok datasource.ConfigureResponse
			conf.Configure(ctx, datasource.ConfigureRequest{ProviderData: good}, &ok)
			if ok.Diagnostics.HasError() {
				t.Fatal(ok.Diagnostics)
			}
			var out datasource.ConfigureResponse
			conf.Configure(ctx, datasource.ConfigureRequest{ProviderData: input}, &out)
			if out.Diagnostics.HasError() != (input != nil) {
				t.Fatalf("%T: %v", d, out.Diagnostics)
			}
			var schema datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &schema)
			read := datasource.ReadResponse{State: tfsdk.State{Schema: schema.Schema}}
			d.Read(ctx, datasource.ReadRequest{}, &read)
			if !read.Diagnostics.HasError() || read.Diagnostics[0].Summary() != "Unconfigured CloudSigma client" {
				t.Fatalf("%T stale selection", d)
			}
		}
	}
	for _, factory := range p.Resources(ctx) {
		r := factory()
		conf := r.(resource.ResourceWithConfigure)
		for _, input := range bad {
			var ok resource.ConfigureResponse
			conf.Configure(ctx, resource.ConfigureRequest{ProviderData: good}, &ok)
			if ok.Diagnostics.HasError() {
				t.Fatal(ok.Diagnostics)
			}
			var out resource.ConfigureResponse
			conf.Configure(ctx, resource.ConfigureRequest{ProviderData: input}, &out)
			if out.Diagnostics.HasError() != (input != nil) {
				t.Fatalf("%T: %v", r, out.Diagnostics)
			}
			if !reflect.ValueOf(r).Elem().FieldByName("client").IsNil() {
				t.Fatalf("%T stale selection", r)
			}
		}
	}
	d := &profileDataSource{}
	for _, c := range []*ProviderClients{{Auth: authBasic, Native: native}, {Auth: authToken, Native: native, Legacy: good.Legacy}} {
		var out datasource.ConfigureResponse
		d.Configure(ctx, datasource.ConfigureRequest{ProviderData: c}, &out)
		if out.Diagnostics.HasError() {
			t.Fatal(out.Diagnostics)
		}
	}
}

// Native cancellation is a deterministic transport failure with no legacy fallback.
func TestOfflineNativeState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	legacy := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: fixtureRT(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("legacy must not dispatch")
	})}))
	d := &profileDataSource{}
	var conf datasource.ConfigureResponse
	d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authBasic, Native: native, Legacy: legacy}}, &conf)
	var schema datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	model := profileDataSourceModel{Address: types.StringValue("a"), Company: types.StringValue("c"), FirstName: types.StringValue("f"), LastName: types.StringValue("l"), Title: types.StringValue("t"), ID: types.StringValue("prior"), UUID: types.StringValue("prior")}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{}, &out)
	if !out.Diagnostics.HasError() || out.Diagnostics[0].Summary() != "Unable to fetch user profile" || out.Diagnostics[0].Detail() != "cancelled" || calls != 0 || !out.State.Raw.Equal(state.Raw) {
		t.Fatalf("native failure/state/fallback: %v", out.Diagnostics)
	}
}
