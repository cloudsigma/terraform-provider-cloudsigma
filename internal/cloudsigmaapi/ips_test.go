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

func TestIPs(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, object := range []string{
			`{"uuid":"185.12.6.183","gateway":"185.12.6.1","netmask":9007199254740993,"nameservers":["1.2.3.4","5.6.7.8"],"owner":{"uuid":"owner"},"server":{"uuid":"server"},"meta":{"n":9007199254740993},"unknown":{}}`,
			`{"uuid":"185.12.6.183","netmask":24}`, `null`, `{}`, `{"netmask":null,"gateway":null,"nameservers":null,"owner":null,"server":null}`, `{"nameservers":[null,"x"]}`, `{"gateway":"first","gateway":null}`, `{"netmask":"24"}`, `{"netmask":1.5}`, `{"netmask":9223372036854775808}`, `{"owner":[]}`, `[]`,
		} {
			t.Run(object, func(t *testing.T) {
				body := object
				if !single {
					body = `{"objects":[` + object + `],"meta":{"limit":0,"total_count":9,"next":"ignored"}}`
				}
				calls := 0
				c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
					calls++
					want := "/api/2.0/ips/185.12.6.183/"
					query := ""
					if !single {
						want = "/api/2.0/ips/detail/"
						query = ""
					}
					if r.URL.Path != want || r.URL.RawQuery != query || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("Authorization") == "" {
						t.Fatal(r.URL)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
				})
				oracle := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(func(r *http.Request) (*http.Response, error) {
					want := "/api/2.0/ips/185.12.6.183/"
					query := ""
					if !single {
						want = "/api/2.0/ips/detail/"
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
					got, e = c.IP(context.Background(), "185.12.6.183")
					old, _, oe = oracle.IPs.Get(context.Background(), "185.12.6.183")
				} else {
					got, e = c.IPs(context.Background())
					old, _, oe = oracle.IPs.List(context.Background())
				}
				if (e == nil) != (oe == nil) || calls != 1 {
					t.Fatalf("native %v oracle %v", e, oe)
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
	for _, id := range []string{"", ".", "..", "a\\b", "a b", "../profile", "a/b", "a?x", "a#x", "a%2fb", "a\n"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("unsafe dispatch"); return nil, nil })
		if _, e := c.IP(context.Background(), id); e == nil {
			t.Fatal(id)
		}
	}
	// Colon-containing opaque path segments are preserved, not claimed as a public IPv6 contract.
	for _, id := range []string{"185.12.6.183", "2001:db8::1", "opaque-id"} {
		calls := 0
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Path != "/api/2.0/ips/"+id+"/" || r.URL.RawQuery != "" {
				t.Fatal(r.URL)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		})
		if _, e := c.IP(context.Background(), id); e != nil || calls != 1 {
			t.Fatal(e, calls)
		}
	}
	for _, body := range []string{`{}`, `null`, `{"objects":null}`, `{"objects":[]}`, `{"objects":{}}`, `{"meta":{"limit":"bad"}}`} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		_, e := c.IPs(context.Background())
		if (e != nil) != (strings.Contains(body, `"objects":{}`) || strings.Contains(body, `"bad"`)) {
			t.Fatal(body, e)
		}
	}
}
func ipListReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.IPs(context.Background())
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func ipListEarlierDeadline(t *testing.T) {
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
	if _, e := c.IPs(ctx); e != nil {
		t.Fatal(e)
	}
}

func ipListLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.IPs(ctx)
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func ipListDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.IPs(ctx)
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestIPListBoundaries(t *testing.T) {
	t.Run("body-read", ipListReadError)
	t.Run("earlier-deadline", ipListEarlierDeadline)
	t.Run("late-cancel", ipListLateCancel)
	t.Run("deadline", ipListDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.IPs(context.Background()); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.IPs(context.Background()); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
func ipSingleReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.IP(context.Background(), "185.12.6.183")
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func ipSingleEarlierDeadline(t *testing.T) {
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
	if _, e := c.IP(ctx, "185.12.6.183"); e != nil {
		t.Fatal(e)
	}
}

func ipSingleLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.IP(ctx, "185.12.6.183")
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func ipSingleDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.IP(ctx, "185.12.6.183")
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestIPSingleBoundaries(t *testing.T) {
	t.Run("body-read", ipSingleReadError)
	t.Run("earlier-deadline", ipSingleEarlierDeadline)
	t.Run("late-cancel", ipSingleLateCancel)
	t.Run("deadline", ipSingleDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.IP(context.Background(), "185.12.6.183"); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.IP(context.Background(), "185.12.6.183"); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}

func TestIPHTTP(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, code := range []int{201, 204, 301, 404, 500} {
			b := &trackedBody{Reader: strings.NewReader("PRIVATE")}
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: code, Body: b}, nil })
			var e error
			if single {
				_, e = c.IP(context.Background(), "185.12.6.183")
			} else {
				_, e = c.IPs(context.Background())
			}
			if e == nil || strings.Contains(e.Error(), "PRIVATE") || !b.closed {
				t.Fatal(e)
			}
		}
	}
}
