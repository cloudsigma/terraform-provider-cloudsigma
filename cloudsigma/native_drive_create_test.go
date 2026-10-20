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

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestOfflineNativeDriveCreate(t *testing.T) {
	for _, code := range []int{201, 401, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			posts, reads, waits := 0, 0, 0
			old := http.DefaultTransport
			http.DefaultTransport = &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
				a, b := net.Pipe()
				go func() {
					defer b.Close()
					r, e := http.ReadRequest(bufio.NewReader(b))
					if e != nil {
						return
					}
					status := 200
					body := `{"uuid":"created","name":"after","size":1073741824,"media":"disk","status":"unmounted"}`
					u, p, ok := r.BasicAuth()
					if !ok || u != "fixture" || p != "fixture" {
						t.Error("auth")
					}
					if r.Method == "POST" {
						posts++
						status = code
						body = `{"objects":[{"uuid":"created"}]}`
						if r.URL.RequestURI() != "/api/2.0/drives/" {
							t.Error("POST route")
						}
					} else {
						reads++
						if r.Method != "GET" || r.URL.RequestURI() != "/api/2.0/drives/created/" {
							t.Error("readback route")
						}
					}
					_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", status, len(body), body)
				}()
				return a, nil
			}}
			conf := &Config{Username: "fixture", Password: "fixture", Location: "zrh"}
			conf.loadAndValidate(context.Background(), "fixture")
			clients, e := conf.clients()
			http.DefaultTransport = old
			if e != nil {
				t.Fatal(e)
			}
			clients.legacy = cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
				waits++
				if r.Method != "GET" || r.URL.RequestURI() != "/api/2.0/drives/created/" {
					t.Error("legacy mutation/fallback")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"uuid":"created","status":"unmounted"}`)), Request: r}, nil
			})}))
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before", "size": 1073741824, "media": "disk"})
			before := d.State()
			ds := resourceCloudSigmaDriveCreate(context.Background(), d, clients)
			if posts != 1 || ds.HasError() != (code != 201) {
				t.Fatal(posts, ds)
			}
			if code == 201 {
				if waits != 1 || reads != 1 || d.Id() != "created" || d.Get("name") != "after" {
					t.Fatal(waits, reads, d.State())
				}
			} else if waits != 0 || reads != 0 || !reflect.DeepEqual(before, d.State()) {
				t.Fatal("failure mutated or followed up")
			}
		})
	}
	oldTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = oldTransport }()
	http.DefaultTransport = &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
		t.Error("incomplete Basic dispatched")
		return nil, fmt.Errorf("denied")
	}}
	for _, pair := range [][2]string{{"", ""}, {"fixture", ""}, {"", "fixture"}} {
		conf := &Config{Username: pair[0], Password: pair[1], Location: "zrh"}
		conf.loadAndValidate(context.Background(), "fixture")
		clients, e := conf.clients()
		if e != nil {
			t.Fatal(e)
		}
		d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before", "size": 1073741824, "media": "disk"})
		before := d.State()
		if ds := resourceCloudSigmaDriveCreate(context.Background(), d, clients); !ds.HasError() || !reflect.DeepEqual(before, d.State()) {
			t.Fatal(ds)
		}
	}
}
