package cloudsigmaapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestDriveCallerContext(t *testing.T) {
	for _, code := range []int{200, 401, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			rt := func(r *http.Request) (*http.Response, error) {
				calls++
				if _, ok := r.Context().Deadline(); ok {
					t.Error("introduced deadline")
				}
				if r.Method != "GET" || r.URL.RequestURI() != "/api/2.0/drives/selected/" {
					t.Error("route")
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(`{"uuid":"selected","status":"running"}`)), Header: make(http.Header)}, nil
			}
			c := reviewNativeNew(rt)
			// Exercise actual http.Client timeout removal, not only a custom Do mock.
			h := &http.Client{Transport: subscriptionOracleRT(rt), Timeout: time.Nanosecond}
			c.transport = h
			got, e := c.DriveWithCallerContext(context.Background(), "selected")
			sdk := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("synthetic"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt)}))
			old, _, oe := sdk.Drives.Get(context.Background(), "selected")
			if (e == nil) != (oe == nil) || calls != 2 || h.Timeout != time.Nanosecond {
				t.Fatal(e, oe, calls, h.Timeout)
			}
			if e == nil && (got.UUID != old.UUID || got.Status != old.Status) {
				t.Fatal("projection")
			}
		})
	}
	for _, mode := range []string{"cancel", "deadline", "late-cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancel" {
				cancel()
			}
			if mode == "deadline" {
				var end context.CancelFunc
				ctx, end = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer end()
			}
			calls := 0
			c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
				calls++
				if mode == "late-cancel" {
					cancel()
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"status":"running"}`)), Header: make(http.Header)}, nil
			})

			_, err := c.DriveWithCallerContext(ctx, "selected")
			if err == nil {
				t.Fatal("cancellation lost")
			}
			if calls != 1 {
				t.Fatal(calls)
			}
		})
	}
}
