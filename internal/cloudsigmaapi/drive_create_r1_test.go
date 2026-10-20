package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

type reviewBody struct {
	io.Reader
	cancel context.CancelFunc
	closed bool
}

func (b *reviewBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	if b.cancel != nil {
		b.cancel()
	}
	return n, e
}
func (b *reviewBody) Close() error { b.closed = true; return nil }
func TestIndependentPOST(t *testing.T) {
	for _, tc := range []struct {
		name, body, encoding string
		code                 int
		want                 string
	}{{"empty", "{\"objects\":[]}", "", 201, "decode_error"}, {"malformed", "broken", "", 201, "decode_error"}, {"wrongtype", "{\"objects\":[{\"size\":\"3\"}]}", "", 201, "decode_error"}, {"oversize", strings.Repeat("x", responseLimit+2), "", 201, "response_too_large"}, {"encoding", "{}", "gzip", 201, "unsupported_encoding"}, {"status", "PRIVATE", "", 500, "http_status"}, {"redirect", "PRIVATE", "", 307, "http_status"}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			b := &reviewBody{Reader: strings.NewReader(tc.body)}
			c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Accept-Encoding") != "identity" {
					t.Fatal("encoding")
				}
				return &http.Response{StatusCode: tc.code, Header: http.Header{"Content-Encoding": []string{tc.encoding}}, Body: b}, nil
			})
			_, e := c.CreateDrive(context.Background(), DriveCreateRequest{})
			if e == nil || e.(*Error).Category != tc.want || calls != 1 || !b.closed || strings.Contains(fmt.Sprint(e), "PRIVATE") {
				t.Fatal(e, calls, b.closed)
			}
		})
	}
	t.Run("latecancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		b := &reviewBody{Reader: strings.NewReader(`{"objects":[{"uuid":"ok"}]}`), cancel: cancel}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 201, Body: b}, nil })
		v, e := c.CreateDrive(ctx, DriveCreateRequest{})
		if v != nil || e == nil || e.(*Error).Category != "cancelled" || !b.closed {
			t.Fatal(v, e)
		}
	})
	for _, body := range []string{`{"objects":[{}]}`, `{"objects":[null]}`, `{"objects":[{"uuid":"../servers/victim"}]}`, `{"objects":[{"uuid":"ok"}]} PRIVATE`} {
		t.Run("identity-"+body, func(t *testing.T) {
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			v, e := c.CreateDrive(context.Background(), DriveCreateRequest{})
			t.Logf("actual v=%+v error=%v", v, e)
			if e == nil {
				t.Errorf("FIX: malformed identity/document admitted")
			}
		})
	}
	t.Run("numeric-sdk", func(t *testing.T) {
		var bodies []string
		rt := func(r *http.Request) (*http.Response, error) {
			b, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(b))
			return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"ok","meta":{"x":{"nested":"v"}}}]}`))}, nil
		}
		c := reviewNativeNew(rt)
		a, _ := c.CreateDrive(context.Background(), DriveCreateRequest{Name: "n", Media: "disk", Size: 9007199254740993})
		sdk := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt)}))
		_, _, e := sdk.Drives.Create(context.Background(), &cs.DriveCreateRequest{Drives: []cs.Drive{{Name: "n", Media: "disk", Size: 9007199254740993}}})
		var x, y interface{}
		dx := json.NewDecoder(strings.NewReader(bodies[0]))
		dx.UseNumber()
		dy := json.NewDecoder(strings.NewReader(bodies[1]))
		dy.UseNumber()
		if err := dx.Decode(&x); err != nil {
			t.Fatal(err)
		}
		if err := dy.Decode(&y); err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(x) != fmt.Sprint(y) || e != nil || !strings.Contains(bodies[0], "9007199254740993") {
			t.Fatal(bodies, e)
		}
		a.Meta["x"].(map[string]interface{})["nested"] = "changed"
		b, _ := c.CreateDrive(context.Background(), DriveCreateRequest{})
		if b.Meta["x"].(map[string]interface{})["nested"] != "v" {
			t.Fatal("ownership")
		}
	})
}

type deadlinePOSTBody struct {
	ctx    context.Context
	closed bool
}

func (b *deadlinePOSTBody) Read([]byte) (int, error) { <-b.ctx.Done(); return 0, b.ctx.Err() }
func (b *deadlinePOSTBody) Close() error             { b.closed = true; return nil }
func TestDriveCreateR1Transport(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			c := reviewNativeNew(nil)
			c.transport = &http.Client{Transport: subscriptionOracleRT(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{"https://zrh.cloudsigma.com/api/2.0/drives/second/"}}, Body: io.NopCloser(strings.NewReader("PRIVATE")), Request: r}, nil
			}), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			_, e := c.CreateDrive(context.Background(), DriveCreateRequest{})
			if e == nil || calls != 1 {
				t.Fatal(e, calls)
			}
		})
	}
	t.Run("blocked-deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		b := &deadlinePOSTBody{ctx: ctx}
		calls := 0
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 201, Body: b}, nil
		})
		_, e := c.CreateDrive(ctx, DriveCreateRequest{})
		if e == nil || e.(*Error).Category != "timeout" || calls != 1 || !b.closed {
			t.Fatal(e, calls, b.closed)
		}
	})
	for _, failure := range []error{fmt.Errorf("PRIVATE"), context.Canceled, context.DeadlineExceeded} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			calls := 0
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) { calls++; return nil, failure })
			_, e := c.CreateDrive(context.Background(), DriveCreateRequest{})
			if e == nil || calls != 1 || strings.Contains(fmt.Sprint(e), "PRIVATE") {
				t.Fatal(e, calls)
			}
		})
	}
	for _, body := range []string{`{"objects":[{"uuid":null}]}`, `{"objects":[{"uuid":""}]}`, `{"objects":[{"uuid":"%2e%2e%2fservers"}]}`, `{"objects":[{"uuid":"ok?query"}]}`, `{"objects":[{"uuid":"ok#fragment"}]}`, `{"objects":[{"uuid":"ok"}]} {}`, `{"objects":[{"uuid":"ok"}]} junk`} {
		t.Run(body, func(t *testing.T) {
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			if _, e := c.CreateDrive(context.Background(), DriveCreateRequest{}); e == nil {
				t.Fatal("admitted")
			}
		})
	}
}
