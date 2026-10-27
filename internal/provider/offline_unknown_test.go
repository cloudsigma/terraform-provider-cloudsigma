package provider

import (
	"context"
	"testing"

	fp "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestOfflineNativeAuthMatrix(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		for _, variant := range []string{"known", "null-location", "unknown-location", "unknown-token", "unknown-username", "unknown-password", "null-token", "null-username", "null-password"} {
			p := New("fixture")()
			ctx := context.Background()
			var s fp.SchemaResponse
			p.Schema(ctx, fp.SchemaRequest{}, &s)
			typ := s.Schema.Type().TerraformType(ctx)
			a := map[string]tftypes.Value{}
			for k, v := range typ.(tftypes.Object).AttributeTypes {
				a[k] = tftypes.NewValue(v, nil)
			}
			a["location"] = tftypes.NewValue(tftypes.String, "zrh")
			effective := bits
			for i, k := range []string{"username", "password", "token"} {
				if bits&(1<<i) != 0 {
					a[k] = tftypes.NewValue(tftypes.String, "synthetic")
				}
				if variant == "unknown-"+k {
					a[k] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
					effective &^= 1 << i
				}
				if variant == "null-"+k {
					a[k] = tftypes.NewValue(tftypes.String, nil)
					effective &^= 1 << i
				}
			}
			if variant == "unknown-location" {
				a["location"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			}
			if variant == "null-location" {
				a["location"] = tftypes.NewValue(tftypes.String, nil)
			}
			var o fp.ConfigureResponse
			p.Configure(ctx, fp.ConfigureRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: tftypes.NewValue(typ, a)}}, &o)
			summary, detail := "", ""
			switch effective {
			case 0:
				summary = "Missing CloudSigma credentials"
				detail = "Ensure that one of the credential types is set: [token] or [username,password]"
			case 1:
				summary = "Incomplete CloudSigma username/password credentials"
				detail = `"password" must be set`
			case 2:
				summary = "Incomplete CloudSigma username/password credentials"
				detail = `"username" must be set`
			case 5, 6, 7:
				summary = "Ambiguous CloudSigma credentials"
				detail = "Only one of the credential type must be set: [token] or [username,password]"
			}
			if summary != "" {
				if len(o.Diagnostics) != 1 || o.Diagnostics[0].Summary() != summary || o.Diagnostics[0].Detail() != detail || o.DataSourceData != nil || o.ResourceData != nil {
					t.Fatalf("bits%d %s: %v", bits, variant, o.Diagnostics)
				}
				continue
			}
			if o.Diagnostics.HasError() {
				t.Fatalf("bits%d %s: %v", bits, variant, o.Diagnostics)
			}
			if variant == "unknown-location" {
				if o.DataSourceData != nil || o.ResourceData != nil {
					t.Fatal("unknown location constructed clients")
				}
				continue
			}
			c, ok := o.DataSourceData.(*ProviderClients)
			if !ok || c == nil || o.ResourceData != c || c.Legacy == nil || (effective == 3 && (c.Auth != authBasic || c.Native == nil)) || (effective == 4 && (c.Auth != authToken || c.Native == nil)) {
				t.Fatalf("selection bits%d %s", bits, variant)
			}
		}
	}
}
