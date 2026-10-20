package cloudsigma

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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func r1Clients(t *testing.T, token, user, pass string, wire func(*http.Request) (int, string)) *configClients {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			r, e := http.ReadRequest(bufio.NewReader(b))
			if e != nil {
				return
			}
			code, body := wire(r)
			fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", code, len(body), body)
		}()
		return a, nil
	}}
	c := &Config{Token: token, Username: user, Password: pass, Location: "zrh"}
	c.loadAndValidate(context.Background(), "review")
	v, e := c.clients()
	if e != nil {
		t.Fatal(e)
	}
	return v
}

func TestR1Lifecycle(t *testing.T) {
	for _, start := range []bool{true, false} {
		for _, terminal := range []string{"target", "unknown", "failed"} {
			for _, raw := range []bool{true, false} {
				t.Run(fmt.Sprint(start, terminal, raw), func(t *testing.T) {
					initial, target, action := "running", "stopped", "stop"
					pending := []string{"running", "stopping", "unavailable"}
					if start {
						initial, target, action = "stopped", "running", "start"
						pending = []string{"stopped", "starting", "unavailable"}
					}
					statuses := append([]string{initial}, pending...)
					last := terminal
					if last == "target" {
						last = target
					}
					statuses = append(statuses, last)
					var order []string
					var times []time.Time
					gets := 0
					wire := func(r *http.Request) (int, string) {
						order = append(order, r.Method)
						if r.Method == "POST" {
							if r.URL.Query().Get("do") != action {
								t.Error("action")
							}
							return 202, "{}"
						}
						times = append(times, time.Now())
						status := statuses[gets]
						gets++
						return 200, fmt.Sprintf(`{"uuid":"fixture","status":%q}`, status)
					}
					v := r1Clients(t, "", "u", "p", wire)
					v.legacy = cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("u", "p"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
						if !raw {
							t.Error("fallback")
						}
						code, b := wire(r)
						return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(b)), Request: r}, nil
					})}))
					var meta interface{} = v
					if raw {
						meta = v.legacy
					}
					var err error
					if start {
						err = startServer(context.Background(), meta, "fixture")
					} else {
						err = stopServer(context.Background(), meta, "fixture")
					}
					if (err == nil) != (terminal == "target") || gets != 5 || !reflect.DeepEqual(order, []string{"GET", "POST", "GET", "GET", "GET", "GET"}) {
						t.Fatal(err, gets, order)
					}
					// Actual pinned waiter: initial 5s then min3s, exponential6s, cap10s.
					for i, want := range []time.Duration{5 * time.Second, 3 * time.Second, 6 * time.Second, 10 * time.Second} {
						elapsed := times[i+1].Sub(times[i])
						if elapsed < want-100*time.Millisecond || elapsed > want+4*time.Second {
							t.Fatal("timing", i, elapsed, want)
						}
					}
					t.Logf("ordered=%v timestamps=%v", order, times)
				})
			}
		}
	}
}

func TestR1Credentials(t *testing.T) {
	for _, start := range []bool{true, false} {
		for mask := 0; mask < 8; mask++ {
			for _, failure := range []string{"success", "action", "poll"} {
				for _, raw := range []bool{false, true} {
					t.Run(fmt.Sprint(start, mask, failure, raw), func(t *testing.T) {
						token, user, pass := "", "", ""
						if mask&1 != 0 {
							token = "token"
						}
						if mask&2 != 0 {
							user = "u"
						}
						if mask&4 != 0 {
							pass = "p"
						}
						gets, posts, native, legacy := 0, 0, 0, 0
						initial, target := "running", "stopped"
						if start {
							initial, target = "stopped", "running"
						}
						wire := func(r *http.Request) (int, string) {
							if token != "" {
								if r.Header.Get("Authorization") != "Bearer "+token {
									t.Error("token")
								}
							} else {
								u, p, ok := r.BasicAuth()
								if !ok || u != user || p != pass {
									t.Error("basic")
								}
							}
							if r.Method == "POST" {
								posts++
								if failure == "action" {
									return 500, "{}"
								}
								return 202, "{}"
							}
							gets++
							if gets > 1 && failure == "poll" {
								return 500, "{}"
							}
							status := initial
							if gets > 1 {
								status = target
							}
							return 200, fmt.Sprintf(`{"uuid":"fixture","status":%q}`, status)
						}
						v := r1Clients(t, token, user, pass, func(r *http.Request) (int, string) { native++; return wire(r) })
						var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(user, pass)
						if token != "" {
							creds = cs.NewTokenCredentialsProvider(token)
						}
						v.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
							legacy++
							code, b := wire(r)
							return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(b)), Request: r}, nil
						})}))
						var meta interface{} = v
						if raw {
							meta = v.legacy
						}
						var err error
						if start {
							err = startServer(context.Background(), meta, "fixture")
						} else {
							err = stopServer(context.Background(), meta, "fixture")
						}
						invalid := token == "" && (user == "" || pass == "")
						if invalid {
							if err == nil || native+legacy != 0 {
								t.Fatal(err, native, legacy)
							}
							return
						}
						wantGets := 2
						if failure == "action" {
							wantGets = 1
						}
						if gets != wantGets || posts != 1 || (err == nil) != (failure == "success") {
							t.Fatal(err, gets, posts)
						}
						if !raw && token == "" {
							if legacy != 0 {
								t.Fatal("fallback")
							}
						} else if native != 0 {
							t.Fatal("native token/raw")
						}
					})
				}
			}
		}
	}
}

