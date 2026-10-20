package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

func TestOfflineNativeSSHKeyUpdate(t *testing.T) {
	for _, mode := range []authMode{authBasic, authToken} {
		for _, code := range []int{200, 201, 202, 204, 302, 401, 404, 500} {
			for _, value := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue(""), types.StringValue("prior-private"), types.StringValue("SYNTHETIC-PRIVATE")} {
				ctx := context.Background()
				calls := 0
				var wire map[string]string
				body := `{"uuid":"returned","name":"returned-name","private_key":"returned-private","public_key":"returned-public"}`
				inspect := func(q *http.Request) {
					calls++
					if q.Method != "PUT" || q.URL.Path != "/api/2.0/keypairs/prior/" || q.URL.RawQuery != "" {
						t.Error("route")
					}
					raw, _ := io.ReadAll(q.Body)
					if e := json.Unmarshal(raw, &wire); e != nil {
						t.Error(e)
					}
				}
				old := http.DefaultTransport
				tr := old.(*http.Transport).Clone()
				tr.Proxy = nil
				tr.DialTLSContext = func(context.Context, string, string) (net.Conn, error) {
					a, b := net.Pipe()
					go func() {
						defer b.Close()
						q, e := http.ReadRequest(bufio.NewReader(b))
						if e != nil {
							return
						}
						inspect(q)
						u, p, ok := q.BasicAuth()
						if !ok || u != "fixture" || p != "fixture" {
							t.Error("auth")
						}
						_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", code, len(body), body)
					}()
					return a, nil
				}
				http.DefaultTransport = tr
				native, e := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
				http.DefaultTransport = old
				if e != nil {
					t.Fatal(e)
				}
				legacy := cs.NewClient(cs.NewTokenCredentialsProvider("fixture"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
					if mode == authBasic {
						t.Fatal("fallback")
					}
					inspect(q)
					return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
				}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
				r := &sshKeyResource{}
				var conf resource.ConfigureResponse
				r.Configure(ctx, resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: mode, Native: native, Legacy: legacy}}, &conf)
				var sc resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &sc)
				state := tfsdk.State{Schema: sc.Schema}
				m := sshKeyResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("prior"), Name: types.StringValue("name"), PrivateKey: types.StringValue("prior-private"), PublicKey: types.StringValue("prior-public")}
				if d := state.Set(ctx, &m); d.HasError() {
					t.Fatal(d)
				}
				m.PrivateKey = value
				m.PublicKey = types.StringValue("\nSYNTHETIC-PUBLIC\n")
				plan := tfsdk.Plan{Schema: sc.Schema}
				if d := plan.Set(ctx, &m); d.HasError() {
					t.Fatal(d)
				}
				out := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &out)
				success := code == 200 || (mode == authToken && code >= 200 && code < 300)
				if out.Diagnostics.HasError() == success || calls != 1 {
					t.Fatal(mode, code, out.Diagnostics, calls)
				}
				expected := map[string]string{"name": "name", "uuid": "prior", "public_key": "\nSYNTHETIC-PUBLIC\n"}
				if !value.Equal(types.StringValue("prior-private")) && value.ValueString() != "" {
					expected["private_key"] = value.ValueString()
				}
				if !reflect.DeepEqual(wire, expected) {
					t.Fatal(wire, expected)
				}
				if success {
					var got sshKeyResourceModel
					if d := out.State.Get(ctx, &got); d.HasError() {
						t.Fatal(d)
					}
					if got.ID.ValueString() != "returned" || got.UUID.ValueString() != "returned" || got.PrivateKey.ValueString() != "returned-private" || got.PublicKey.ValueString() != "returned-public" || got.Name.ValueString() != "returned-name" {
						t.Fatal("projection")
					}
				} else if !out.State.Raw.Equal(state.Raw) {
					t.Fatal("state changed")
				}
			}
		}
	}
}
func TestOfflineNativeSSHKeyUpdateInvalid(t *testing.T) {
	for _, id := range []types.String{types.StringNull(), types.StringUnknown(), types.StringValue("../x"), types.StringValue("prior")} {
		ctx, cancel := context.WithCancel(context.Background())
		if id.ValueString() == "prior" {
			cancel()
		}
		calls := 0
		native := keypairDeleteFixture(t, 200, `{}`, &calls, "", "")
		r := &sshKeyResource{native: native}
		var sc resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sc)
		m := sshKeyResourceModel{ID: id, UUID: types.StringNull(), Name: types.StringValue("name"), PrivateKey: types.StringNull(), PublicKey: types.StringNull()}
		state := tfsdk.State{Schema: sc.Schema}
		state.Set(ctx, &m)
		plan := tfsdk.Plan{Schema: sc.Schema}
		plan.Set(ctx, &m)
		out := resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &out)
		cancel()
		if !out.Diagnostics.HasError() || calls != 0 || !out.State.Raw.Equal(state.Raw) {
			t.Fatal(out.Diagnostics, calls)
		}
	}
}
