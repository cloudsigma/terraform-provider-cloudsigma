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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Actual constructor and HTTP client, in-memory wire only. No listening socket.
func nativeDeleteFixture(t *testing.T, code int, body string, before func(), calls *atomic.Int32) *configClients {
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
			if !ok || u != "fixture" || p != "fixture" || r.Method != "DELETE" || r.Host != "zrh.cloudsigma.com" || r.URL.RequestURI() != "/api/2.0/drives/fixture/" || r.Header.Get("Accept-Encoding") != "identity" {
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

func TestOfflineLegacyNativeDriveDelete(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 205, 301, 302, 307, 308, 400, 401, 403, 404, 409, 429, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls atomic.Int32
			clients := nativeDeleteFixture(t, code, "PRIVATE", nil, &calls)
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "prior", "size": 536870912, "media": "disk", "clone_drive_id": "keep"})
			d.SetId("fixture")
			before := d.State()
			ds := resourceCloudSigmaDriveDelete(context.Background(), d, clients)
			if calls.Load() != 1 || ds.HasError() != (code != 204) {
				t.Fatal(calls.Load(), ds)
			}
			if code == 204 {
				if d.Id() != "" {
					t.Fatal("not removed")
				}
			} else if !reflect.DeepEqual(before, d.State()) {
				t.Fatal("mutated")
			}
			if strings.Contains(fmt.Sprint(ds), "PRIVATE") {
				t.Fatal("leak")
			}
			legacyCalls := 0
			legacy := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
				legacyCalls++
				if r.Method != "DELETE" || r.URL.RequestURI() != "/api/2.0/drives/fixture/" {
					t.Fatal(r)
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("PRIVATE")), Request: r}, nil
			})}))
			old := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "prior", "size": 536870912, "media": "disk"})
			old.SetId("fixture")
			prior := old.State()
			od := resourceCloudSigmaDriveDelete(context.Background(), old, legacy)
			if legacyCalls != 1 || od.HasError() != (code < 200 || code >= 300) {
				t.Fatal(od, legacyCalls)
			}
			if od.HasError() && !reflect.DeepEqual(prior, old.State()) {
				t.Fatal("legacy failure state")
			}
		})
	}
}
func TestOfflineLegacyNativeDriveDeleteCancellation(t *testing.T) {
	for _, scenario := range []string{"pre-cancel", "pre-deadline", "late-cancel", "late-deadline", "unsafe-id"} {
		t.Run(scenario, func(t *testing.T) {
			var calls atomic.Int32
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			beforeResponse := func() {}
			switch scenario {
			case "pre-cancel":
				cancel()
			case "pre-deadline":
				time.Sleep(110 * time.Millisecond)
			case "late-cancel":
				beforeResponse = cancel
			case "late-deadline":
				beforeResponse = func() { <-ctx.Done() }
			}
			clients := nativeDeleteFixture(t, 204, "", beforeResponse, &calls)
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "prior", "size": 536870912, "media": "disk"})
			d.SetId("fixture")
			if scenario == "unsafe-id" {
				d.SetId("../bad")
			}
			before := d.State()
			ds := resourceCloudSigmaDriveDelete(ctx, d, clients)
			if !ds.HasError() || !reflect.DeepEqual(before, d.State()) {
				t.Fatal(ds)
			}
			want := int32(0)
			if strings.HasPrefix(scenario, "late-") {
				want = 1
			}
			if calls.Load() != want {
				t.Fatal(calls.Load())
			}
		})
	}
}
func TestOfflineLegacyNativeDriveDeleteMounted(t *testing.T) {
	for _, preparation := range []int{200, 404, 500} {
		t.Run(fmt.Sprint(preparation), func(t *testing.T) {
			var calls atomic.Int32
			clients := nativeDeleteFixture(t, 204, "", nil, &calls)
			servers := 0
			clients.legacy = cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
				servers++
				if r.Method != "GET" || r.URL.RequestURI() != "/api/2.0/servers/server/" || calls.Load() != 0 {
					t.Fatal("preparation order/route")
				}
				return &http.Response{StatusCode: preparation, Body: io.NopCloser(strings.NewReader(`{"uuid":"server","status":"stopped"}`)), Request: r}, nil
			})}))
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "prior", "size": 536870912, "media": "disk"})
			d.SetId("fixture")
			if err := d.Set("mounted_on", []interface{}{map[string]interface{}{"uuid": "server", "resource_uri": "/server/"}}); err != nil {
				t.Fatal(err)
			}
			before := d.State()
			ds := resourceCloudSigmaDriveDelete(context.Background(), d, clients)
			if servers != 1 || ds.HasError() != (preparation == 500) {
				t.Fatal(ds, servers)
			}
			if preparation == 500 {
				if calls.Load() != 0 || !reflect.DeepEqual(before, d.State()) {
					t.Fatal("preparation failure")
				}
			} else if calls.Load() != 1 || d.Id() != "" {
				t.Fatal("delete missing")
			}
		})
	}
}
