package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestKeypairDelete(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 205, 301, 302, 307, 308, 400, 401, 403, 404, 409, 429, 500} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			calls := 0
			bodies := []*trackedBody{}
			rt := func(q *http.Request) (*http.Response, error) {
				calls++
				if q.Body != nil {
					payload, _ := io.ReadAll(q.Body)
					if len(payload) != 0 {
						t.Fatal("unexpected body")
					}
				}
				if q.Method != "DELETE" || q.URL.Path != "/api/2.0/keypairs/fixture/" || q.URL.RawQuery != "" {
					t.Fatal(q)
				}
				b := &trackedBody{Reader: strings.NewReader("PRIVATE")}
				bodies = append(bodies, b)
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: b, Request: q}, nil
			}
			c := reviewNativeNew(rt)
			err := c.DeleteKeypair(context.Background(), "fixture")
			if (err == nil) != (code == 204) {
				t.Fatal(code, err)
			}
			if err != nil {
				var e *Error
				if !errors.As(err, &e) || e.Status != code || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal(err)
				}
			}
			old := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
			_, oe := old.Keypairs.Delete(context.Background(), "fixture")
			if (oe == nil) != (code >= 200 && code < 300) || calls != 2 {
				t.Fatal(oe, calls)
			}
			for _, b := range bodies {
				if !b.closed {
					t.Fatal("unclosed")
				}
			}
		})
	}
}
func TestKeypairDeleteBoundaries(t *testing.T) {
	for _, id := range []string{"", "../x", "a/b", "a?b", "a#b", "a%2fb", "a\n"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("dispatch"); return nil, nil })
		if c.DeleteKeypair(context.Background(), id) == nil {
			t.Fatal(id)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("pre-cancel dispatch"); return nil, nil })
	if e := c.DeleteKeypair(ctx, "fixture"); e == nil || e.Error() != "cancelled" {
		t.Fatal(e)
	}
	for _, late := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		b := &trackedBody{Reader: strings.NewReader("")}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			calls++
			cancel()
			if late {
				return &http.Response{StatusCode: 204, Body: b}, nil
			}
			return nil, context.Canceled
		})
		if e := c.DeleteKeypair(ctx, "fixture"); e == nil || e.Error() != "cancelled" || calls != 1 {
			t.Fatal(e, calls)
		}
		if late && !b.closed {
			t.Fatal("unclosed")
		}
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if e := c.DeleteKeypair(ctx, "fixture"); e == nil || e.Error() != "timeout" {
		t.Fatal(e)
	}
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF })
	if e := c.DeleteKeypair(context.Background(), "fixture"); e == nil || e.Error() != "transport" {
		t.Fatal(e)
	}
}

// A delete has no response document: neither errors nor success read secrets.
type keypairDeleteUnreadable struct{ closed bool }

func (b *keypairDeleteUnreadable) Read([]byte) (int, error) {
	return 0, errors.New("SYNTHETIC-PRIVATE-BODY")
}
func (b *keypairDeleteUnreadable) Close() error { b.closed = true; return nil }

func TestKeypairDeleteFailures(t *testing.T) {
	for _, scenario := range []string{"success-body-error", "OTP401", "unknown-commit", "nil-response", "late-deadline"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			body := &keypairDeleteUnreadable{}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			c := reviewNativeNew(func(q *http.Request) (*http.Response, error) {
				calls++
				u, p, ok := q.BasicAuth()
				deadline, bounded := q.Context().Deadline()
				if !ok || u != "synthetic" || p != "synthetic" || q.Header.Get("Cookie") != "" || q.Header.Get("Accept-Encoding") != "identity" || !bounded || time.Until(deadline) > time.Second {
					t.Fatal("auth/deadline policy")
				}
				resp := &http.Response{StatusCode: 204, Header: make(http.Header), Body: body}
				switch scenario {
				case "OTP401":
					resp.StatusCode = 401
					resp.Header.Set("WWW-Authenticate", `OTP realm="SYNTHETIC-CHALLENGE"`)
				case "unknown-commit":
					return resp, errors.New("SYNTHETIC-PRIVATE-TRANSPORT")
				case "nil-response":
					return nil, nil
				case "late-deadline":
					<-q.Context().Done()
				}
				return resp, nil
			})
			err := c.DeleteKeypair(ctx, "fixture")
			if (err == nil) != (scenario == "success-body-error") || calls != 1 {
				t.Fatal(err, calls)
			}
			if scenario != "nil-response" && !body.closed {
				t.Fatal("body not closed")
			}
			if err != nil && strings.Contains(err.Error(), "SYNTHETIC") {
				t.Fatal("sensitive diagnostic")
			}
			if scenario == "late-deadline" && err.Error() != "timeout" {
				t.Fatal(err)
			}
		})
	}
}
