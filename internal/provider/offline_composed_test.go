package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"

	ro "github.com/cloudsigma/terraform-provider-cloudsigma/readonly"
)

func TestOfflineConfiguredReadonly(t *testing.T) {
	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)
	f := ro.New("fixture-account", map[string]bool{})
	f.Base = fixtureRT(func(q *http.Request) (*http.Response, error) {
		if q.URL.Path != "/api/2.0/profile/" {
			t.Fatal("unexpected path")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"fixture-account","company":"PRIVATE-CANARY","address":"PRIVATE-CANARY"}`)), Request: q}, nil
	})
	oldClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: f}
	defer func() { http.DefaultClient = oldClient }()
	old := http.DefaultTransport
	http.DefaultTransport = f
	defer func() { http.DefaultTransport = old }()
	p := New("offline")()
	var s provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &s)
	typ := s.Schema.Type().TerraformType(ctx)
	attrs := map[string]tftypes.Value{}
	for k, v := range typ.(tftypes.Object).AttributeTypes {
		attrs[k] = tftypes.NewValue(v, nil)
	}
	for k, v := range map[string]string{"username": "PRIVATE-CANARY", "password": "PRIVATE-CANARY", "location": "next"} {
		attrs[k] = tftypes.NewValue(tftypes.String, v)
	}
	var config provider.ConfigureResponse
	p.Configure(ctx, provider.ConfigureRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: tftypes.NewValue(typ, attrs)}, TerraformVersion: "offline"}, &config)
	if config.Diagnostics.HasError() {
		t.Fatal("configure")
	}
	d := NewProfileDataSource()
	var ds datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &ds)
	var conf datasource.ConfigureResponse
	d.(datasource.DataSourceWithConfigure).Configure(ctx, datasource.ConfigureRequest{ProviderData: config.DataSourceData}, &conf)
	out := datasource.ReadResponse{State: tfsdk.State{Schema: ds.Schema}}
	d.Read(ctx, datasource.ReadRequest{}, &out)
	if out.Diagnostics.HasError() || !f.Admitted || f.Calls != 1 {
		t.Fatalf("composed admission read calls=%d admitted=%v diagnostics=%v", f.Calls, f.Admitted, out.Diagnostics)
	}
	if strings.Contains(logs.String(), "PRIVATE-CANARY") {
		t.Fatal("logging leak")
	}
}
