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

func TestNative26AttachmentFaults(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		for _, fault := range []string{"401", "404", "500", "empty", "malformed", "disconnect", "cancel-body", "deadline-body"} {
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
				defer cancel()
				var end context.CancelFunc
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
					body := `{"uuid":"created","status":"unmounted"}`
					code := 200
					if r.Method == "POST" {
						posts++
						code = 202
						body = `{"objects":[{"uuid":"created"}]}`
					}
					if r.Method == "PUT" {
						puts++
						t.Error("PUT after failed GET")
					}
					if r.Method == "GET" {
						gets++
						if gets == 2 {
							switch fault {
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
					clients.native = native26Transport(t, func(r *http.Request) (*http.Response, error) { native++; return handler(r) })
				}
				var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
				if tok != "" {
					creds = cs.NewTokenCredentialsProvider(tok)
				}
				clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) { legacy++; return handler(r) })}))
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before", "size": 1073741824, "media": "disk", "clone_drive_id": "source", "tags": []interface{}{"tag"}})
				ds := resourceCloudSigmaDriveCreate(ctx, d, clients)
				if end != nil {
					end()
				}
				if !ds.HasError() || puts != 0 {
					t.Fatal(ds, puts)
				}
				if incomplete {
					if native+legacy != 0 || d.Id() != "" {
						t.Fatal(native, legacy, d.Id())
					}
					return
				}
				wn := 0
				if bits == 3 {
					wn = 2
					if !reviewBaseline {
						wn++
					}
				}
				if native != wn || legacy != 3-wn || posts != 1 || gets != 2 || d.Id() != "created" {
					t.Fatal(native, legacy, posts, gets, d.Id())
				}
				t.Logf("native=%d legacy=%d POST=%d GET=%d PUT=%d retained=%s", native, legacy, posts, gets, puts, d.Id())
			})
		}
	}
}
