package cloudsigma

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"bufio"
	"net"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// nativeWaitBody blocks inside the actual response Read until the caller ends.
// The transport alone is overlaid; StateChangeConf timing is never replaced.
type nativeWaitBody struct{ ctx context.Context }

func (b *nativeWaitBody) Read(p []byte) (int, error) { <-b.ctx.Done(); return 0, b.ctx.Err() }
func (b *nativeWaitBody) Close() error               { return nil }

func TestNativeDriveWaiter(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		for _, outcome := range []string{"progress", "unknown", "failed", "malformed", "null", "limit", "cancel", "deadline"} {
			t.Run(fmt.Sprintf("%d/%s", bits, outcome), func(t *testing.T) {
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
				incomplete := tok == "" && (u == "" || p == "")
				calls := 0
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if outcome == "deadline" {
					var end context.CancelFunc
					ctx, end = context.WithTimeout(ctx, 5500*time.Millisecond)
					defer end()
				}
				handler := func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "GET" || r.URL.RequestURI() != "/api/2.0/drives/selected/" {
						t.Error("route", r.Method, r.URL)
					}
					if tok != "" {
						if r.Header.Get("Authorization") != "Bearer "+tok {
							t.Error("token precedence")
						}
					} else {
						a, b, ok := r.BasicAuth()
						if !ok || a != u || b != p {
							t.Error("basic")
						}
					}
					if outcome != "deadline" {
						if _, ok := r.Context().Deadline(); ok {
							t.Error("introduced read timeout")
						}
					}
					body := `{"uuid":"selected","status":"unmounted"}`
					switch outcome {
					case "progress":
						if calls == 1 {
							body = `{"uuid":"selected","status":"creating"}`
						} else if calls == 2 {
							body = `{"uuid":"selected","status":"cloning_dst"}`
						}
					case "unknown":
						body = `{"status":"future"}`
					case "failed":
						body = `{"status":"failed"}`
					case "malformed":
						body = `{`
					case "null":
						body = `null`
					case "limit":
						body = strings.Repeat(" ", 1048577)
					case "cancel", "deadline":
						if outcome == "cancel" {
							time.AfterFunc(50*time.Millisecond, cancel)
						}
						return &http.Response{StatusCode: 200, Header: make(http.Header), Body: &nativeWaitBody{ctx: ctx}, Request: r}, nil
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}
				clients := native23Clients(t, u, p, tok, handler)
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before", "media": "disk", "size": 1073741824})
				d.SetId("selected")
				before := d.State()
				waiter := &retry.StateChangeConf{Pending: []string{"cloning_dst", "creating"}, Target: []string{"mounted", "unmounted"}, Refresh: driveStatusRefreshFunc(ctx, clients, clients.legacy, d.Id()), Timeout: d.Timeout(schema.TimeoutCreate), Delay: 5 * time.Second, MinTimeout: 3 * time.Second}
				start := time.Now()
				value, err := waiter.WaitForStateContext(ctx)
				if !reflect.DeepEqual(before, d.State()) {
					t.Fatal("waiter mutated state")
				}
				want := 1
				if incomplete {
					want = 0
				}
				if outcome == "progress" && !incomplete {
					want = 3
				}
				if calls != want {
					t.Fatal("dispatch count", calls, want, err)
				}
				if (err == nil) != (outcome == "progress" && !incomplete) {
					t.Fatal("outcome", err, value)
				}
				if err == nil {
					if tok == "" {
						if _, ok := value.(*driveWaitStatus); !ok {
							t.Fatalf("native projection %T", value)
						}
					} else if _, ok := value.(*cs.Drive); !ok {
						t.Fatalf("legacy result %T", value)
					}
				}
				if time.Since(start) > 20*time.Second {
					t.Fatal("unbounded fixture")
				}
			})
		}
	}
}

func native23Clients(t *testing.T, u, p, tok string, handler func(*http.Request) (*http.Response, error)) *configClients {
	t.Helper()
	old := http.DefaultTransport
	http.DefaultTransport = &http.Transport{DialTLSContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			r, e := http.ReadRequest(bufio.NewReader(b))
			if e != nil {
				return
			}
			r = r.WithContext(ctx)
			res, e := handler(r)
			if e != nil {
				return
			}
			defer res.Body.Close()
			res.ProtoMajor = 1
			res.ProtoMinor = 1
			res.Close = true
			res.ContentLength = -1
			_ = res.Write(b)
		}()
		return a, nil
	}}
	conf := &Config{Username: u, Password: p, Token: tok, Location: "zrh"}
	conf.loadAndValidate(context.Background(), "fixture")
	clients, e := conf.clients()
	http.DefaultTransport = old
	if e != nil {
		t.Fatal(e)
	}
	var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
	if tok != "" {
		creds = cs.NewTokenCredentialsProvider(tok)
	}
	clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(handler)}))
	return clients
}
