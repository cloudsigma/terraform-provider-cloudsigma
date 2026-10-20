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
func nativeServerDeleteFixture(t *testing.T, code int, body string, before func(), calls *atomic.Int32) *configClients {
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
			if !ok || u != "fixture" || p != "fixture" || r.Method != "DELETE" || r.Host != "zrh.cloudsigma.com" || r.URL.RequestURI() != "/api/2.0/servers/fixture/" || r.Header.Get("Accept-Encoding") != "identity" {
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
	clients.legacy = cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
		if r.Method == "GET" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"uuid":"fixture","status":"stopped"}`)), Request: r}, nil
		}
		t.Error("native fell back to legacy")
		return nil, fmt.Errorf("denied")
	})}))
	return clients
}

func TestOfflineLegacyNativeServerDelete(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 205, 301, 302, 307, 308, 400, 401, 403, 404, 409, 429, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls atomic.Int32
			clients := nativeServerDeleteFixture(t, code, "PRIVATE", nil, &calls)
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912})
			d.SetId("fixture")
			before := d.State()
			ds := resourceCloudSigmaServerDelete(context.Background(), d, clients)
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
				if r.Method == "GET" {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"uuid":"fixture","status":"stopped"}`)), Request: r}, nil
				}
				legacyCalls++
				if r.Method != "DELETE" || r.URL.RequestURI() != "/api/2.0/servers/fixture/" {
					t.Fatal(r)
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("PRIVATE")), Request: r}, nil
			})}))
			old := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912})
			old.SetId("fixture")
			prior := old.State()
			od := resourceCloudSigmaServerDelete(context.Background(), old, legacy)
			if legacyCalls != 1 || od.HasError() != (code < 200 || code >= 300) {
				t.Fatal(od, legacyCalls)
			}
			if od.HasError() && !reflect.DeepEqual(prior, old.State()) {
				t.Fatal("legacy failure state")
			}
		})
	}
}
func TestOfflineLegacyNativeServerDeleteCancellation(t *testing.T) {
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
			clients := nativeServerDeleteFixture(t, 204, "", beforeResponse, &calls)
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912})
			d.SetId("fixture")
			if scenario == "unsafe-id" {
				d.SetId("../bad")
				clients.legacy = cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"uuid":"../bad","status":"stopped"}`)), Request: r}, nil
				})}))
			}
			before := d.State()
			ds := resourceCloudSigmaServerDelete(ctx, d, clients)
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

// Exercise the actual unchanged stop/wait machinery, not a replacement waiter.
func TestOfflineLegacyNativeServerDeletePreparation(t *testing.T) {
	for _, scenario := range []string{"stopped", "running", "initial404", "initial500", "second404", "second500", "stop500", "poll500"} {
		t.Run(scenario, func(t *testing.T) {
			var deletes atomic.Int32
			clients := nativeServerDeleteFixture(t, 204, "", nil, &deletes)
			gets, stops := 0, 0
			clients.legacy = cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
				if deletes.Load() != 0 {
					t.Error("preparation after final mutation")
				}
				code := 200
				body := `{"uuid":"fixture","status":"stopped"}`
				switch r.Method {
				case "GET":
					gets++
					if gets == 1 && scenario == "initial404" || gets == 2 && scenario == "second404" {
						code = 404
					}
					if gets == 1 && scenario == "initial500" || gets == 2 && scenario == "second500" || gets == 3 && scenario == "poll500" {
						code = 500
					}
					if gets <= 2 && (scenario == "running" || scenario == "stop500" || scenario == "poll500") {
						body = `{"uuid":"fixture","status":"running"}`
					}
					if r.URL.RequestURI() != "/api/2.0/servers/fixture/" {
						t.Error("GET route")
					}
				case "POST":
					stops++
					if r.URL.RequestURI() != "/api/2.0/servers/fixture/action/?do=stop" {
						t.Errorf("stop route %s", r.URL.RequestURI())
					}
					body = `{"uuid":"fixture"}`
					if scenario == "stop500" {
						code = 500
					}
				default:
					t.Error("legacy DELETE fallback")
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})}))
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912})
			d.SetId("fixture")
			prior := d.State()
			ds := resourceCloudSigmaServerDelete(context.Background(), d, clients)
			failed := strings.HasSuffix(scenario, "500")
			if ds.HasError() != failed {
				t.Fatal(ds)
			}
			if failed {
				if !reflect.DeepEqual(prior, d.State()) || deletes.Load() != 0 {
					t.Fatal("failure mutation")
				}
			} else if d.Id() != "" {
				t.Fatal("not cleared")
			}
			want := int32(1)
			if failed || scenario == "initial404" {
				want = 0
			}
			if deletes.Load() != want {
				t.Fatal(deletes.Load())
			}
			if scenario == "running" && (gets != 3 || stops != 1) {
				t.Fatal(gets, stops)
			}
		})
	}
}
