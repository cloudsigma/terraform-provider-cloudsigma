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
	"sync/atomic"
	"testing"
	"time"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Actual constructor and HTTP client, in-memory wire only. No listening socket.
func nativeReadFixture(t *testing.T, code int, body string, before func(), calls *atomic.Int32) *configClients {
	t.Helper()
	old := http.DefaultTransport
	tr := &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			r, e := http.ReadRequest(bufio.NewReader(b))
			if e != nil {
				return
			}
			calls.Add(1)
			u, p, ok := r.BasicAuth()
			if !ok || u != "fixture" || p != "fixture" || r.Method != "GET" || r.Host != "zrh.cloudsigma.com" || r.URL.RequestURI() != "/api/2.0/drives/fixture/" || r.Header.Get("Accept-Encoding") != "identity" {
				t.Error("wire mismatch")
				return
			}
			if before != nil {
				before()
			}
			_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", code, len(body), body)
		}()
		return a, nil
	}}
	http.DefaultTransport = tr
	c := &Config{Username: "fixture", Password: "fixture", Location: "zrh"}
	c.loadAndValidate(context.Background(), "fixture")
	clients, e := c.clients()
	http.DefaultTransport = old
	if e != nil {
		t.Fatal(e)
	}
	clients.legacy = cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(*http.Request) (*http.Response, error) {
		t.Error("native fell back to legacy")
		return nil, fmt.Errorf("denied")
	})}))
	return clients
}

