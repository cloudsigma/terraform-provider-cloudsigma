package provider

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

// TestOfflineTagResourceImportRead verifies the compatibility boundary promised
// by the passthrough importer: imported ID-only state can be refreshed through
// the native GET path without invoking the SDK or requiring a create operation.
func TestOfflineTagResourceImportRead(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/tags/tag-imported/" {
			t.Fatalf("route=%s %s", req.Method, req.URL.Path)
		}
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture" || password != "fixture" {
			t.Fatalf("authentication=%q", req.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"uuid":"tag-imported","name":"imported","resource_uri":"/api/2.0/tags/tag-imported/"}`)
	}))
	defer server.Close()

	old := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // synthetic test server only
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		},
	}
	defer func() { http.DefaultTransport = old }()

	native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	if err != nil {
		t.Fatal(err)
	}
	r := &tagResource{native: native}
	ctx := context.Background()
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diags := state.Set(ctx, &tagResourceModel{ID: types.StringValue("tag-imported")}); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var got tagResourceModel
	if diags := response.State.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if got.ID.ValueString() != "tag-imported" || got.Name.ValueString() != "imported" || got.ResourceURI.ValueString() != "/api/2.0/tags/tag-imported/" {
		t.Fatalf("state=%+v", got)
	}
}

func TestOfflineTagResourceImportReadMissingRemovesState(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/api/2.0/tags/missing/" {
			t.Fatalf("route=%s %s", req.Method, req.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	old := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // synthetic test server only
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		},
	}
	defer func() { http.DefaultTransport = old }()
	native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	if err != nil {
		t.Fatal(err)
	}
	r := &tagResource{native: native}
	ctx := context.Background()
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diags := state.Set(ctx, &tagResourceModel{ID: types.StringValue("missing")}); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if response.State.Raw.IsKnown() {
		// RemoveResource produces a null object; a non-null state would retain a
		// missing imported object and cause Terraform to manage stale identity.
		if !response.State.Raw.IsNull() {
			t.Fatalf("missing tag retained state: %s", response.State.Raw)
		}
	}
}
