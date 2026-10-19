package provider

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestOfflineConfigurePrivacy(t *testing.T) {
	for _, mode := range []string{"basic", "token", "missing", "ambiguous", "username", "password"} {
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
		if mode == "basic" || mode == "ambiguous" {
			attrs["username"] = tftypes.NewValue(tftypes.String, "PRIVATE-CANARY")
			attrs["password"] = tftypes.NewValue(tftypes.String, "PRIVATE-CANARY")
		}
		if mode == "token" || mode == "ambiguous" {
			attrs["token"] = tftypes.NewValue(tftypes.String, "PRIVATE-CANARY")
		}
		if mode == "username" {
			attrs["username"] = tftypes.NewValue(tftypes.String, "fixture")
		}
		if mode == "password" {
			attrs["password"] = tftypes.NewValue(tftypes.String, "fixture")
		}
		var out provider.ConfigureResponse
		p.Configure(ctx, provider.ConfigureRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: tftypes.NewValue(typ, attrs)}, TerraformVersion: "offline"}, &out)
		if mode != "basic" && mode != "token" && !out.Diagnostics.HasError() {
			t.Fatal("missing diagnostic")
		}
		if (mode == "basic" || mode == "token") && (out.Diagnostics.HasError() || out.DataSourceData == nil) {
			t.Fatal("config")
		}
		if strings.Contains(log.String(), "PRIVATE-CANARY") {
			t.Fatal("log leak")
		}
	}
}
