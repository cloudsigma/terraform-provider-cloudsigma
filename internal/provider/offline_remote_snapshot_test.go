package provider

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOfflineRemoteSnapshotNativeCreate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/2.0/remotesnapshots/" {
			t.Fatalf("route %s %s", req.Method, req.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"objects":[{"uuid":"fixture","resource_uri":"/api/2.0/remotesnapshots/fixture/","location":"far","status":"creating"}]}`)
	}))
	defer server.Close()
	old := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
		},
	}
	defer func() { http.DefaultTransport = old }()
	native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	if err != nil {
		t.Fatal(err)
	}
	r := &remoteSnapshotResource{native: native}
	ctx := context.Background()
	var s resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &s)
	plan := tfsdk.Plan{Schema: s.Schema}
	if d := plan.Set(ctx, &remoteSnapshotModel{Drive: types.StringValue("drive"), Location: types.StringValue("far")}); d.HasError() {
		t.Fatal(d)
	}
	out := resource.CreateResponse{State: tfsdk.State{Schema: s.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &out)
	if out.Diagnostics.HasError() {
		t.Fatal(out.Diagnostics)
	}
	var got remoteSnapshotModel
	if d := out.State.Get(ctx, &got); d.HasError() || got.ID.ValueString() != "fixture" {
		t.Fatalf("state=%v diags=%v", got, d)
	}
}
