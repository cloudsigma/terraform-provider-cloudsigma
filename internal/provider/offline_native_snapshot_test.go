package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOfflineNativeSnapshot(t *testing.T) {
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	for _, mode := range []authMode{authBasic, authToken} {
		for _, code := range []int{200, 401, 403, 404, 500, 302, 204} {
			for _, body := range []string{`{"uuid":"response","drive":{"uuid":"drive"},"name":"name","status":"available","timestamp":"time","resource_uri":"uri"}`, `{"drive":null}`, ``} {
				ctx := context.Background()
				calls := 0
				native := libraryFixture(t, code, body, &calls, "/api/2.0/snapshots/prior/", "")
				legacy := cs.NewClient(cs.NewTokenCredentialsProvider("fixture-token"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
					if mode == authBasic {
						t.Fatal("legacy fallback")
					}
					calls++
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"response","drive":{"uuid":"drive"}}`)), Request: q}, nil
				})}))
				r := &snapshotResource{}
				var conf resource.ConfigureResponse
				r.Configure(ctx, resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: mode, Native: native, Legacy: legacy}}, &conf)
				if conf.Diagnostics.HasError() {
					t.Fatal(conf.Diagnostics)
				}
				var sc resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &sc)
				state := tfsdk.State{Schema: sc.Schema}
				m := snapshotResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("prior"), Drive: types.StringValue("old"), Name: types.StringValue("old")}
				if d := state.Set(ctx, &m); d.HasError() {
					t.Fatal(d)
				}
				out := resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, &out)
				if calls != 1 {
					t.Fatal(calls)
				}
				success := mode == authToken || (code == 200 && strings.Contains(body, `"response"`))
				if success {
					if out.Diagnostics.HasError() {
						t.Fatal(out.Diagnostics)
					}
					var got snapshotResourceModel
					if d := out.State.Get(ctx, &got); d.HasError() || got.ID.ValueString() != "response" || got.Drive.ValueString() != "drive" {
						t.Fatal(got, d)
					}
				} else if code == 404 {
					if !out.State.Raw.IsNull() || out.Diagnostics.HasError() {
						t.Fatal(out)
					}
				} else if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) {
					t.Fatal("state/error", out)
				}
				// Reconfiguration must discard the native selection as well as legacy.
				r.Configure(ctx, resource.ConfigureRequest{}, &conf)
				if r.native != nil || r.client != nil {
					t.Fatal("stale client")
				}
			}
		}
	}
	// Cancellation does not fall back to legacy or remove existing state.
	http.DefaultTransport = original
	native, _ := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
	legacy := cs.NewClient(cs.NewTokenCredentialsProvider("fixture"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(*http.Request) (*http.Response, error) { t.Fatal("fallback"); return nil, errors.New("fallback") })}))
	r := &snapshotResource{native: native, client: legacy}
	var sc resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &sc)
	state := tfsdk.State{Schema: sc.Schema}
	_ = state.Set(context.Background(), &snapshotResourceModel{ID: types.StringValue("prior")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &out)
	if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(out)
	}
}
