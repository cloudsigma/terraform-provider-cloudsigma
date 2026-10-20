package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

const native27Baseline = false
const native27Rich = `{"uuid":"created","status":"running","cpu":2000,"mem":536870912,"smp":2,"name":"created","meta":{"n":9007199254740993,"nested":{"x":null}},"drives":[{"boot_order":2,"dev_channel":"0:2","device":"virtio","drive":{"uuid":"existing","size":9007199254740993,"licenses":[{"amount":7,"license":{"free":9007199254740993,"initial_purchase_requirement":1.2300,"user_metric":"cores"}}],"meta":{"n":9007199254740993}}}],"nics":[{"model":"virtio","mac":"aa:bb","ip_v4_conf":{"conf":"static","ip":{"uuid":"192.0.2.1"}},"vlan":{"uuid":"vlan"},"firewall_policy":{"uuid":"fw","rules":[{"action":"accept","dst_port":"443"}]}}],"pubkeys":[{"uuid":"key","public_key":"synthetic","fingerprint":"fp","has_private_key":true}],"tags":[{"uuid":"tag"}],"unknown":true}`

func TestNative27Callback(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		for _, raw := range []bool{false, true} {
			for _, attach := range []bool{false, true} {
				for _, fault := range []string{"success", "post500", "disconnect", "put500"} {
					t.Run(fmt.Sprintf("%d/raw%t/attach%t/%s", bits, raw, attach, fault), func(t *testing.T) {
						old := http.DefaultTransport
						defer func() { http.DefaultTransport = old }()
						http.DefaultTransport = &http.Transport{}
						u, p, tok := "", "", ""
						if bits&1 != 0 {
							u = "u"
						}
						if bits&2 != 0 {
							p = "p"
						}
						if bits&4 != 0 {
							tok = "t"
						}
						cfg := &Config{Username: u, Password: p, Token: tok, Location: "zrh"}
						cfg.loadAndValidate(context.Background(), "fixture")
						clients, e := cfg.clients()
						if e != nil {
							t.Fatal(e)
						}
						oracle := cs.NewClient(cs.NewTokenCredentialsProvider("oracle"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
							return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"objects":[` + native27Rich + `]}`)), Header: make(http.Header), Request: r}, nil
						})}))
						models, _, e := oracle.Servers.Create(context.Background(), &cs.ServerCreateRequest{})
						if e != nil {
							t.Fatal(e)
						}
						model := models[0]
						model.UUID = ""
						model.Drives = append(model.Drives, cs.ServerDrive{BootOrder: 2, DevChannel: "0:2", Device: "virtio", Drive: &cs.Drive{UUID: "added"}})
						wantPUT, _ := json.Marshal(&cs.ServerUpdateRequest{Server: &model})
						wantPUT = append(wantPUT, '\n')
						calls := []string{}
						native, legacy := 0, 0
						handler := func(r *http.Request) (*http.Response, error) {
							calls = append(calls, r.Method+" "+r.URL.RequestURI())
							if tok != "" {
								if r.Header.Get("Authorization") != "Bearer "+tok {
									t.Error("token")
								}
							} else {
								a, b, _ := r.BasicAuth()
								if a != u || b != p {
									t.Error("basic")
								}
							}
							code, body := 200, `{"uuid":"created","status":"running","cpu":2000,"mem":536870912,"smp":2,"name":"created"}`
							if r.Method == "POST" {
								code = 201
								body = `{"objects":[` + native27Rich + `]}`
								if fault == "post500" {
									code = 500
								}
								if fault == "disconnect" {
									return nil, fmt.Errorf("synthetic disconnect")
								}
							}
							if r.Method == "PUT" {
								b, _ := io.ReadAll(r.Body)
								if !bytes.Equal(b, wantPUT) {
									t.Errorf("PUT mismatch\n%s\n%s", b, wantPUT)
								}
								if fault == "put500" {
									code = 500
								}
							}
							return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
						}
						if clients.native != nil {
							clients.native = native26Transport(t, func(r *http.Request) (*http.Response, error) { native++; return handler(r) })
						}
						var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
						if tok != "" {
							creds = cs.NewTokenCredentialsProvider(tok)
						}
						clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) { legacy++; return handler(r) })}))
						var meta interface{} = clients
						if raw {
							meta = clients.legacy
						}
						data := map[string]interface{}{"name": "before", "cpu": 2000, "memory": 536870912, "smp": 2}
						if attach {
							data["drive"] = []interface{}{map[string]interface{}{"uuid": "added", "device": "virtio"}}
						}
						d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, data)
						ds := resourceCloudSigmaServerCreate(context.Background(), d, meta)
						incomplete := tok == "" && (u == "" || p == "")
						failed := fault == "post500" || fault == "disconnect" || (attach && fault == "put500")
						if incomplete {
							if !ds.HasError() || len(calls) != 0 || d.Id() != "" {
								t.Fatal(ds, calls, d.Id())
							}
							return
						}
						if ds.HasError() != failed {
							t.Fatal(ds, calls)
						}
						id := "created"
						if fault == "post500" || fault == "disconnect" {
							id = ""
						}
						if d.Id() != id {
							t.Fatal(d.Id(), id)
						}
						want := []string{"POST /api/2.0/servers/"}
						if id != "" {
							if attach {
								want = append(want, "PUT /api/2.0/servers/created/")
							}
							if !failed {
								want = append(want, "GET /api/2.0/servers/created/", "GET /api/2.0/servers/created/")
							}
						}
						if fmt.Sprint(calls) != fmt.Sprint(want) {
							t.Fatal(calls, want)
						}
						wn := 0
						if !raw && bits == 3 {
							if !native27Baseline {
								wn++
							}
							if !failed {
								wn += 2
							}
						}
						if native != wn || legacy != len(want)-wn {
							t.Fatal("dispatch", native, legacy, wn)
						}
					})
				}
			}
		}
	}
}
