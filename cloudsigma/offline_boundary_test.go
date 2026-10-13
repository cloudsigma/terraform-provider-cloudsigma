package cloudsigma

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type fakeRT func(*http.Request) (*http.Response, error)

func (f fakeRT) RoundTrip(q *http.Request) (*http.Response, error) { return f(q) }
func init() {
	http.DefaultTransport = fakeRT(func(*http.Request) (*http.Response, error) { return nil, fmt.Errorf("offline network denied") })
}
func TestOfflineRefresh(t *testing.T) {
	for _, kind := range []string{"drive", "server"} {
		for _, status := range []int{200, 404, 500} {
			t.Run(fmt.Sprint(kind, status), func(t *testing.T) {
				calls := 0
				path := "/api/2.0/" + kind + "s/fixture/"
				c := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithLocation("next"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(q *http.Request) (*http.Response, error) {
					calls++
					if q.Method != "GET" || q.URL.RequestURI() != path || q.URL.Host != "next.cloudsigma.com" {
						t.Fatal("wire")
					}
					body := `{"uuid":"fixture","name":"fixture","status":"future-state","size":1024,"media":"disk"}`
					if status != 200 {
						body = `[]`
					}
					return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
				})}))
				r := resourceCloudSigmaDrive()
				read := resourceCloudSigmaDriveRead
				if kind == "server" {
					r = resourceCloudSigmaServer()
					read = resourceCloudSigmaServerRead
				}
				d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{})
				d.SetId("fixture")
				diags := read(context.Background(), d, c)
				if calls != 1 {
					t.Fatal("count")
				}
				if status == 404 && d.Id() != "" {
					t.Fatal("404 state")
				}
				if status == 500 && !diags.HasError() {
					t.Fatal("missing error")
				}
				if status == 200 && (diags.HasError() || d.Id() != "fixture") {
					t.Fatal("refresh")
				}
			})
		}
	}
}
func TestOfflineCloneEmptyGuard(t *testing.T) {
	calls := 0
	c := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithLocation("next"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(q *http.Request) (*http.Response, error) {
		calls++
		if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/drives/fixture/action/?do=clone" {
			t.Fatal("wire")
		}
		return &http.Response{StatusCode: 201, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[]}`)), Request: q}, nil
	})}))
	d, r, e := c.Drives.Clone(context.Background(), "fixture", nil)
	if calls != 1 || d != nil || r == nil || r.StatusCode != 201 || e == nil {
		t.Fatal("clone empty guard")
	}
}