func TestOfflineLegacyNativeDriveRead(t *testing.T) {
	object := `{"uuid":"response-id","name":"image","media":"disk","resource_uri":"/drive/fixture/","size":9007199254740993,"status":"unmounted","storage_type":"ssd","mounted_on":[{"uuid":"server-id","resource_uri":"/server/id/"}],"tags":[{"uuid":"tag-id"}],"meta":{"nested":{"number":9007199254740993}},"licenses":[{"amount":9007199254740993}],"runtime":{"snapshots_allocated_size":9007199254740993}}`
	for _, tc := range []struct {
		name   string
		status int
		body   string
		bad    bool
	}{
		{"success", 200, object, false}, {"missing", 404, `PRIVATE`, false}, {"unauthorized", 401, `PRIVATE`, true}, {"forbidden", 403, `PRIVATE`, true}, {"server", 500, `PRIVATE`, true}, {"rate-limit", 429, `PRIVATE`, true}, {"redirect", 302, `PRIVATE`, true}, {"unexpected", 204, "", true}, {"decode", 200, `{"size":"wrong"}`, true}, {"null", 200, `null`, false}, {"empty", 200, `{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			clients := nativeReadFixture(t, tc.status, tc.body, nil, &calls)
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before", "size": 536870912, "media": "disk", "clone_drive_id": "source"})
			d.SetId("fixture")
			before := d.State()
			ds := resourceCloudSigmaDriveRead(context.Background(), d, clients)
			if calls.Load() != 1 || ds.HasError() != tc.bad {
				t.Fatalf("calls/diags: %d %#v", calls.Load(), ds)
			}
			if tc.bad {
				if !reflect.DeepEqual(before, d.State()) {
					t.Fatal("error mutated state")
				}
				if strings.Contains(fmt.Sprint(ds), "PRIVATE") {
					t.Fatal("body leak")
				}
				return
			}
			if tc.status == 404 {
				if d.Id() != "" {
					t.Fatal("404 did not remove ID")
				}
				return
			}
			if d.Id() != "fixture" || d.Get("clone_drive_id") != "source" {
				t.Fatal("identity/config changed")
			}
			if tc.name == "success" {
				want := map[string]interface{}{"media": "disk", "name": "image", "resource_uri": "/drive/fixture/", "size": int(9007199254740993), "status": "unmounted", "storage_type": "ssd", "uuid": "response-id", "mounted_on": []interface{}{map[string]interface{}{"uuid": "server-id", "resource_uri": "/server/id/"}}}
				for k, v := range want {
					if !reflect.DeepEqual(d.Get(k), v) {
						t.Fatalf("%s: %#v != %#v", k, d.Get(k), v)
					}
				}
				if !d.Get("tags").(*schema.Set).Contains("tag-id") {
					t.Fatal("tag projection")
				}
				// Pinned SDK singleton oracle, same actual SDKv2 callback and state projection.
				legacy := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					if r.URL.RequestURI() != "/api/2.0/drives/fixture/" {
						t.Fatal("SDK route")
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(object)), Request: r}, nil
				})}))
				ld := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"clone_drive_id": "source"})
				ld.SetId("fixture")
				if resourceCloudSigmaDriveRead(context.Background(), ld, legacy).HasError() || !reflect.DeepEqual(ld.State(), d.State()) {
					t.Fatal("SDK projection mismatch")
				}
			}
		})
	}
}

func TestOfflineLegacyNativeDriveCancellation(t *testing.T) {
	for _, kind := range []string{"cancel", "deadline", "late-cancel", "late-deadline", "invalid-id"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var before func()
			if kind == "cancel" {
				cancel()
			}
			if kind == "deadline" {
				var c context.CancelFunc
				ctx, c = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer c()
			}
			if kind == "late-cancel" {
				before = cancel
			}
			if kind == "late-deadline" {
				var c context.CancelFunc
				ctx, c = context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer c()
				before = func() { <-ctx.Done() }
			}
			var calls atomic.Int32
			clients := nativeReadFixture(t, 404, `PRIVATE`, before, &calls)
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before"})
			d.SetId("fixture")
			if kind == "invalid-id" {
				d.SetId("../invalid")
			}
			state := d.State()
			if !resourceCloudSigmaDriveRead(ctx, d, clients).HasError() || !reflect.DeepEqual(state, d.State()) {
				t.Fatal("cancel/invalid state")
			}
			if (kind == "cancel" || kind == "deadline" || kind == "invalid-id") && calls.Load() != 0 {
				t.Fatal("pre dispatch")
			}
		})
	}
}

func TestOfflineLegacyClientSeam(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = &http.Transport{}
	for _, tc := range []struct{ token, user, pass string }{{"", "", ""}, {"", "fixture", ""}, {"", "", "fixture"}, {"", "fixture", "fixture"}, {"synthetic", "", ""}, {"synthetic", "fixture", "fixture"}} {
		p := Provider()
		d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{"token": tc.token, "username": tc.user, "password": tc.pass, "location": "zrh", "base_url": "ignored.invalid"})
		m, ds := p.ConfigureContextFunc(context.Background(), d)
		if ds.HasError() {
			t.Fatal(ds)
		}
		c, ok := m.(*configClients)
		if !ok || c.legacy == nil || (c.native != nil) != (tc.token == "") {
			t.Fatal("auth precedence")
		}
		if got, e := extractLegacy(c); e != nil || got != c.legacy {
			t.Fatal("wrapper extraction")
		}
		if got, e := extractLegacy(c.legacy); e != nil || got != c.legacy {
			t.Fatal("raw extraction")
		}
	}
	// Native constructor failure cannot silently select SDK Basic.
	c := &Config{Location: "bad/location"}
	c.loadAndValidate(context.Background(), "fixture")
	if _, e := c.clients(); e == nil {
		t.Fatal("native construction fallback")
	}
	c.Token = "synthetic"
	if got, e := c.clients(); e != nil || got.native != nil {
		t.Fatal("token must not construct native")
	}
	callbacks := []func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics{resourceCloudSigmaDriveCreate, resourceCloudSigmaDriveRead, resourceCloudSigmaDriveUpdate, resourceCloudSigmaDriveDelete, resourceCloudSigmaServerCreate, resourceCloudSigmaServerRead, resourceCloudSigmaServerUpdate, resourceCloudSigmaServerDelete}
	for _, cb := range callbacks {
		for _, bad := range []interface{}{nil, "wrong", (*configClients)(nil), &configClients{}, (*cs.Client)(nil)} {
			if !cb(context.Background(), nil, bad).HasError() {
				t.Fatal("missing type guard")
			}
		}
	}
	// Server read and Drive token Read unwrap and dispatch unchanged SDK auth.
	for _, server := range []bool{false, true} {
		calls := 0
		legacy := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("Authorization") != "Bearer synthetic" {
				t.Errorf("token auth %q", r.Header.Get("Authorization"))
			}
			return &http.Response{StatusCode: 404, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: r}, nil
		})}))
		resource, read := resourceCloudSigmaDrive(), resourceCloudSigmaDriveRead
		if server {
			resource, read = resourceCloudSigmaServer(), resourceCloudSigmaServerRead
		}
		d := schema.TestResourceDataRaw(t, resource.Schema, nil)
		d.SetId("fixture")
		if read(context.Background(), d, &configClients{legacy: legacy}).HasError() || calls != 1 || d.Id() != "" {
			t.Fatal("wrapped legacy read")
		}
	}
}

// Every adapted callback accepts the wrapper and reaches the same legacy wire
// boundary as a raw SDK client. Failure before waiters prevents delays/writes.
func TestOfflineLegacyCallbackExtraction(t *testing.T) {
	for _, server := range []bool{false, true} {
		callbacks := []func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics{resourceCloudSigmaDriveCreate, resourceCloudSigmaDriveRead, resourceCloudSigmaDriveUpdate, resourceCloudSigmaDriveDelete}
		resource := resourceCloudSigmaDrive()
		if server {
			resource = resourceCloudSigmaServer()
			callbacks = []func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics{resourceCloudSigmaServerCreate, resourceCloudSigmaServerRead, resourceCloudSigmaServerUpdate, resourceCloudSigmaServerDelete}
		}
		for index, cb := range callbacks {
			t.Run(fmt.Sprintf("server-%t-callback-%d", server, index), func(t *testing.T) {
				var want string
				for _, wrapped := range []bool{false, true} {
					calls := 0
					wire := ""
					client := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
						calls++
						wire = r.Method + " " + r.URL.RequestURI()
						if r.Header.Get("Authorization") != "Bearer synthetic" {
							t.Error("auth")
						}
						return &http.Response{StatusCode: 403, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: r}, nil
					})}))
					d := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{"name": "fixture"})
					d.SetId("fixture")
					var meta interface{} = client
					if wrapped {
						meta = &configClients{legacy: client}
					}
					ds := cb(context.Background(), d, meta)
					if !ds.HasError() || calls != 1 || d.Id() != "fixture" {
						t.Fatalf("callback boundary: %d %#v", calls, ds)
					}
					if wrapped && wire != want {
						t.Fatalf("wrapper changed dispatch: %s != %s", wire, want)
					}
					want = wire
				}
			})
		}
	}
}
