package provider

import (
	"context"
	"testing"

	legacy "github.com/cloudsigma/terraform-provider-cloudsigma/cloudsigma"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
)

// TestOfflineImportProtocol exercises the same protocol adapters/mux as main,
// without configuring a client or refreshing a resource. Import success here is
// identifier-only state, not existence verification or an import-to-no-op plan.
func TestOfflineImportProtocol(t *testing.T) {
	ctx := context.Background()
	upgraded, err := tf5to6server.UpgradeServer(ctx, legacy.Provider().GRPCProvider)
	if err != nil {
		t.Fatal(err)
	}
	server, err := tf6muxserver.NewMuxServer(ctx, providerserver.NewProtocol6(New("offline")()), func() tfprotov6.ProviderServer { return upgraded })
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil || len(schemas.Diagnostics) != 0 {
		t.Fatalf("schema: %v %v", err, schemas)
	}
	for _, name := range []string{"cloudsigma_tag", "cloudsigma_ssh_key", "cloudsigma_snapshot"} {
		t.Run(name, func(t *testing.T) {
			// Two requests on the same server catch stale identity/state reuse. Empty
			// identifiers characterize passthrough, not documented API ID validity.
			for _, id := range []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", ""} {
				got, err := server.ImportResourceState(ctx, &tfprotov6.ImportResourceStateRequest{TypeName: name, ID: id})
				if err != nil || got == nil {
					t.Fatalf("import: %v", err)
				}
				if len(got.Diagnostics) != 0 || got.Deferred != nil || len(got.ImportedResources) != 1 {
					t.Fatalf("unexpected import: %#v", got)
				}
				imported := got.ImportedResources[0]
				if imported.TypeName != name || imported.State == nil || len(imported.Private) != 0 {
					t.Fatalf("unexpected resource: %#v", imported)
				}
				typ := schemas.ResourceSchemas[name].ValueType()
				value, err := imported.State.Unmarshal(typ)
				if err != nil {
					t.Fatal(err)
				}
				expected := map[string]tftypes.Value{}
				for key, attrType := range typ.(tftypes.Object).AttributeTypes {
					expected[key] = tftypes.NewValue(attrType, nil)
				}
				expected["id"] = tftypes.NewValue(tftypes.String, id)
				if !value.Equal(tftypes.NewValue(typ, expected)) {
					t.Fatalf("identifier-only state mismatch: %s", value)
				}
			}
		})
	}
	for _, name := range []string{"cloudsigma_drive", "cloudsigma_server"} {
		t.Run(name+"_unsupported", func(t *testing.T) {
			got, err := server.ImportResourceState(ctx, &tfprotov6.ImportResourceStateRequest{TypeName: name, ID: "11111111-1111-4111-8111-111111111111"})
			if err != nil || got == nil {
				t.Fatalf("import: %v", err)
			}
			if len(got.ImportedResources) != 0 || len(got.Diagnostics) != 1 || got.Diagnostics[0].Severity != tfprotov6.DiagnosticSeverityError || got.Diagnostics[0].Summary != "resource "+name+" doesn't support import" {
				t.Fatalf("expected unsupported import: %#v", got)
			}
		})
	}
}
