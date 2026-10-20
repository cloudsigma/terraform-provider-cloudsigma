package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestServerPower(t *testing.T) {
	for _, action := range []string{"start", "stop"} {
		for _, code := range []int{200, 201, 202, 204, 205, 301, 302, 303, 307, 308, 401, 404, 500} {
			for _, body := range []string{`{"action":"start","result":"success","uuid":"ack-not-server"}`, `{}`, `null`, `{} trailing`, ``, `[]`, `{"uuid":7}`, `broken`} {
				t.Run(action+http.StatusText(code)+body, func(t *testing.T) {
					calls := 0
					rt := func(r *http.Request) (*http.Response, error) {
						calls++
						b, _ := io.ReadAll(r.Body)
						if string(b) != "{}" && string(b) != "{}\n" {
							t.Fatal(string(b))
						}
						if r.Method != "POST" || r.URL.RequestURI() != "/api/2.0/servers/fixture/action/?do="+action || r.Header.Get("Content-Type") != "application/json" {
							t.Fatal(r)
						}
						return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
					}
					c := reviewNativeNew(rt)
					var e error
					if action == "start" {
						e = c.StartServer(context.Background(), "fixture")
					} else {
						e = c.StopServer(context.Background(), "fixture")
					}
					old := cs.NewClient(cs.NewTokenCredentialsProvider("fixture"), cs.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
					var oe error
					if action == "start" {
						_, _, oe = old.Servers.Start(context.Background(), "fixture")
					} else {
						_, _, oe = old.Servers.Stop(context.Background(), "fixture")
					}
					want := code >= 200 && code < 300 && (body == `{}` || body == `null` || body == `{} trailing` || strings.Contains(body, `"result":"success"`))
					if (e == nil) != want || (oe == nil) != want || calls != 2 {
						t.Fatal(e, oe, calls, want)
					}
				})
			}
		}
	}
}
func TestServerPowerBoundaries(t *testing.T) {
	for _, start := range []bool{true, false} {
		invoke := func(c *Client, ctx context.Context, id string) error {
			if start {
				return c.StartServer(ctx, id)
			}
			return c.StopServer(ctx, id)
		}
		for _, id := range []string{"", "../x", "a/b", "a?b", "a#b", "a%2fb", "a\n"} {
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("sent invalid ID"); return nil, nil })
			if invoke(c, context.Background(), id) == nil {
				t.Fatal(id)
			}
		}
		for _, mode := range []string{"pre", "deadline", "late", "unknown", "read"} {
			t.Run(mode, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if mode == "pre" {
					cancel()
				}
				if mode == "deadline" {
					var done context.CancelFunc
					ctx, done = context.WithDeadline(ctx, time.Now().Add(-time.Second))
					defer done()
				}
				calls := 0
				c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
					calls++
					if mode == "late" {
						cancel()
					}
					if mode == "unknown" {
						return nil, errors.New("PRIVATE unknown commit")
					}
					var reader io.Reader = strings.NewReader(`{}`)
					if mode == "read" {
						reader = powerErrorReader{}
					}
					return &http.Response{StatusCode: 202, Body: io.NopCloser(reader)}, nil
				})
				e := invoke(c, ctx, "fixture")
				want := 1
				if mode == "pre" || mode == "deadline" {
					want = 0
				}
				if e == nil || calls != want || strings.Contains(e.Error(), "PRIVATE") {
					t.Fatal(e, calls)
				}
			})
		}
	}
}

type powerErrorReader struct{}

func (powerErrorReader) Read([]byte) (int, error) { return 0, errors.New("PRIVATE body failure") }
