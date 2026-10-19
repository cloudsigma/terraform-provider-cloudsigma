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

func TestTags(t *testing.T) {
	for _, body := range []string{`{"objects":[{"uuid":"tag-id","name":"blue","meta":{"n":9007199254740993},"owner":{"uuid":"o"},"resources":[{"uuid":"a","res_type":"server","owner":{"uuid":"nested"}},{"uuid":"b"}],"unknown":{}}],"meta":{"limit":20,"total_count":99,"next":"ignored"}}`, `null`, `{}`, `{"objects":null}`, `{"objects":[]}`, `{"objects":[null,{}]}`, `{"objects":[{"owner":null,"resources":null,"meta":null,"name":null}]}`, `{"objects":{}}`, `{"objects":[{"resources":{}}]}`, `{"objects":[{"owner":[]}]}`, `{"meta":{"limit":"bad"}}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			oldCalls := 0
			rt := func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/2.0/tags/" || r.URL.RawQuery != "" || r.Method != "GET" {
					t.Fatal(r.URL)
				}
				if r.Body != nil {
					b, e := io.ReadAll(r.Body)
					if e != nil || len(b) != 0 {
						t.Fatal("nonempty body", string(b), e)
					}
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			}
			c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("Authorization") == "" {
					t.Fatal(r.Header)
				}
				return rt(r)
			})
			old := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("fixture"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(func(r *http.Request) (*http.Response, error) { oldCalls++; return rt(r) })}))
			got, e := c.Tags(context.Background())
			want, _, oe := old.Tags.List(context.Background())
			if (e == nil) != (oe == nil) || calls != 1 || oldCalls != 1 {
				t.Fatal(e, oe, calls, oldCalls)
			}
			if e == nil {
				a, _ := json.Marshal(got)
				b, _ := json.Marshal(want)
				if !reflect.DeepEqual(a, b) {
					t.Fatalf("%s != %s", a, b)
				}
				if strings.Contains(body, `"nested"`) && (got[0].Resources[0].Owner.UUID != "nested" || got[0].Resources[1].UUID != "b") {
					t.Fatal(got)
				}
			}
		})
	}
}
func TestTagsHTTP(t *testing.T) {
	for _, code := range []int{201, 204, 301, 404, 500} {
		b := &trackedBody{Reader: strings.NewReader("PRIVATE")}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: code, Body: b}, nil })
		v, e := c.Tags(context.Background())
		if v != nil || e == nil || strings.Contains(e.Error(), "PRIVATE") || !b.closed {
			t.Fatal(v, e, b.closed)
		}
	}
}
func tagListReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Tags(context.Background())
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func tagListEarlierDeadline(t *testing.T) {
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
	if _, e := c.Tags(ctx); e != nil {
		t.Fatal(e)
	}
}

func tagListLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Tags(ctx)
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func tagListDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Tags(ctx)
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestTagListBoundaries(t *testing.T) {
	t.Run("body-read", tagListReadError)
	t.Run("earlier-deadline", tagListEarlierDeadline)
	t.Run("late-cancel", tagListLateCancel)
	t.Run("deadline", tagListDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.Tags(context.Background()); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.Tags(context.Background()); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
