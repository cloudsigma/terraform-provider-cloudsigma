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

func TestIndependentCredentialPaths(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for bits := 0; bits < 8; bits++ {
		for _, clone := range []bool{false, true} {
			for _, tag := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/clone%t/tag%t", bits, clone, tag), func(t *testing.T) {
					u, p, tok := "", "", ""
					if bits&1 != 0 {
						u = "fixture-u"
					}
					if bits&2 != 0 {
						p = "fixture-p"
					}
					if bits&4 != 0 {
						tok = "fixture-token"
					}
					incomplete := tok == "" && (u == "" || p == "")
					calls := []string{}
					handler := func(r *http.Request) (*http.Response, error) {
						calls = append(calls, r.Method+" "+r.URL.RequestURI())
						if tok != "" {
							if r.Header.Get("Authorization") != "Bearer "+tok {
								t.Error("token")
							}
						} else {
							a, b, ok := r.BasicAuth()
							if !ok || a != u || b != p {
								t.Error("basic")
							}
						}
						body := `{"uuid":"created","name":"after","size":1073741824,"media":"disk","status":"unmounted","tags":[{"uuid":"tag"}]}`
						code := 200
						if r.Method == "POST" {
							code = 201
							if true {
								body = `{"objects":[{"uuid":"created"}]}`
							}
						}
						if r.Method == "PUT" {
							b, _ := io.ReadAll(r.Body)
							if !strings.Contains(string(b), `"uuid":"tag"`) {
								t.Error("tags payload", string(b))
							}
						}
						return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
					}
					http.DefaultTransport = &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
						a, b := net.Pipe()
						go func() {
							defer b.Close()
							r, e := http.ReadRequest(bufio.NewReader(b))
							if e != nil {
								return
							}
							res, _ := handler(r)
							body, _ := io.ReadAll(res.Body)
							fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", res.StatusCode, len(body), body)
						}()
						return a, nil
					}}
					conf := &Config{Username: u, Password: p, Token: tok, Location: "zrh"}
					conf.loadAndValidate(context.Background(), "fixture")
					clients, e := conf.clients()
					if e != nil {
						t.Fatal(e)
					}
					var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
					if tok != "" {
						creds = cs.NewTokenCredentialsProvider(tok)
					}
					clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(handler)}))
					raw := map[string]interface{}{"name": "before", "size": 1073741824, "media": "disk"}
					if clone {
						raw["clone_drive_id"] = "source"
					}
					if tag {
						raw["tags"] = []interface{}{"tag"}
					}
					d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, raw)
					before := d.State()
					ds := resourceCloudSigmaDriveCreate(context.Background(), d, clients)
					if incomplete {
						if !ds.HasError() || len(calls) != 0 || !reflect.DeepEqual(before, d.State()) {
							t.Fatal(ds, calls)
						}
						return
					}
					if ds.HasError() || d.Id() != "created" {
						t.Fatal(ds, calls, d.State())
					}
					want := 3
					if tag {
						want = 5
					}
					if len(calls) != want {
						t.Fatal(calls)
					}
					if tag && calls[3] != "PUT /api/2.0/drives/created/" {
						t.Fatal(calls)
					}
					t.Log(calls)
				})
			}
		}
	}
}

func TestNativeCreateR1Callback(t *testing.T) {
	for _, scenario := range []string{"missing", "null", "empty", "traversal", "encoded", "trailing-json", "trailing-text", "tag-get", "tag-put", "intermediate"} {
		t.Run(scenario, func(t *testing.T) {
			old := http.DefaultTransport
			defer func() { http.DefaultTransport = old }()
			calls := []string{}
			gets := 0
			handler := func(r *http.Request) (*http.Response, error) {
				calls = append(calls, r.Method+" "+r.URL.RequestURI())
				code := 200
				body := `{"uuid":"created","status":"unmounted"}`
				switch r.Method {
				case "POST":
					code = 201
					body = `{"objects":[{"uuid":"created"}]}`
					switch scenario {
					case "missing":
						body = `{"objects":[{}]}`
					case "null":
						body = `{"objects":[null]}`
					case "empty":
						body = `{"objects":[{"uuid":""}]}`
					case "traversal":
						body = `{"objects":[{"uuid":"../servers/victim"}]}`
					case "encoded":
						body = `{"objects":[{"uuid":"%2e%2e%2fservers"}]}`
					case "trailing-json":
						body += ` {}`
					case "trailing-text":
						body += ` PRIVATE`
					}
				case "GET":
					gets++
					if scenario == "tag-get" && gets == 2 {
						code = 500
					}
					if scenario == "intermediate" && gets <= 2 {
						status := "creating"
						if gets == 2 {
							status = "cloning_dst"
						}
						body = fmt.Sprintf(`{"uuid":"created","status":%q}`, status)
					}
				case "PUT":
					if scenario == "tag-put" {
						code = 500
					}
				}
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}
			http.DefaultTransport = &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
				a, b := net.Pipe()
				go func() {
					defer b.Close()
					r, e := http.ReadRequest(bufio.NewReader(b))
					if e != nil {
						return
					}
					res, _ := handler(r)
					body, _ := io.ReadAll(res.Body)
					fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", res.StatusCode, len(body), body)
				}()
				return a, nil
			}}
			conf := &Config{Username: "u", Password: "p", Location: "zrh"}
			conf.loadAndValidate(context.Background(), "fixture")
			clients, e := conf.clients()
			if e != nil {
				t.Fatal(e)
			}
			clients.legacy = cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("u", "p"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(handler)}))
			raw := map[string]interface{}{"name": "before", "size": 1073741824, "media": "disk", "tags": []interface{}{"tag"}}
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, raw)
			d.SetId("prior")
			before := d.State()
			ds := resourceCloudSigmaDriveCreate(context.Background(), d, clients)
			switch scenario {
			case "intermediate":
				if ds.HasError() || d.Id() != "created" || gets != 5 || len(calls) != 7 {
					t.Fatal(ds, calls)
				}
			case "tag-get", "tag-put":
				want := 3
				if scenario == "tag-put" {
					want = 4
				}
				if !ds.HasError() || d.Id() != "created" || len(calls) != want {
					t.Fatal(ds, calls)
				}
				d.SetId("prior")
				if !reflect.DeepEqual(before, d.State()) {
					t.Fatal("non-ID state changed")
				}
			default:
				if !ds.HasError() || len(calls) != 1 || !reflect.DeepEqual(before, d.State()) || strings.Contains(fmt.Sprint(ds), "PRIVATE") {
					t.Fatal(ds, calls, d.State())
				}
			}
		})
	}
}
