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

func TestNative26RichSuccess(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		for _, raw := range []bool{false, true} {
			for _, tag := range []bool{false, true} {
				for _, clone := range []bool{false, true} {
					t.Run(fmt.Sprintf("%d/raw%t/tag%t/clone%t", bits, raw, tag, clone), func(t *testing.T) {
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
						incomplete := tok == "" && (u == "" || p == "")
						conf := &Config{Username: u, Password: p, Token: tok, Location: "zrh"}
						conf.loadAndValidate(context.Background(), "fixture")
						clients, e := conf.clients()
						if e != nil {
							t.Fatal(e)
						}
						rich := `{"allow_multimount":true,"licenses":[{"amount":7,"license":{"burstable":true,"free":9007199254740993,"initial_purchase_requirement":1.2300,"subsequent_purchase_requirement":"42","long_name":"Long","name":"lic","resource_uri":"/licenses/lic/","type":"cpu","user_metric":"cores","unknown":true},"user":{"uuid":"user","resource_uri":"/users/user/"}},{"license":null,"user":null}],"unknown":{"discard":true},"media":"disk","meta":{"nested":{"keep":true},"n":9007199254740993,"fraction":1.234567890123456789,"null":null},"mounted_on":[{"uuid":"mount"}],"name":"after","owner":{"uuid":"owner"},"remote_snapshots":[{"uuid":"remote"}],"resource_uri":"/api/2.0/drives/created/","runtime":{"is_snapshotable":true,"snapshots_allocated_size":19,"storage_type":"nvme"},"size":1073741824,"snapshots":[{"uuid":"snap"}],"status":"unmounted","storage_type":"nvme","tags":[{"uuid":"old"}],"uuid":"created"}`
						oracle := cs.NewClient(cs.NewTokenCredentialsProvider("oracle"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
							return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(rich)), Request: r}, nil
						})}))
						model, _, e := oracle.Drives.Get(context.Background(), "created")
						if e != nil {
							t.Fatal(e)
						}
						model.UUID = ""
						model.Tags = []cs.Tag{{UUID: "tag"}}
						wantPUT, e := json.Marshal(&cs.DriveUpdateRequest{Drive: model})
						if e != nil {
							t.Fatal(e)
						}
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
							body := rich
							code := 200
							if r.Method == "POST" {
								code = 202
								if !clone {
									code = 201
								}
								body = `{"objects":[{"uuid":"created"}]}`
								b, _ := io.ReadAll(r.Body)
								wantPost := `{"media":"disk","name":"before","size":1073741824,"storage_type":"nvme"}`
								if !clone {
									wantPost = `{"objects":[` + wantPost + `]}`
								}
								if strings.TrimSpace(string(b)) != wantPost {
									t.Error("POST", string(b))
								}
							}
							if r.Method == "PUT" {
								b, _ := io.ReadAll(r.Body)
								if !bytes.Equal(b, wantPUT) {
									t.Fatalf("full PUT\ngot %s\nwant %s", b, wantPUT)
								}
								t.Logf("PUT %s", b)
							}
							return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
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
						data := map[string]interface{}{"name": "before", "size": 1073741824, "media": "disk", "storage_type": "nvme", "clone_drive_id": "source"}
						if !clone {
							delete(data, "clone_drive_id")
						}
						if tag {
							data["tags"] = []interface{}{"tag"}
						}
						d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, data)
						ds := resourceCloudSigmaDriveCreate(context.Background(), d, meta)
						if incomplete {
							if !ds.HasError() || len(calls) != 0 || d.Id() != "" {
								t.Fatal(ds, calls)
							}
							return
						}
						if ds.HasError() || d.Id() != "created" {
							t.Fatal(ds, calls)
						}
						want := []string{"POST /api/2.0/drives/source/action/?do=clone", "GET /api/2.0/drives/created/"}
						if !clone {
							want[0] = "POST /api/2.0/drives/"
						}
						if tag {
							want = append(want, "GET /api/2.0/drives/created/", "PUT /api/2.0/drives/created/")
						}
						want = append(want, "GET /api/2.0/drives/created/")
						if fmt.Sprint(want) != fmt.Sprint(calls) {
							t.Fatal(calls, want)
						}
						wn := 0
						if !raw && bits == 3 {
							wn = 3
							if !reviewBaseline && tag {
								wn += 2
							}
						}
						if native != wn || legacy != len(want)-wn {
							t.Fatal("dispatch", native, legacy, wn)
						}
						t.Logf("native=%d legacy=%d calls=%v", native, legacy, calls)
					})
				}
			}
		}
	}
}