func TestR1Callbacks(t *testing.T) {
	for _, update := range []bool{false, true} {
		var oracleBodies []interface{}
		var oracleOrder []string
		for _, raw := range []bool{true, false} {
			t.Run(fmt.Sprint(update, raw), func(t *testing.T) {
				status := "stopped"
				if update {
					status = "running"
				}
				var bodies []interface{}
				var order []string
				server := func() string {
					return fmt.Sprintf(`{"uuid":"fixture","status":%q,"name":"review","cpu":1000,"mem":536870912,"smp":1,"meta":{"keep":"value"},"drives":[],"nics":[]}`, status)
				}
				wire := func(r *http.Request) (int, string) {
					order = append(order, r.Method+" "+r.URL.RequestURI())
					body := server()
					action := r.URL.Query().Get("do")
					if action != "" {
						if action == "start" {
							status = "running"
						} else {
							status = "stopped"
						}
						return 202, "{}"
					}
					if r.Method != "GET" {
						var v interface{}
						decoder := json.NewDecoder(r.Body)
						decoder.UseNumber()
						if err := decoder.Decode(&v); err != nil {
							t.Error(err)
						}
						bodies = append(bodies, v)
						if r.Method == "POST" {
							body = `{"objects":[` + body + `]}`
						}
					}
					return 200, body
				}
				v := r1Clients(t, "", "u", "p", wire)
				v.legacy = cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("u", "p"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					if !raw && r.Method == "GET" {
						t.Error("unexpected legacy helper GET")
					}
					code, b := wire(r)
					return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(b)), Request: r}, nil
				})}))
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "review", "cpu": 1000, "memory": 536870912, "smp": 1, "meta": map[string]interface{}{"keep": "value"}, "drive": []interface{}{map[string]interface{}{"uuid": "disk", "device": "virtio"}}})
				var meta interface{} = v
				if raw {
					meta = v.legacy
				}
				if update {
					d.SetId("fixture")
					if ds := resourceCloudSigmaServerUpdate(context.Background(), d, meta); ds.HasError() {
						t.Fatal(ds)
					}
				} else {
					if ds := resourceCloudSigmaServerCreate(context.Background(), d, meta); ds.HasError() {
						t.Fatal(ds)
					}
				}
				if d.Id() != "fixture" || d.Get("name") != "review" || d.Get("memory") != 536870912 {
					t.Fatal(d.State())
				}
				if raw {
					oracleBodies = bodies
					oracleOrder = order
				} else if !reflect.DeepEqual(bodies, oracleBodies) || !reflect.DeepEqual(order, oracleOrder) {
					t.Fatal(bodies, oracleBodies, order, oracleOrder)
				}
				if len(bodies) != 1 && update || len(bodies) != 2 && !update {
					t.Fatal("missing mutation", bodies)
				}
				t.Logf("payloads=%#v order=%v", bodies, order)
			})
		}
	}
}

func TestR1Cancellation(t *testing.T) {
	for _, start := range []bool{true, false} {
		for _, boundary := range []string{"initial", "action", "poll"} {
			for _, deadline := range []bool{false, true} {
				for _, raw := range []bool{true, false} {
					t.Run(fmt.Sprint(start, boundary, deadline, raw), func(t *testing.T) {
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()
						if deadline {
							var end context.CancelFunc
							ctx, end = context.WithTimeout(ctx, 100*time.Millisecond)
							defer end()
						}
						gets, posts := 0, 0
						initial, target := "running", "stopped"
						if start {
							initial, target = "stopped", "running"
						}
						trigger := func() {
							if deadline {
								<-ctx.Done()
							} else {
								cancel()
							}
						}
						wire := func(r *http.Request) (int, string) {
							if r.Method == "POST" {
								posts++
								if raw && !start && r.Context().Done() != nil {
									t.Error("Stop action lost Background")
								}
								if boundary == "action" {
									trigger()
								}
								return 202, "{}"
							}
							gets++
							if gets == 1 && boundary == "initial" || gets > 1 && boundary == "poll" {
								trigger()
							}
							if raw && gets > 1 && ctx.Err() != nil {
								if r.Context().Err() == nil {
									t.Error("poll lost caller context")
								}
							}
							status := initial
							if gets > 1 {
								status = target
							}
							return 200, fmt.Sprintf(`{"uuid":"fixture","status":%q}`, status)
						}
						v := r1Clients(t, "", "u", "p", wire)
						v.legacy = cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("u", "p"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
							code, b := wire(r)
							return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(b)), Request: r}, nil
						})}))
						var meta interface{} = v
						if raw {
							meta = v.legacy
						}
						began := time.Now()
						var err error
						if start {
							err = startServer(ctx, meta, "fixture")
						} else {
							err = stopServer(ctx, meta, "fixture")
						}
						if time.Since(began) > 8*time.Second {
							t.Fatal("unbounded completion")
						}
						// SDK fake transport may accept a late successful body; native checks cancellation.
						if !raw && err == nil {
							t.Fatal("native admitted cancelled body")
						}
						if posts > 1 || gets > 2 {
							t.Fatal("retry", gets, posts)
						}
						if boundary == "initial" && !raw && posts != 0 {
							t.Fatal("action after cancelled initial read")
						}
						if boundary == "action" && posts != 1 {
							t.Fatal("missing action", posts)
						}
						t.Logf("error=%v GET=%d POST=%d elapsed=%s", err, gets, posts, time.Since(began))
					})
				}
			}
		}
	}
}
