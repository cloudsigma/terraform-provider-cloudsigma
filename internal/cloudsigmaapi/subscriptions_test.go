package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestSubscriptions(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"objects":[{"uuid":"selected","id":"14","amount":"9007199254740993","price":"0.000003839842975139617919921875","remaining":"0E-20","auto_renew":true,"free_tier":false,"subscribed_object":null,"unknown":{}}],"meta":{"limit":20,"offset":0,"total_count":99,"next":"https://untrusted.invalid"}}`, true},
		{`{}`, true},
		{`null`, true},
		{`{"objects":null}`, true},
		{`{"objects":[null,{}]}`, true},
		{`{"objects":[{"amount":null,"auto_renew":null}]}`, true},
		{`{"objects":[],"objects":[{"uuid":"last"}]} {}`, true},
		{`{"objects":[{"amount":"1","amount":null,"price":"-0","remaining":"1e999"}]}`, true},
		{`{"objects":[{"amount":null,"unknown":{"deep":123}}]}`, true},
		{`{"objects":[{"id":null,"unknown":{"deep":123}}]}`, true},
		{`{"objects":[{"price":null,"unknown":{"deep":123}}]}`, true},
		{`{"objects":[{"remaining":null,"unknown":{"deep":123}}]}`, true},
		{`{"objects":[{"uuid":null,"unknown":{"deep":123}}]}`, true},
		{`{"objects":[{"auto_renew":null,"unknown":{"deep":123}}]}`, true},
		{`{"objects":[{"free_tier":null,"unknown":{"deep":123}}]}`, true},
		{`[]`, false},
		{`{"objects":{}}`, false},
		{`{"objects":[{"amount":9007199254740993}]}`, false},
		{`{"objects":[{"price":1.23}]}`, false},
		{`{"objects":[{"id":14}]}`, false},
		{`{"objects":[{"remaining":2}]}`, false},
		{`{"meta":{"limit":"bad"}}`, false},
		{``, false},
		{`{"objects":[{"amount":true,"amount":"ok"}]}`, false},
		{`{"objects":[{"subscribed_object":{}}]}`, false},
		{`{"objects":[{"auto_renew":"true"}]}`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			calls := 0
			b := &trackedBody{Reader: strings.NewReader(tc.body)}
			c, _ := New("zrh", "fixture-user", "fixture-pass", "agent")
			c.transport = fakeDoer(func(r *http.Request) (*http.Response, error) {
				calls++
				u, p, ok := r.BasicAuth()
				if !ok || u != "fixture-user" || p != "fixture-pass" || r.URL.String() != "https://zrh.cloudsigma.com/api/2.0/subscriptions/" || r.URL.RawQuery != "" || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("User-Agent") != "agent" {
					t.Fatal("request")
				}
				return &http.Response{StatusCode: 200, Body: b}, nil
			})
			got, err := c.Subscriptions(context.Background())
			if (err == nil) != tc.valid || calls != 1 || !b.closed {
				t.Fatalf("%v calls%d", err, calls)
			}
			legacyClient := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}))
			legacy, _, legacyErr := legacyClient.Subscriptions.List(context.Background())
			if (legacyErr == nil) != tc.valid || (tc.valid && !subscriptionEqual(got, legacy)) {
				t.Fatalf("pinned service oracle: %v got=%+v legacy=%+v", legacyErr, got, legacy)
			}

		})
	}
	for _, code := range []int{201, 204, 301, 404, 500} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("PRIVATE"))}, nil
		})
		if _, err := c.Subscriptions(context.Background()); err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal(err)
		}
	}
	for _, cancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if cancelled {
			cancel()
		}
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			if cancelled {
				return nil, r.Context().Err()
			}
			return nil, errors.New("PRIVATE")
		})
		_, err := c.Subscriptions(ctx)
		cancel()
		if err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal(err)
		}
	}
}

// The pinned SDK model is an independent compatibility oracle, test-only.
func subscriptionEqual(got []Subscription, legacy []cloudsigma.Subscription) bool {
	if len(got) != len(legacy) {
		return false
	}
	for i := range got {
		if !reflect.DeepEqual(got[i], Subscription(legacy[i])) {
			return false
		}
	}
	return (got == nil) == (legacy == nil)
}

type subscriptionOracleRT func(*http.Request) (*http.Response, error)

func (f subscriptionOracleRT) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func subscriptionReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Subscriptions(context.Background())
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func subscriptionEarlierDeadline(t *testing.T) {
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
	if _, e := c.Subscriptions(ctx); e != nil {
		t.Fatal(e)
	}
}

func subscriptionLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Subscriptions(ctx)
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func subscriptionDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Subscriptions(ctx)
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestSubscriptionsBoundaries(t *testing.T) {
	t.Run("body-read", subscriptionReadError)
	t.Run("earlier-deadline", subscriptionEarlierDeadline)
	t.Run("late-cancel", subscriptionLateCancel)
	t.Run("deadline", subscriptionDeadline)
	got, err := decodeSubscriptions([]byte(`{"objects":[{"amount":"9007199254740993","price":"0.000003839842975139617919921875","remaining":"0E-20"}]}`))
	if err != nil || got[0].Amount != "9007199254740993" || got[0].Price != "0.000003839842975139617919921875" || got[0].Remaining != "0E-20" {
		t.Fatal(got, err)
	}
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.Subscriptions(context.Background()); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.Subscriptions(context.Background()); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
