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

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

func snapshotDeleteFixture(t *testing.T, code int, body string, calls *int, path, query string) *cloudsigmaapi.Client {
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
			if !ok || u != "fixture" || p != "fixture" || r.Method != "DELETE" || r.Header.Get("Accept-Encoding") != "identity" || r.URL.Path != path || r.URL.RawQuery != query {
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
func TestOfflineNativeSnapshotDelete(t *testing.T) {
	for _, mode := range []authMode{authBasic, authToken} {
		for _, code := range []int{200, 201, 202, 204, 302, 401, 403, 404, 409, 429, 500} {
			ctx := context.Background()
			calls := 0
			native := snapshotDeleteFixture(t, code, "", &calls, "/api/2.0/snapshots/prior/", "")
			legacy := cs.NewClient(cs.NewTokenCredentialsProvider("fixture"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
				if mode == authBasic {
					t.Fatal("fallback")
				}
				calls++
				if q.Method != "DELETE" || q.URL.Path != "/api/2.0/snapshots/prior/" || q.URL.RawQuery != "" {
					t.Fatal(q)
				}
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: q}, nil
			}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
			r := &snapshotResource{}
			var conf resource.ConfigureResponse
			r.Configure(ctx, resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: mode, Native: native, Legacy: legacy}}, &conf)
			var sc resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &sc)
			state := tfsdk.State{Schema: sc.Schema}
			m := snapshotResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("identity"), Drive: types.StringValue("drive"), Name: types.StringValue("name"), Status: types.StringUnknown(), Timestamp: types.StringNull(), ResourceURI: types.StringValue("uri")}
			if d := state.Set(ctx, &m); d.HasError() {
				t.Fatal(d)
			}
			out := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &out)
			success := code == 204 || (mode == authToken && code >= 200 && code < 300)
			if out.Diagnostics.HasError() == success || calls != 1 || !out.State.Raw.Equal(state.Raw) {
				t.Fatal(out.Diagnostics, calls, "state changed")
			}
		}
	}
}
func TestOfflineNativeSnapshotDeleteInvalid(t *testing.T) {
	for _, id := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue("../x"), types.StringValue("prior")} {
		ctx, cancel := context.WithCancel(context.Background())
		if id.ValueString() == "prior" {
			cancel()
		}
		defer cancel()
		calls := 0
		native := snapshotDeleteFixture(t, 204, "", &calls, "/api/2.0/snapshots/prior/", "")
		r := &snapshotResource{native: native}
		var sc resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sc)
		state := tfsdk.State{Schema: sc.Schema}
		_ = state.Set(context.Background(), &snapshotResourceModel{ID: id, UUID: types.StringValue("identity")})
		out := resource.DeleteResponse{State: state}
		r.Delete(ctx, resource.DeleteRequest{State: state}, &out)
		if !out.Diagnostics.HasError() || calls != 0 || !out.State.Raw.Equal(state.Raw) {
			t.Fatal(out.Diagnostics, calls)
		}
	}
}
