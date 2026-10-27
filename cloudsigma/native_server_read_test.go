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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Actual constructor and HTTP client, in-memory wire only. No listening socket.
func nativeServerReadFixture(t *testing.T, code int, body string, before func(), calls *atomic.Int32) *configClients {
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
			if !ok || u != "fixture" || p != "fixture" || r.Method != "GET" || r.Host != "zrh.cloudsigma.com" || r.URL.RequestURI() != "/api/2.0/servers/fixture/" || r.Header.Get("Accept-Encoding") != "identity" {
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

func TestOfflineLegacyNativeServerCancellation(t *testing.T) {
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
			clients := nativeServerReadFixture(t, 404, `PRIVATE`, before, &calls)
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "before"})
			d.SetId("fixture")
			if kind == "invalid-id" {
				d.SetId("../invalid")
			}
			state := d.State()
			if !resourceCloudSigmaServerRead(ctx, d, clients).HasError() || !reflect.DeepEqual(state, d.State()) {
				t.Fatal("cancel/invalid state")
			}
			if (kind == "cancel" || kind == "deadline" || kind == "invalid-id") && calls.Load() != 0 {
				t.Fatal("pre dispatch")
			}
		})
	}
}

func TestOfflineLegacyNativeServerRead(t *testing.T) {
	object := `{"uuid":"response-not-id","cpu":4000,"mem":9007199254740993,"name":"server","smp":2,"vnc_password":"fixture","resource_uri":"/servers/fixture/","status":"running","drives":[{"device":"ide","drive":{"uuid":"other"}}],"meta":{"ssh_public_key":"ignored","keep":"value"},"pubkeys":[{"uuid":"key"}],"epcs":[{"size":9007199254740993}],"tags":[{"uuid":"tag"}],"nics":[{"ip_v4_conf":{"conf":"static","ip":{"uuid":"192.0.2.1"}}},{"vlan":{"uuid":"vlan"}},{"ip_v4_conf":null}],"runtime":{"nics":[{"interface_type":"private","ip_v4":{"uuid":"wrong"}},{"interface_type":"public","ip_v4":{"uuid":"192.0.2.2"}},{"interface_type":"public","ip_v4":{"uuid":"wrong2"}}]}}`
	for _, body := range []string{object, `null`, `{}`, `{"meta":{},"pubkeys":[],"nics":[],"tags":[],"epcs":[]}`, `{"meta":{"ssh_public_key":"ignore"},"pubkeys":null,"nics":null,"runtime":null,"future":{"n":1}}`, `{"cpu":null,"mem":null,"meta":null,"epcs":null}`} {
		t.Run(body, func(t *testing.T) {
			initial := map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 1024, "vnc_password": "prior", "meta": map[string]interface{}{"old": "retain"}, "ssh_keys": []interface{}{"old-key"}, "network": []interface{}{map[string]interface{}{"type": "dhcp"}}, "drive": []interface{}{map[string]interface{}{"uuid": "keep-drive", "device": "virtio"}}, "tags": []interface{}{"old-tag"}, "enclave_page_caches": []interface{}{1}}
			var calls atomic.Int32
			native := nativeServerReadFixture(t, 200, body, nil, &calls)
			legacy := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
				if r.URL.RequestURI() != "/api/2.0/servers/fixture/" {
					t.Fatal("SDK route")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})}))
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, initial)
			d.SetId("fixture")
			old := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, initial)
			old.SetId("fixture")
			if resourceCloudSigmaServerRead(context.Background(), d, native).HasError() || resourceCloudSigmaServerRead(context.Background(), old, legacy).HasError() {
				t.Fatal("read failed")
			}
			if calls.Load() != 1 || !reflect.DeepEqual(d.State(), old.State()) {
				t.Fatalf("projection mismatch\n%#v\n%#v", d.State(), old.State())
			}
			if d.Id() != "fixture" {
				t.Fatal("identity/drive changed")
			}
			if body == object {
				drives := d.Get("drive").([]interface{})
				if len(drives) != 1 || drives[0].(map[string]interface{})["uuid"] != "other" || drives[0].(map[string]interface{})["device"] != "ide" {
					t.Fatal("attached drive was not refreshed")
				}
			} else {
				if d.Get("drive").([]interface{})[0].(map[string]interface{})["uuid"] != "keep-drive" {
					t.Fatal("partial response erased drive state")
				}
			}
			if body == object && (d.Get("memory") != int(9007199254740993) || d.Get("ipv4_address") != "192.0.2.2" || d.Get("meta").(map[string]interface{})["keep"] != "value" || !d.Get("ssh_keys").(*schema.Set).Contains("key")) {
				t.Fatal("explicit projection")
			}
		})
	}
	for _, tc := range []struct {
		code int
		body string
	}{{404, `PRIVATE`}, {401, `PRIVATE`}, {403, `PRIVATE`}, {429, `PRIVATE`}, {500, `PRIVATE`}, {302, `PRIVATE`}, {204, ``}, {200, `{"cpu":"wrong"}`}, {200, `[]`}} {
		var calls atomic.Int32
		c := nativeServerReadFixture(t, tc.code, tc.body, nil, &calls)
		d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "meta": map[string]interface{}{"old": "keep"}})
		d.SetId("fixture")
		before := d.State()
		ds := resourceCloudSigmaServerRead(context.Background(), d, c)
		if calls.Load() != 1 || ds.HasError() != (tc.code != 404) {
			t.Fatal(tc, ds)
		}
		if tc.code == 404 {
			if d.Id() != "" {
				t.Fatal("404")
			}
		} else if !reflect.DeepEqual(before, d.State()) || strings.Contains(fmt.Sprint(ds), "PRIVATE") {
			t.Fatal("state/privacy")
		}
	}
}

func TestOfflineServerDriveReplacementIsRefreshed(t *testing.T) {
	body := `{"uuid":"fixture","cpu":1000,"mem":536870912,"name":"server","smp":1,"vnc_password":"fixture","drives":[{"device":"virtio","drive":{"uuid":"new-drive"}}]}`
	for _, path := range []string{"native", "token", "bare"} {
		t.Run(path, func(t *testing.T) {
			var calls atomic.Int32
			var client interface{}
			if path == "native" {
				client = nativeServerReadFixture(t, 200, body, nil, &calls)
			} else {
				legacy := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.URL.RequestURI() != "/api/2.0/servers/fixture/" {
						t.Fatalf("unexpected server request: %s", r.URL)
					}
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				})}))
				client = legacy
				if path == "token" {
					client = &configClients{legacy: legacy}
				}
			}
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{
				"name": "server", "cpu": 1000, "memory": 536870912, "smp": 1, "vnc_password": "fixture",
				"drive": []interface{}{map[string]interface{}{"uuid": "old-drive", "device": "virtio"}},
			})
			d.SetId("fixture")
			if diags := resourceCloudSigmaServerRead(context.Background(), d, client); diags.HasError() {
				t.Fatal(diags)
			}
			if calls.Load() != 1 {
				t.Fatalf("server read calls: %d", calls.Load())
			}
			drives := d.Get("drive").([]interface{})
			if len(drives) != 1 || drives[0].(map[string]interface{})["uuid"] != "new-drive" {
				t.Fatalf("replacement drive not reflected in state: %#v", drives)
			}
		})
	}
}
