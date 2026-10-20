package cloudsigma

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestNative26Dispatch(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		for _, fault := range []string{"401", "404", "500", "empty", "malformed", "disconnect", "cancel", "deadline"} {
			t.Run(fmt.Sprintf("%d/%s", bits, fault), func(t *testing.T) {
				old := http.DefaultTransport
				defer func() { http.DefaultTransport = old }()
				u, p, tok := "", "", ""
				if bits&1 != 0 {
					u = "fixture-u"
				}
				if bits&2 != 0 {
					p = "fixture-p"
				}
				if bits&4 != 0 {
					tok = "fixture-token"
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if fault == "deadline" {
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(context.Background(), 30*time.Millisecond)
					defer stop()
				}
				calls := 0
				nativeCalls, legacyCalls := 0, 0
				handler := func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "POST" || r.URL.RequestURI() != "/api/2.0/drives/source/action/?do=clone" {
						t.Error("unexpected request", r.Method, r.URL)
					}
					b, _ := io.ReadAll(r.Body)
					if strings.TrimSpace(string(b)) != `{"media":"disk","name":"before","size":1073741824}` {
						t.Error("payload", string(b))
					}
					if tok != "" {
						if r.Header.Get("Authorization") != "Bearer "+tok {
							t.Error("token precedence")
						}
					} else {
						a, b, ok := r.BasicAuth()
						if !ok || a != u || b != p {
							t.Error("Basic")
						}
					}
					if fault == "disconnect" {
						return nil, io.ErrUnexpectedEOF
					}
					if fault == "cancel" {
						cancel()
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					if fault == "deadline" {
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					code := 500
					body := `{"error":"fixture"}`
					switch fault {
					case "401":
						code = 401
					case "404":
						code = 404
					case "empty":
						code = 202
						body = `{"objects":[]}`
					case "malformed":
						code = 202
						body = `invalid`
					}
					return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}
				http.DefaultTransport = &http.Transport{DialTLSContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					a, b := net.Pipe()
					go func() {
						defer b.Close()
						r, e := http.ReadRequest(bufio.NewReader(b))
						if e != nil {
							return
						}
						r = r.WithContext(ctx)
						nativeCalls++
						res, e := handler(r)
						if e != nil {
							return
						}
						body, _ := io.ReadAll(res.Body)
						fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", res.StatusCode, len(body), body)
					}()
					return a, nil
				}}
				conf := &Config{Username: u, Password: p, Token: tok, Location: "zrh"}
				conf.loadAndValidate(context.Background(), "fixture")
				clients, e := conf.clients()
				if e != nil {
					t.Fatal(e)
				}
				var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
				if tok != "" {
					creds = cs.NewTokenCredentialsProvider(tok)
				}
				clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) { legacyCalls++; return handler(r) })}))
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before", "media": "disk", "size": 1073741824, "clone_drive_id": "source"})
				before := d.State()
				ds := resourceCloudSigmaDriveCreate(ctx, d, clients)
				want := 1
				if tok == "" && (u == "" || p == "") {
					want = 0
				}
				wantNative := 0
				if bits == 3 {
					wantNative = 1
				}
				if nativeCalls != wantNative || legacyCalls != want-wantNative {
					t.Fatalf("dispatch native=%d legacy=%d want native=%d total=%d", nativeCalls, legacyCalls, wantNative, want)
				}
				t.Logf("native=%d legacy=%d id=%q", nativeCalls, legacyCalls, d.Id())
				if !ds.HasError() || calls != want || !reflect.DeepEqual(before, d.State()) {
					t.Fatal(ds, calls, d.State())
				}
			})
		}
	}
}
