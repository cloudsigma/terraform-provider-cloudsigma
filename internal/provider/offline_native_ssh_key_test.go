package provider

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const sshKeyFixture = `{"uuid":"response","name":"fixture","private_key":"SYNTHETIC-PRIVATE-SENTINEL","public_key":"SYNTHETIC-PUBLIC-SENTINEL"}`

func TestOfflineNativeSSHKey(t *testing.T) {
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	for _, mode := range []authMode{authBasic, authToken} {
		for _, code := range []int{200, 401, 403, 404, 500, 302, 204} {
			for _, body := range []string{sshKeyFixture, `null`, `{}`, `{"private_key":null,"public_key":null}`, `{"private_key":[]}`, `{"meta":[]}`, ``} {
				ctx := context.Background()
				calls := 0
				native := libraryFixture(t, code, body, &calls, "/api/2.0/keypairs/prior/", "")
				legacy := cs.NewClient(cs.NewTokenCredentialsProvider("fixture-token"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
					if mode == authBasic {
						t.Fatal("legacy fallback")
					}
					calls++
					if q.URL.Path != "/api/2.0/keypairs/prior/" || q.URL.RawQuery != "" || q.Header.Get("Authorization") != "Bearer fixture-token" {
						t.Fatal("legacy selection")
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(sshKeyFixture)), Request: q}, nil
				})}))
				r := &sshKeyResource{}
				var conf resource.ConfigureResponse
				r.Configure(ctx, resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: mode, Native: native, Legacy: legacy}}, &conf)
				if conf.Diagnostics.HasError() {
					t.Fatal(conf.Diagnostics)
				}
				var sc resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &sc)
				state := tfsdk.State{Schema: sc.Schema}
				m := sshKeyResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("prior"), Name: types.StringValue("old"), PrivateKey: types.StringValue("SYNTHETIC-OLD-PRIVATE"), PublicKey: types.StringUnknown()}
				if d := state.Set(ctx, &m); d.HasError() {
					t.Fatal(d)
				}
				out := resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, &out)
				if calls != 1 {
					t.Fatal(calls)
				}
				success := mode == authToken || code == 200 && body != "" && !strings.Contains(body, "[]")
				if success {
					if out.Diagnostics.HasError() {
						t.Fatal(out.Diagnostics)
					}
					var got sshKeyResourceModel
					if d := out.State.Get(ctx, &got); d.HasError() {
						t.Fatal(d)
					}
					want := sshKeyResourceModel{Name: types.StringValue(""), ID: types.StringValue(""), UUID: types.StringValue(""), PrivateKey: types.StringValue(""), PublicKey: types.StringValue("")}
					if mode == authToken || body == sshKeyFixture {
						want = sshKeyResourceModel{Name: types.StringValue("fixture"), ID: types.StringValue("response"), UUID: types.StringValue("response"), PrivateKey: types.StringValue("SYNTHETIC-PRIVATE-SENTINEL"), PublicKey: types.StringValue("SYNTHETIC-PUBLIC-SENTINEL")}
					}
					rebuilt := tfsdk.State{Schema: sc.Schema}
					if d := rebuilt.Set(ctx, &want); d.HasError() {
						t.Fatal(d)
					}
					if !rebuilt.Raw.Equal(out.State.Raw) {
						t.Fatal("full projection mismatch")
					}
				} else if code == 404 {
					if !out.State.Raw.IsNull() || out.Diagnostics.HasError() {
						t.Fatal("404 state")
					}
				} else {
					if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) {
						t.Fatal("error state")
					}
				}
				r.Configure(ctx, resource.ConfigureRequest{}, &conf)
				if r.native != nil || r.client != nil {
					t.Fatal("stale configure")
				}
			}
		}
	}
}

func TestOfflineNativeSSHKeyStateBoundaries(t *testing.T) {
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	for _, kind := range []string{"null-id", "unknown-id", "cancel", "otp-challenge"} {
		ctx := context.Background()
		calls := 0
		// A synthetic challenge fixture is not a claim about normative OTP wire syntax.
		tr := original.(*http.Transport).Clone()
		tr.Proxy = nil
		tr.DialTLSContext = func(context.Context, string, string) (net.Conn, error) {
			a, b := net.Pipe()
			go func() {
				defer b.Close()
				q, e := http.ReadRequest(bufio.NewReader(b))
				if e != nil {
					return
				}
				calls++
				if q.URL.Path != "/api/2.0/keypairs/prior/" {
					return
				}
				body := "SYNTHETIC-PRIVATE-ERROR"
				_, _ = fmt.Fprintf(b, "HTTP/1.1 401 Unauthorized\r\nWWW-Authenticate: synthetic-otp-challenge\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
			}()
			return a, nil
		}
		http.DefaultTransport = tr
		native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
		http.DefaultTransport = original
		if err != nil {
			t.Fatal(err)
		}
		legacy := cs.NewClient(cs.NewTokenCredentialsProvider("fixture"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(*http.Request) (*http.Response, error) { t.Fatal("fallback"); return nil, nil })}))
		r := &sshKeyResource{native: native, client: legacy}
		var sc resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &sc)
		m := sshKeyResourceModel{ID: types.StringValue("prior"), PrivateKey: types.StringUnknown(), PublicKey: types.StringNull()}
		if kind == "null-id" {
			m.ID = types.StringNull()
		}
		if kind == "unknown-id" {
			m.ID = types.StringUnknown()
		}
		state := tfsdk.State{Schema: sc.Schema}
		if d := state.Set(ctx, &m); d.HasError() {
			t.Fatal(d)
		}
		if kind == "cancel" {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		out := resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, &out)
		if !out.Diagnostics.HasError() || !out.State.Raw.Equal(state.Raw) {
			t.Fatal("state changed")
		}
		wantCalls := 0
		if kind == "otp-challenge" {
			wantCalls = 1
		}
		if calls != wantCalls {
			t.Fatal(kind, calls)
		}
		for _, d := range out.Diagnostics {
			if strings.Contains(d.Detail(), "SYNTHETIC-PRIVATE-ERROR") || strings.Contains(d.Detail(), "synthetic-otp-challenge") {
				t.Fatal("diagnostic leak")
			}
		}
	}
}
