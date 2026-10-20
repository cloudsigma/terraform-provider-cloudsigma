package cloudsigma

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestOfflineServerMetadataAtomic(t *testing.T) {
	cases := []struct {
		name, metadata string
		invalid        bool
	}{
		{"missing", "", false}, {"null-container", `,"meta":null`, false}, {"empty", `,"meta":{}`, false},
		{"strings", `,"meta":{"unknown":"SECRET-value","empty":"","ssh_public_key":"ignored"}`, false},
		{"ssh-only", `,"meta":{"ssh_public_key":{"legacy":"ignored"}}`, false},
		{"number", `,"meta":{"SECRET-key":7}`, true}, {"boolean", `,"meta":{"SECRET-key":false}`, true},
		{"null-member", `,"meta":{"SECRET-key":null}`, true}, {"array", `,"meta":{"SECRET-key":["SECRET-value"]}`, true},
		{"nested", `,"meta":{"SECRET-key":{"nested":"SECRET-value"}}`, true},
		{"mixed", `,"meta":{"valid":"SECRET-value","invalid":7,"last":"SECRET-last"}`, true},
	}
	for _, path := range []string{"native", "token", "bare"} {
		for _, tc := range cases {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				body := `{"cpu":4000,"mem":1073741824,"smp":4,"name":"changed","vnc_password":"SECRET-password","resource_uri":"changed","pubkeys":[{"uuid":"new-key"}],"nics":[{"ip_v4_conf":{"conf":"static","ip":{"uuid":"192.0.2.1"}}}],"epcs":[{"size":32}],"tags":[{"uuid":"new-tag"}]` + tc.metadata + `}`
				var calls atomic.Int32
				var client interface{}
				if path == "native" {
					client = nativeServerReadFixture(t, 200, body, nil, &calls)
				} else {
					legacy := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
						calls.Add(1)
						return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
					})}))
					client = legacy
					if path == "token" {
						client = &configClients{legacy: legacy}
					}
				}
				initial := map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912, "smp": 1, "vnc_password": "prior", "meta": map[string]interface{}{"old": "retain"}, "ssh_keys": []interface{}{"old-key"}, "network": []interface{}{map[string]interface{}{"type": "dhcp"}}, "drive": []interface{}{map[string]interface{}{"uuid": "keep-drive", "device": "virtio"}}, "tags": []interface{}{"old-tag"}, "enclave_page_caches": []interface{}{1}}
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, initial)
				d.SetId("fixture")
				before := d.State()
				// Recovery makes baseline panic AND its partial-state consequence visible.
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("read panicked; prior-state-preserved=%t", reflect.DeepEqual(before, d.State()))
					}
				}()
				ds := resourceCloudSigmaServerRead(context.Background(), d, client)
				if calls.Load() != 1 {
					t.Fatal("dispatch count")
				}
				if tc.invalid {
					if len(ds) != 1 || ds[0].Summary != "invalid server metadata response: expected string values" || ds[0].Detail != "" || !ds.HasError() {
						t.Fatal("expected fixed sanitized metadata diagnostic")
					}
					if strings.Contains(fmt.Sprint(ds), "SECRET") {
						t.Fatal("diagnostic leak")
					}
					if !reflect.DeepEqual(before, d.State()) {
						t.Fatal("partial state projection")
					}
				} else {
					if ds.HasError() || d.Get("name") != "changed" {
						t.Fatal("valid read failed")
					}
					want := map[string]interface{}{"old": "retain"}
					if tc.name == "strings" {
						want = map[string]interface{}{"unknown": "SECRET-value", "empty": ""}
					}
					if !reflect.DeepEqual(d.Get("meta"), want) {
						t.Fatal("metadata compatibility")
					}
					if d.Id() != "fixture" || !reflect.DeepEqual(d.Get("drive"), initial["drive"]) {
						t.Fatal("unprojected state changed")
					}
				}
			})
		}
	}
}
