package provider

import (
	"bytes"
	"context"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"strings"
	"testing"
)

func TestOfflineConfigurePrivacy(t *testing.T) {
	for _, mode := range []string{"basic", "token", "missing"} {
		var log bytes.Buffer
		ctx := tflogtest.RootLogger(context.Background(), &log)
		p := New("offline")()
		var s provider.SchemaResponse
		p.Schema(ctx, provider.SchemaRequest{}, &s)
		typ := s.Schema.Type().TerraformType(ctx)
		attrs := map[string]tftypes.Value{}
		for k, v := range typ.(tftypes.Object).AttributeTypes {
			attrs[k] = tftypes.NewValue(v, nil)
		}
		attrs["location"] = tftypes.NewValue(tftypes.String, "next")
		if mode == "basic" {
			attrs["username"] = tftypes.NewValue(tftypes.String, "PRIVATE-CANARY")
			attrs["password"] = tftypes.NewValue(tftypes.String, "PRIVATE-CANARY")
		}
		if mode == "token" {
			attrs["token"] = tftypes.NewValue(tftypes.String, "PRIVATE-CANARY")
		}
		var out provider.ConfigureResponse
		p.Configure(ctx, provider.ConfigureRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: tftypes.NewValue(typ, attrs)}, TerraformVersion: "offline"}, &out)
		if mode == "missing" && !out.Diagnostics.HasError() {
			t.Fatal("missing diagnostic")
		}
		if mode != "missing" && (out.Diagnostics.HasError() || out.DataSourceData == nil) {
			t.Fatal("config")
		}
		if strings.Contains(log.String(), "PRIVATE-CANARY") {
			t.Fatal("log leak")
		}
	}
}
