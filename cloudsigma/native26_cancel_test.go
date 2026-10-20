package cloudsigma

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type reviewCancelBody struct {
	io.Reader
	cancel context.CancelFunc
}

func (b reviewCancelBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.cancel()
	return n, e
}
func (b reviewCancelBody) Close() error { return nil }
func TestNative26AckCancel(t *testing.T) {
	for _, phase := range []string{"before-body", "after-body", "deadline-before-body", "deadline-after-body"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if strings.HasPrefix(phase, "deadline") {
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()
			}
			trigger := func() {
				if strings.HasPrefix(phase, "deadline") {
					<-ctx.Done()
				} else {
					cancel()
				}
			}
			calls := 0
			handler := func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" {
					t.Fatal("unexpected retry/read", r.Method)
				}
				body := io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"committed"}]}`))
				if strings.HasSuffix(phase, "before-body") {
					trigger()
				} else {
					body = reviewCancelBody{strings.NewReader(`{"objects":[{"uuid":"committed"}]}`), trigger}
				}
				return &http.Response{StatusCode: 202, Header: make(http.Header), Body: body, Request: r}, nil
			}
			old := http.DefaultTransport
			defer func() { http.DefaultTransport = old }()
			http.DefaultTransport = &http.Transport{}
			conf := &Config{Username: "u", Password: "p", Location: "zrh"}
			conf.loadAndValidate(context.Background(), "fixture")
			clients, e := conf.clients()
			if e != nil {
				t.Fatal(e)
			}
			clients.native = nil
			clients.legacy = cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("u", "p"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(handler)}))
			d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "fixture", "size": 1073741824, "media": "disk", "clone_drive_id": "source"})
			ds := resourceCloudSigmaDriveCreate(ctx, d, clients)
			t.Logf("baseline=%t phase=%s calls=%d id=%q error=%t", reviewBaseline, phase, calls, d.Id(), ds.HasError())
			if calls != 1 || !ds.HasError() {
				t.Fatal(calls, ds)
			}
			want := "committed"
			if d.Id() != want {
				t.Fatalf("id=%q want=%q", d.Id(), want)
			}
		})
	}
}
