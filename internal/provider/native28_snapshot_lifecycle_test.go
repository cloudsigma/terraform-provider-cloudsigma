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

// Exercise the real Create callback, including both real waiter delays. POST is
// intentionally still legacy: this test grants no mutation-contract authority.
func TestNative28SnapshotLifecycle(t *testing.T) {
	for _, basic := range []bool{true, false} {
		for _, fault := range []string{"", "wait404", "final500", "missingdrive", "progression", "malformed", "null", "invalidid", "privateprecision"} {
			t.Run(fmt.Sprintf("basic%v/%s", basic, fault), func(t *testing.T) {
				ctx := context.Background()
				order := []string{}
				reads := 0
				transport := func(native bool) fixtureRT {
					return fixtureRT(func(q *http.Request) (*http.Response, error) {
						order = append(order, fmt.Sprintf("%v:%s:%s", native, q.Method, q.URL.Path))
						code := 200
						body := `{"uuid":"drive","status":"unmounted"}`
						if strings.Contains(q.URL.Path, "snapshots") {
							body = `{"uuid":"created","drive":{"uuid":"drive"},"status":"available","name":"done","timestamp":"time","resource_uri":"uri"}`
							if q.Method == "POST" {
								if native {
									t.Fatal("native mutation")
								}
								if fault == "invalidid" {
									body = `{"uuid":"bad/id","drive":{"uuid":"drive"}}`
								}
								body = `{"objects":[` + body + `]}`
							} else {
								if native != basic {
									t.Fatal("wrong read dispatch")
								}
								reads++
								if fault == "progression" && reads == 1 {
									body = `{"uuid":"created","status":"creating"}`
								}
								if fault == "privateprecision" {
									body = `{"uuid":"created","drive":{"uuid":"drive","size":9007199254740993},"name":"done","status":"available","timestamp":"time","resource_uri":"uri","private":"DO_NOT_PROJECT"}`
								}
								if reads == 2 && fault == "malformed" {
									body = `{`
								}
								if reads == 2 && fault == "null" && basic {
									body = `null`
								}
								if fault == "wait404" {
									code = 404
									body = `{}`
								} else if reads == 2 && fault == "final500" {
									code = 500
									body = `{}`
								} else if reads == 2 && fault == "missingdrive" && basic {
									body = `{"uuid":"created"}`
								}
							}
						}
						return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
					})
				}
				legacy := cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("fixture", "fixture"), cs.WithHTTPClient(&http.Client{Transport: transport(false)}))
				old := http.DefaultTransport
				tr := old.(*http.Transport).Clone()
				tr.Proxy = nil
				tr.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					a, b := net.Pipe()
					go func() {
						defer b.Close()
						q, e := http.ReadRequest(bufio.NewReader(b))
						if e != nil {
							return
						}
						resp, e := transport(true).RoundTrip(q)
						if e != nil {
							return
						}
						defer resp.Body.Close()
						body, _ := io.ReadAll(resp.Body)
						_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", resp.StatusCode, len(body), body)
					}()
					return a, nil
				}
				http.DefaultTransport = tr
				defer func() { http.DefaultTransport = old }()
				native, e := cloudsigmaapi.New("zrh", "fixture", "fixture", "")
				if e != nil {
					t.Fatal(e)
				}
				r := &snapshotResource{}
				mode := authToken
				if basic {
					mode = authBasic
				}
				var configured resource.ConfigureResponse
				r.Configure(ctx, resource.ConfigureRequest{ProviderData: &ProviderClients{Legacy: legacy, Native: native, Auth: mode}}, &configured)
				if configured.Diagnostics.HasError() {
					t.Fatal(configured.Diagnostics)
				}
				var sc resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &sc)
				plan := tfsdk.Plan{Schema: sc.Schema}
				d := plan.Set(ctx, &snapshotResourceModel{Drive: types.StringValue("drive"), Name: types.StringValue("requested")})
				if d.HasError() {
					t.Fatal(d)
				}
				out := resource.CreateResponse{State: tfsdk.State{Schema: sc.Schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, &out)
				wantErr := fault == "malformed" || fault == "null" && basic || fault == "invalidid" || fault == "wait404" || fault == "final500" || (fault == "missingdrive" && basic)
				if out.Diagnostics.HasError() != wantErr {
					t.Fatal(out.Diagnostics, order)
				}
				wantReads := 2
				if fault == "wait404" {
					wantReads = 1
				}
				if fault == "progression" {
					wantReads = 3
				}
				if fault == "invalidid" {
					wantReads = 0
				}
				if reads != wantReads || len(order) != 2+wantReads {
					t.Fatal(order)
				}
				if !wantErr {
					var got snapshotResourceModel
					if d := out.State.Get(ctx, &got); d.HasError() || got.ID.ValueString() != "created" || got.Name.ValueString() != "done" {
						t.Fatal(got, d)
					}
				}
			})
		}
	}
}
