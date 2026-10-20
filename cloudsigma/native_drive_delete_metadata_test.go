package cloudsigma

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// One configured Basic client exercises both callbacks; no cloud socket is used.
func TestOfflineDriveDeleteMetadataCoexistence(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	var calls atomic.Int32
	http.DefaultTransport = &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			r, err := http.ReadRequest(bufio.NewReader(b))
			if err != nil {
				return
			}
			n := calls.Add(1)
			u, p, ok := r.BasicAuth()
			if !ok || u != "fixture" || p != "fixture" {
				t.Error("auth")
			}
			switch n {
			case 1:
				if r.Method != "DELETE" || r.URL.RequestURI() != "/api/2.0/drives/fixture/" {
					t.Error("delete route/order")
				}
				_, _ = fmt.Fprint(b, "HTTP/1.1 204 deleted\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			case 2:
				if r.Method != "GET" || r.URL.RequestURI() != "/api/2.0/servers/fixture/" {
					t.Error("server route/order")
				}
				body := `{"name":"changed","cpu":4000,"meta":{"SECRET-key":7}}`
				_, _ = fmt.Fprintf(b, "HTTP/1.1 200 ok\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
			default:
				t.Error("extra request")
			}
		}()
		return a, nil
	}}
	config := &Config{Username: "fixture", Password: "fixture", Location: "zrh"}
	config.loadAndValidate(context.Background(), "fixture")
	clients, err := config.clients()
	http.DefaultTransport = old
	if err != nil {
		t.Fatal(err)
	}
	drive := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "prior", "media": "disk", "size": 536870912})
	drive.SetId("fixture")
	if ds := resourceCloudSigmaDriveDelete(context.Background(), drive, clients); ds.HasError() || drive.Id() != "" {
		t.Fatal(ds)
	}
	server := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912, "smp": 1, "meta": map[string]interface{}{"old": "retain"}})
	server.SetId("fixture")
	before := server.State()
	ds := resourceCloudSigmaServerRead(context.Background(), server, clients)
	if len(ds) != 1 || !ds.HasError() || ds[0].Summary != "invalid server metadata response: expected string values" || ds[0].Detail != "" {
		t.Fatal("metadata hardening inactive", ds)
	}
	if !reflect.DeepEqual(before, server.State()) || calls.Load() != 2 {
		t.Fatal("state/count", calls.Load())
	}
}
