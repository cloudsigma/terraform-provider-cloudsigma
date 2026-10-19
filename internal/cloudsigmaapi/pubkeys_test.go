package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestPubkeys(t *testing.T) {
	for _, body := range []string{`{"uuid":"response","name":"fixture","public_key":"ssh-fixture PUBLIC-ONLY","fingerprint":"fixture","has_private_key":true,"meta":{"n":9007199254740993},"unknown":true}`, `null`, `{}`, `{"meta":null}`, `{"meta":{}}`, `{"meta":[]}`, `{"has_private_key":"bad"}`, `{"public_key":{}}`, `{"uuid":[]}`, `{"name":null}`, `{"public_key":null}`, ``} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			rt := func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					u, p, ok := r.BasicAuth()
					if !ok || u != "synthetic" || p != "synthetic" || r.Header.Get("Cookie") != "" || r.Header.Get("Accept-Encoding") != "identity" {
						t.Fatal("native auth policy")
					}
				}
				if r.Method != "GET" || r.URL.Path != "/api/2.0/pubkeys/fixture/" || r.URL.RawQuery != "" {
					t.Fatal(r.URL)
				}
				if r.Body != nil {
					b, _ := io.ReadAll(r.Body)
					if len(b) != 0 {
						t.Fatal("body")
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			}
			c := reviewNativeNew(rt)
			old := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt)}))
			got, e := c.Pubkey(context.Background(), "fixture")
			want, _, oe := old.Pubkeys.Get(context.Background(), "fixture")
			if (e == nil) != (oe == nil) || calls != 2 {
				t.Fatal(e, oe, calls)
			}
			if e == nil {
				if strings.Contains(body, "9007199254740993") && got.Meta["n"] != float64(9007199254740992) {
					t.Fatal("legacy float64 characterization changed")
				}
				a, _ := json.Marshal(got)
				b, _ := json.Marshal(want)
				var av, bv map[string]interface{}
				_ = json.Unmarshal(a, &av)
				_ = json.Unmarshal(b, &bv)
				delete(bv, "private_key")
				if !reflect.DeepEqual(av, bv) {
					t.Fatalf("%s != %s", a, b)
				}
			}
		})
	}
}
func TestPubkeyHTTP(t *testing.T) {
	for _, code := range []int{201, 204, 301, 302, 303, 307, 308, 401, 403, 404, 500} {
		b := &trackedBody{Reader: strings.NewReader("PRIVATE")}
		calls := 0
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: code, Body: b}, nil
		})
		v, e := c.Pubkey(context.Background(), "fixture")
		var failure *Error
		if !errors.As(e, &failure) || failure.Status != code || v != nil || strings.Contains(e.Error(), "PRIVATE") || !b.closed || calls != 1 {
			t.Fatal(v, e, calls)
		}
	}
}
func TestPubkeyInvalidID(t *testing.T) {
	for _, id := range []string{"", "../x", "a/b", "a?b", "a#b", "a%2fb", "a\n"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("dispatch"); return nil, nil })
		if _, e := c.Pubkey(context.Background(), id); e == nil {
			t.Fatal(id)
		}
	}
}

func pubkeyReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Pubkey(context.Background(), "fixture")
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func pubkeyEarlierDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	want, _ := ctx.Deadline()
	c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
		got, _ := r.Context().Deadline()
		if !got.Equal(want) {
			t.Fatal("deadline extended")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	if _, e := c.Pubkey(ctx, "fixture"); e != nil {
		t.Fatal(e)
	}
}

func pubkeyLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Pubkey(ctx, "fixture")
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func pubkeyDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Pubkey(ctx, "fixture")
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestPubkeyBoundaries(t *testing.T) {
	t.Run("body-read", pubkeyReadError)
	t.Run("earlier-deadline", pubkeyEarlierDeadline)
	t.Run("late-cancel", pubkeyLateCancel)
	t.Run("deadline", pubkeyDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.Pubkey(context.Background(), "fixture"); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.Pubkey(context.Background(), "fixture"); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
