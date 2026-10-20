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

func TestOfflineNativeServerPower(t *testing.T) {
	for _, start := range []bool{true, false} {
		for _, code := range []int{202, 401, 404, 500} {
			t.Run(fmt.Sprint(start, code), func(t *testing.T) {
				action, target, initial := "stop", "stopped", "running"
				if start {
					action, target, initial = "start", "running", "stopped"
				}
				old := http.DefaultTransport
				var posts atomic.Int32
				var nativeGets atomic.Int32
				tr := &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
					a, b := net.Pipe()
					go func() {
						defer b.Close()
						r, e := http.ReadRequest(bufio.NewReader(b))
						if e != nil {
							return
						}
						if r.Method == "GET" {
							n := nativeGets.Add(1)
							status := initial
							if n > 1 {
								status = target
							}
							body := fmt.Sprintf(`{"uuid":"fixture","status":%q}`, status)
							_, _ = fmt.Fprintf(b, "HTTP/1.1 200 response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
							return
						}
						posts.Add(1)
						body, _ := io.ReadAll(r.Body)
						u, p, ok := r.BasicAuth()
						if !ok || u != "fixture" || p != "fixture" || r.Method != "POST" || r.URL.RequestURI() != "/api/2.0/servers/fixture/action/?do="+action || string(body) != "{}" {
							t.Error("wire mismatch")
						}
						ack := `{"uuid":"ack-not-server","result":"success"}`
						_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", code, len(ack), ack)
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
				gets := 0
				clients.legacy = cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					if r.Method != "GET" {
						t.Fatal("fallback")
					}
					gets++
					status := initial
					if gets > 1 {
						status = target
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"uuid":"fixture","status":%q}`, status))), Request: r}, nil
				})}))
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaServer().Schema, map[string]interface{}{"name": "prior", "cpu": 1000, "memory": 536870912})
				d.SetId("fixture")
				before := d.State()
				if start {
					e = startServer(context.Background(), clients, d.Id())
				} else {
					e = stopServer(context.Background(), clients, d.Id())
				}
				if (e == nil) != (code == 202) || posts.Load() != 1 || !reflect.DeepEqual(before, d.State()) {
					t.Fatal(e, posts.Load())
				}
				if gets != 0 {
					t.Fatal("legacy GET fallback", gets)
				}
				if code == 202 && nativeGets.Load() != 2 {
					t.Fatal("ACK skipped waiter")
				}
				if code != 202 && nativeGets.Load() != 1 {
					t.Fatal("polled after failed action")
				}
			})
		}
	}
}

func TestOfflineNativeServerPowerCredentials(t *testing.T) {
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	for _, token := range []string{"", "token"} {
		for _, user := range []string{"", "user"} {
			for _, pass := range []string{"", "pass"} {
				var nativeCalls atomic.Int32
				http.DefaultTransport = &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
					a, b := net.Pipe()
					go func() {
						defer b.Close()
						r, e := http.ReadRequest(bufio.NewReader(b))
						if e != nil {
							return
						}
						nativeCalls.Add(1)
						u, p, ok := r.BasicAuth()
						if !ok || u != user || p != pass || r.Method != "GET" {
							t.Error("auth/method")
						}
						body := `{"uuid":"fixture","status":"running"}`
						_, _ = fmt.Fprintf(b, "HTTP/1.1 200 OK\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(body), body)
					}()
					return a, nil
				}}
				c := &Config{Token: token, Username: user, Password: pass, Location: "zrh"}
				c.loadAndValidate(context.Background(), "fixture")
				clients, e := c.clients()
				http.DefaultTransport = old
				if e != nil {
					t.Fatal(e)
				}
				if (clients.native != nil) != (token == "") {
					t.Fatal("precedence")
				}
				var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(user, pass)
				if token != "" {
					creds = cs.NewTokenCredentialsProvider(token)
				}
				calls := 0
				clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					calls++
					if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
						t.Fatal("auth")
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"uuid":"fixture","status":"running"}`)), Request: r}, nil
				})}))
				// Incomplete Basic refuses before either native or Legacy dispatch.
				e = startServer(context.Background(), clients, "fixture")
				incomplete := token == "" && (user == "" || pass == "")
				if (e != nil) != incomplete || (incomplete && calls != 0) || (!incomplete && calls+int(nativeCalls.Load()) != 1) {
					t.Fatal(e, calls)
				}
			}
		}
	}
}
