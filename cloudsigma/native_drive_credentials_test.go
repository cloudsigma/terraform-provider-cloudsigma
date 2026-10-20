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

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestOfflineLegacyNativeCredentialParity(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, token := range []string{"", "synthetic-token-marker"} {
		for _, user := range []string{"", "synthetic-user-marker"} {
			for _, password := range []string{"", "synthetic-password-marker"} {
				t.Run(fmt.Sprintf("token=%t,user=%t,password=%t", token != "", user != "", password != ""), func(t *testing.T) {
					incomplete := token == "" && (user == "" || password == "")
					var nativeDials atomic.Int32
					tr := &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
						nativeDials.Add(1)
						a, b := net.Pipe()
						go func() {
							defer b.Close()
							r, e := http.ReadRequest(bufio.NewReader(b))
							if e != nil {
								return
							}
							u, p, ok := r.BasicAuth()
							if !ok || u != user || p != password || r.URL.RequestURI() != "/api/2.0/drives/fixture/" {
								t.Error("native auth/route")
							}
							_, _ = fmt.Fprint(b, "HTTP/1.1 404 missing\r\nContent-Length: 2\r\nConnection: close\r\n\r\n[]")
						}()
						return a, nil
					}}
					http.DefaultTransport = tr
					c := &Config{Token: token, Username: user, Password: password, Location: "zrh"}
					c.loadAndValidate(context.Background(), "fixture")
					configured, e := c.clients()
					http.DefaultTransport = old
					if e != nil {
						t.Fatal("configuration must retain lazy credential validation", e)
					}
					if (configured.native != nil) != (token == "") {
						t.Fatal("token precedence")
					}
					var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(user, password)
					if token != "" {
						creds = cs.NewTokenCredentialsProvider(token)
					}
					var baselineDiag string
					for _, wrapped := range []bool{false, true} {
						calls := 0
						legacy := cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
							calls++
							if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
								t.Error("token auth")
							}
							return &http.Response{StatusCode: 404, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[]`)), Request: r}, nil
						})}))
						var meta interface{} = legacy
						if wrapped {
							configured.legacy = legacy
							meta = configured
						}
						d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "prior", "media": "disk", "size": 536870912, "clone_drive_id": "keep"})
						d.SetId("fixture")
						before := d.State()
						ds := resourceCloudSigmaDriveRead(context.Background(), d, meta)
						diagnostic := fmt.Sprint(ds)
						for _, secret := range []string{"synthetic-token-marker", "synthetic-user-marker", "synthetic-password-marker"} {
							if strings.Contains(diagnostic, secret) {
								t.Fatal("credential diagnostic leak")
							}
						}
						if incomplete {
							if !ds.HasError() || calls != 0 || nativeDials.Load() != 0 || !reflect.DeepEqual(before, d.State()) {
								t.Fatalf("incomplete credentials dispatched/mutated: legacy=%d native=%d diag=%v", calls, nativeDials.Load(), ds)
							}
							if !wrapped {
								baselineDiag = diagnostic
							} else if diagnostic != baselineDiag {
								t.Fatalf("SDK diagnostic mismatch: %q != %q", diagnostic, baselineDiag)
							}
						} else {
							if ds.HasError() || d.Id() != "" {
								t.Fatal("404 control")
							}
							if wrapped && token == "" {
								if calls != 0 || nativeDials.Load() != 1 {
									t.Fatal("complete Basic not exclusively native")
								}
							} else if calls != 1 {
								t.Fatal("legacy control count")
							}
						}
					}
				})
			}
		}
	}
}
