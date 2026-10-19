package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestVLANs(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, object := range []string{
			`{"uuid":"vlan-id","subscription":{"id":9007199254740993,"resource_uri":"subscription"},"owner":{"uuid":"owner"},"servers":[{"uuid":"a"},{"uuid":"b"}],"tags":[{"name":"tag","resources":[{"uuid":"r","res_type":"server","owner":{"uuid":"o"}}]}],"meta":{"name":"blue","n":9007199254740993},"unknown":{}}`,
			`{"uuid":"vlan-id","subscription":{"id":12}}`, `null`, `{}`, `{"subscription":null,"owner":null,"servers":null,"tags":null,"meta":null}`, `{"servers":[null,{}]}`, `{"uuid":"first","uuid":null}`, `{"subscription":{"id":"12"}}`, `{"subscription":{"id":1.5}}`, `{"subscription":{"id":9223372036854775808}}`, `{"owner":[]}`, `[]`, `{"tags":[{"resources":{}}]}`,
		} {
			t.Run(object, func(t *testing.T) {
				body := object
				if !single {
					body = `{"objects":[` + object + `],"meta":{"limit":0,"total_count":9,"next":"ignored"}}`
				}
				calls := 0
				oracleCalls := 0
				c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
					calls++
					want := "/api/2.0/vlans/vlan-id/"
					query := ""
					if !single {
						want = "/api/2.0/vlans/detail/"
						query = ""
					}
					if r.URL.Path != want || r.URL.RawQuery != query || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("Authorization") == "" {
						t.Fatal(r.URL)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
				})
				oracle := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(func(r *http.Request) (*http.Response, error) {
					oracleCalls++
					want := "/api/2.0/vlans/vlan-id/"
					query := ""
					if !single {
						want = "/api/2.0/vlans/detail/"
						query = ""
					}
					if r.URL.Path != want || r.URL.RawQuery != query {
						t.Fatal(r.URL)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}))
				var got, old any
				var e, oe error
				if single {
					got, e = c.VLAN(context.Background(), "vlan-id")
					old, _, oe = oracle.VLANs.Get(context.Background(), "vlan-id")
				} else {
					got, e = c.VLANs(context.Background())
					old, _, oe = oracle.VLANs.List(context.Background())
				}
				if (e == nil) != (oe == nil) || calls != 1 || oracleCalls != 1 {
					t.Fatalf("native %v oracle %v", e, oe)
				}
				if e == nil && strings.Contains(object, `"id":9007199254740993`) {
					var v VLAN
					if single {
						v = *got.(*VLAN)
					} else {
						v = got.([]VLAN)[0]
					}
					if v.Subscription.ID != 9007199254740993 || v.Servers[0].UUID != "a" || v.Servers[1].UUID != "b" || v.Tags[0].Resources[0].Owner.UUID != "o" {
						t.Fatal(v)
					}
				}
				if e == nil {
					a, _ := json.Marshal(got)
					b, _ := json.Marshal(old)
					if !reflect.DeepEqual(a, b) {
						t.Fatalf("%s != %s", a, b)
					}
				}
			})
		}
	}
	for _, id := range []string{"", ".", "..", "a\\b", "a b", "../profile", "a/b", "a?x", "a#x", "a%2fb", "a\n", "a:b", "a.b", "é"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("unsafe dispatch"); return nil, nil })
		if _, e := c.VLAN(context.Background(), id); e == nil {
			t.Fatal(id)
		}
	}
	// Colon-containing opaque path segments are preserved, not claimed as a public VLANv6 contract.
	for _, id := range []string{"vlan-id", "opaque-id"} {
		calls := 0
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Path != "/api/2.0/vlans/"+id+"/" || r.URL.RawQuery != "" {
				t.Fatal(r.URL)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		})
		if _, e := c.VLAN(context.Background(), id); e != nil || calls != 1 {
			t.Fatal(e, calls)
		}
	}
	for _, body := range []string{`{}`, `null`, `{"objects":null}`, `{"objects":[]}`, `{"objects":{}}`, `{"meta":{"limit":"bad"}}`} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		_, e := c.VLANs(context.Background())
		if (e != nil) != (strings.Contains(body, `"objects":{}`) || strings.Contains(body, `"bad"`)) {
			t.Fatal(body, e)
		}
	}
}
func vlanListReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.VLANs(context.Background())
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func vlanListEarlierDeadline(t *testing.T) {
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
	if _, e := c.VLANs(ctx); e != nil {
		t.Fatal(e)
	}
}

func vlanListLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.VLANs(ctx)
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func vlanListDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.VLANs(ctx)
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestVLANListBoundaries(t *testing.T) {
	t.Run("body-read", vlanListReadError)
	t.Run("earlier-deadline", vlanListEarlierDeadline)
	t.Run("late-cancel", vlanListLateCancel)
	t.Run("deadline", vlanListDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.VLANs(context.Background()); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.VLANs(context.Background()); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
func vlanSingleReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.VLAN(context.Background(), "vlan-id")
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func vlanSingleEarlierDeadline(t *testing.T) {
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
	if _, e := c.VLAN(ctx, "vlan-id"); e != nil {
		t.Fatal(e)
	}
}

func vlanSingleLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.VLAN(ctx, "vlan-id")
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func vlanSingleDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.VLAN(ctx, "vlan-id")
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestVLANSingleBoundaries(t *testing.T) {
	t.Run("body-read", vlanSingleReadError)
	t.Run("earlier-deadline", vlanSingleEarlierDeadline)
	t.Run("late-cancel", vlanSingleLateCancel)
	t.Run("deadline", vlanSingleDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.VLAN(context.Background(), "vlan-id"); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.VLAN(context.Background(), "vlan-id"); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}

func TestVLANHTTP(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, code := range []int{201, 204, 301, 404, 500} {
			b := &trackedBody{Reader: strings.NewReader("PRIVATE")}
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: code, Body: b}, nil })
			var e error
			if single {
				_, e = c.VLAN(context.Background(), "vlan-id")
			} else {
				_, e = c.VLANs(context.Background())
			}
			if e == nil || strings.Contains(e.Error(), "PRIVATE") || !b.closed {
				t.Fatal(e)
			}
		}
	}
}
