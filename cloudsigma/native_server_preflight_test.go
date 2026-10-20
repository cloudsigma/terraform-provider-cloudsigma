package cloudsigma

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Actual Delete callback, actual ResourceData and pinned SDK baseline. The outer
// response is never a mutation payload; returned identity remains authoritative.
func TestOfflineServerDeletePreflight(t *testing.T) {
	for _, scenario := range []string{"stopped", "running", "initial404", "malformed", "mismatch", "null", "precision", "unknowncommit"} {
		for _, raw := range []bool{true, false} {
			t.Run(fmt.Sprint(scenario, raw), func(t *testing.T) {
				var order []string
				gets := 0
				wire := func(r *http.Request) (int, string) {
					order = append(order, r.Method+" "+r.URL.Path)
					if r.Method == "DELETE" {
						if scenario == "unknowncommit" {
							return 500, `{"private":"PRIVATE"}`
						}
						return 204, ""
					}
					if r.Method == "POST" {
						b, _ := io.ReadAll(r.Body)
						if strings.TrimSpace(string(b)) != "{}" {
							t.Error("stop payload", string(b))
						}
						return 202, `{"uuid":"ack"}`
					}
					gets++
					if gets == 1 {
						switch scenario {
						case "initial404":
							return 404, `{}`
						case "malformed":
							return 200, `{"uuid":`
						case "mismatch":
							return 200, `{"uuid":"other","status":"stopped"}`
						case "null":
							return 200, `null`
						case "precision":
							return 200, `{"uuid":"fixture","status":"stopped","mem":9007199254740993,"meta":{"private":"PRIVATE","number":9007199254740993},"jobs":[{"private":"PRIVATE"}]}`
						}
					}
					if scenario == "running" && gets == 2 {
						return 200, `{"uuid":"fixture","status":"running"}`
					}
					return 200, `{"uuid":"fixture","status":"stopped"}`
				}
				v := r1Clients(t, "", "u", "p", wire)
				v.legacy = cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("u", "p"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					if !raw {
						t.Error("native fallback")
					}
					code, b := wire(r)
					return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(b)), Request: r}, nil
				})}))
				var meta interface{} = v
				if raw {
					meta = v.legacy
				}
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912})
				d.SetId("fixture")
				before := d.State()
				ds := resourceCloudSigmaServerDelete(context.Background(), d, meta)
				fail := scenario == "malformed" || scenario == "unknowncommit" || scenario == "null"
				if ds.HasError() != fail {
					t.Fatal(ds, order)
				}
				if fail {
					if !reflect.DeepEqual(before, d.State()) {
						t.Fatal("state changed")
					}
				} else if d.Id() != "" {
					t.Fatal("ID retained")
				}
				if !raw && strings.Contains(fmt.Sprint(ds), "PRIVATE") {
					t.Fatal("private leak")
				}
				want := []string{"GET /api/2.0/servers/fixture/", "GET /api/2.0/servers/fixture/", "DELETE /api/2.0/servers/fixture/"}
				switch scenario {
				case "malformed", "initial404":
					want = want[:1]
				case "null":
					want = want[:1]
				case "mismatch":
					want[1] = "GET /api/2.0/servers/other/"
					want[2] = "DELETE /api/2.0/servers/other/"
				case "running":
					want = []string{want[0], want[1], "POST /api/2.0/servers/fixture/action/", want[1], want[2]}
				}
				if !reflect.DeepEqual(order, want) {
					t.Fatalf("order %v want %v", order, want)
				}
			})
		}
	}
}
