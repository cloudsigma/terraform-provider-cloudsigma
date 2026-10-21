package cloudsigma

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestNative32Ack(t *testing.T) {
	for _, raw := range []bool{false, true} {
		t.Run(fmt.Sprintf("raw%t", raw), func(t *testing.T) { native32Ack(t, raw) })
	}
}
func native32Ack(t *testing.T, raw bool) {
	for bits := 0; bits < 8; bits++ {
		for _, fault := range []string{"401", "404", "500", "empty", "malformed", "disconnect", "cancel-body", "deadline-body", "ack-cancel-before", "ack-cancel-after", "ack-deadline-before", "ack-deadline-after", "ack-read-error"} {
			t.Run(fmt.Sprintf("%d/%s", bits, fault), func(t *testing.T) {
				old := http.DefaultTransport
				defer func() { http.DefaultTransport = old }()
				http.DefaultTransport = &http.Transport{}
				u, p, tok := "", "", ""
				if bits&1 != 0 {
					u = "u"
				}
				if bits&2 != 0 {
					p = "p"
				}
				if bits&4 != 0 {
					tok = "t"
				}
				incomplete := tok == "" && (u == "" || p == "")
				conf := &Config{Username: u, Password: p, Token: tok, Location: "zrh"}
				conf.loadAndValidate(context.Background(), "fixture")
				clients, e := conf.clients()
				if e != nil {
					t.Fatal(e)
				}
				ctx, cancel := context.WithCancel(context.Background())
				// A real caller deadline expires at the selected acknowledgment boundary.
				// It expires only when the selected PUT acknowledgment boundary waits for it.
				if strings.Contains(fault, "deadline") && strings.HasPrefix(fault, "ack-") {
					ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
				}
				defer cancel()
				var end context.CancelFunc
				routes := []string{}
				counts := map[string]int{}
				native, legacy, gets, posts, puts := 0, 0, 0, 0, 0
				handler := func(r *http.Request) (*http.Response, error) {
					if tok != "" {
						if r.Header.Get("Authorization") != "Bearer "+tok {
							t.Error("token")
						}
					} else {
						a, b, _ := r.BasicAuth()
						if a != u || b != p {
							t.Error("basic")
						}
					}
					body := `{"uuid":"created","status":"running"}`
					code := 200
					if r.Method == "POST" {
						posts++
						code = 201
						body = `{"objects":[{"uuid":"created"}]}`
					}
					if r.Method == "PUT" {
						puts++
					}
					if r.Method == "GET" {
						gets++
					}
					if r.Method == "PUT" {
						{
							switch fault {
							case "ack-cancel-before", "ack-cancel-after", "ack-deadline-before", "ack-deadline-after", "ack-read-error":
								finish := cancel
								if strings.Contains(fault, "deadline") {
									finish = func() { <-ctx.Done() }
								}
								ack := &native31AckBody{data: []byte(`{"uuid":"created"}`), readError: fault == "ack-read-error"}
								if strings.HasSuffix(fault, "before") {
									finish()
								} else if strings.HasSuffix(fault, "after") {
									ack.finish = finish
								}
								return &http.Response{StatusCode: 200, Header: make(http.Header), Body: ack, Request: r}, nil
							case "401":
								code = 401
							case "404":
								code = 404
							case "500":
								code = 500
							case "empty":
								body = ""
							case "malformed":
								body = "{"
							case "disconnect":
								return nil, fmt.Errorf("synthetic disconnect")
							default:
								// Return actual HTTP response headers then block inside Body.Read using net.Pipe.
								if fault == "deadline-body" {
									var child context.Context
									child, end = context.WithTimeout(ctx, 40*time.Millisecond)
									ctx = child
								} else {
									go func() { time.Sleep(20 * time.Millisecond); cancel() }()
								}
								tr := &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
									a, b := net.Pipe()
									go func() {
										defer b.Close()
										_, err := http.ReadRequest(bufio.NewReader(b))
										if err != nil {
											return
										}
										fmt.Fprint(b, "HTTP/1.1 200 OK\r\nContent-Length: 999\r\nConnection: close\r\n\r\n{")
										<-ctx.Done()
									}()
									return a, nil
								}}
								rr := r.Clone(ctx)
								return (&http.Client{Transport: tr}).Do(rr)
							}
						}
					}
					return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}
				if clients.native != nil {
					clients.native = native31Transport(t, func(r *http.Request) (*http.Response, error) {
						native++
						counts["native/"+r.Method]++
						routes = append(routes, r.Method+" "+r.URL.RequestURI())
						return handler(r)
					})
				}
				var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
				if tok != "" {
					creds = cs.NewTokenCredentialsProvider(tok)
				}
				clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					legacy++
					counts["legacy/"+r.Method]++
					routes = append(routes, r.Method+" "+r.URL.RequestURI())
					return handler(r)
				})}))
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "before", "cpu": 2000, "memory": 536870912, "smp": 2, "drive": []interface{}{map[string]interface{}{"uuid": "added", "device": "virtio"}}})
				var meta interface{} = clients
				if raw {
					meta = clients.legacy
				}
				ds := resourceCloudSigmaServerCreate(ctx, d, meta)
				if end != nil {
					end()
				}
				ack := strings.HasPrefix(fault, "ack-")
				strict := !raw && bits == 3 && !native32Baseline
				// Legacy SDK accepts valid bytes plus read-error, and complete
				// acknowledgments despite canceled context. Later native GET
				// still refuses canceled context on configured Basic baseline.
				wantError := incomplete || !ack || strict || (!raw && bits == 3 && fault != "ack-read-error")
				if ds.HasError() != wantError {
					t.Fatal(ds, fault, raw, bits)
				}
				if incomplete {
					if native+legacy != 0 || d.Id() != "" {
						t.Fatal(native, legacy, d.Id())
					}
					return
				}
				if posts != 1 || puts != 1 || d.Id() != "created" {
					t.Fatal(posts, puts, d.Id())
				}
				wantGets := 0
				if ack && !strict && (raw || bits != 3 || fault == "ack-read-error") {
					wantGets = 2
				}
				if gets != wantGets {
					t.Fatal("gets", gets, wantGets, routes)
				}
				wn := 0
				if !raw && bits == 3 {
					wn = 1 + wantGets
					if !native32Baseline {
						wn++
					}
				}
				if native != wn || legacy != 2+wantGets-wn {
					t.Fatal("dispatch", native, legacy, wn)
				}
				expected := []string{"POST /api/2.0/servers/", "PUT /api/2.0/servers/created/"}
				for i := 0; i < wantGets; i++ {
					expected = append(expected, "GET /api/2.0/servers/created/")
				}
				if fmt.Sprint(routes) != fmt.Sprint(expected) {
					t.Fatal(routes, expected)
				}
				t.Logf("POST=%d PUT=%d GET=%d retained=%s", posts, puts, gets, d.Id())
			})
		}
	}
}
