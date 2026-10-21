package cloudsigmaapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type serverAttachmentErrorReader struct {
	cancel context.CancelFunc
	fail   bool
}

func (r serverAttachmentErrorReader) Read(p []byte) (int, error) {
	n := copy(p, `{"uuid":"other"}`)
	if r.cancel != nil {
		r.cancel()
	}
	if r.fail {
		return n, io.ErrUnexpectedEOF
	}
	return n, io.EOF
}
func (serverAttachmentErrorReader) Close() error { return nil }

func TestNative32Transport(t *testing.T) {
	for _, mode := range []string{"ok", "different-id", "empty-object", "401", "404", "500", "201", "204", "empty", "malformed", "trailing", "null", "array", "encoding", "oversize", "disconnect", "error-valid-bytes", "cancel-valid-ack", "pre-cancel", "pre-deadline"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "pre-cancel" {
				cancel()
			}
			if mode == "pre-deadline" {
				var end context.CancelFunc
				ctx, end = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer end()
			}
			calls := 0
			c := reviewNativeNew(nil)
			h := &http.Client{Timeout: time.Nanosecond, Transport: subscriptionOracleRT(func(r *http.Request) (*http.Response, error) {
				calls++
				if _, ok := r.Context().Deadline(); ok {
					t.Error("introduced deadline")
				}
				if r.Method != "PUT" || r.URL.RequestURI() != "/api/2.0/servers/selected/" {
					t.Error("route")
				}
				b, _ := io.ReadAll(r.Body)
				if string(b) != "{\"tags\":[]}\n" {
					t.Error("bytes")
				}
				body := `{"uuid":"selected"}`
				code := 200
				header := make(http.Header)
				switch mode {
				case "different-id":
					body = `{"uuid":"other"}`
				case "empty-object":
					body = `{}`
				case "401":
					code = 401
				case "404":
					code = 404
				case "500":
					code = 500
				case "201":
					code = 201
				case "204":
					code = 204
				case "empty":
					body = ""
				case "malformed":
					body = "{"
				case "trailing":
					body = "{} {}"
				case "null":
					body = "null"
				case "array":
					body = "[]"
				case "encoding":
					header.Set("Content-Encoding", "gzip")
				case "oversize":
					body = strings.Repeat("x", responseLimit+1)
				case "disconnect":
					return nil, fmt.Errorf("unknown commit")
				}
				var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
				if mode == "error-valid-bytes" {
					reader = serverAttachmentErrorReader{fail: true}
				}
				if mode == "cancel-valid-ack" {
					reader = serverAttachmentErrorReader{cancel: cancel}
				}
				return &http.Response{StatusCode: code, Header: header, Body: reader}, nil
			})}
			c.transport = h
			err := c.PutServerAttachment(ctx, "selected", []byte("{\"tags\":[]}\n"))
			success := mode == "ok" || mode == "different-id" || mode == "empty-object"
			if (err == nil) != success {
				t.Fatal(err)
			}
			want := 1
			if strings.HasPrefix(mode, "pre-") {
				want = 0
			}
			if calls != want || h.Timeout != time.Nanosecond {
				t.Fatal(calls, h.Timeout)
			}
		})
	}
}
