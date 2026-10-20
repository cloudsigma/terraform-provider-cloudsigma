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

func TestServer(t *testing.T) {
	for _, body := range []string{
		`{"uuid":"selected","cpu":9007199254740993,"mem":9007199254740993,"epcs":[{"size":9007199254740993}],"jobs":[{"n":9007199254740993}],"drives":[{"boot_order":1,"drive":{"size":9007199254740993,"licenses":[{"amount":9007199254740993,"license":{"free":9007199254740993,"initial_purchase_requirement":1.2300,"subsequent_purchase_requirement":1e99}}]}}],"nics":[{"boot_order":null,"firewall_policy":{"uuid":"fw","rules":[{"action":"accept"}]},"ip_v4_conf":{"conf":"static","ip":{"uuid":"192.0.2.1","netmask":24}},"vlan":{"uuid":"vlan","subscription":{"id":9007199254740993}}}],"meta":{"n":9007199254740993,"nested":{}},"pubkeys":[{"uuid":"key","private_key":"synthetic"}],"runtime":{"nics":[{"interface_type":"public","ip_v4":{"uuid":"192.0.2.2"}}]},"status":"running","unknown":{"x":1}}`,
		`null`, `{}`, `{"drives":[],"nics":[],"jobs":null}`, `{"drives":[null,{"drive":null}],"nics":[null],"meta":null}`, `{"drives":[{"drive":{"licenses":[{"license":{"free":1,"free":null}}]}}]}`, `{"cpu":"bad"}`, `{"mem":1.5}`, `{"cpu":9223372036854775808}`, `{"runtime":[]}`, `{"drives":[{"drive":{"licenses":[{"license":{"free":true}}]}}]}`, `[]`,
	} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			rt := func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Body != nil {
					data, err := io.ReadAll(r.Body)
					if err != nil || len(data) != 0 {
						t.Fatal("nonempty request body")
					}
				}
				if r.Method != "GET" || r.URL.RequestURI() != "/api/2.0/servers/selected/" {
					t.Fatal(r.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			}
			c := reviewNativeNew(rt)
			sdk := cloudsigma.NewClient(cloudsigma.NewTokenCredentialsProvider("synthetic"), cloudsigma.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt)}))
			got, e := c.Server(context.Background(), "selected")
			old, _, oe := sdk.Servers.Get(context.Background(), "selected")
			if (e == nil) != (oe == nil) || calls != 2 {
				t.Fatalf("native=%v oracle=%v", e, oe)
			}
			if e == nil {
				a, _ := json.Marshal(got)
				b, _ := json.Marshal(old)
				if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(got.Jobs, old.Jobs) {
					t.Fatalf("%s != %s", a, b)
				}
			}
		})
	}
	for _, id := range []string{"", "../profile", "a/b", "a?x", "a#x", "a%2fb", "a\\b", "a\n", "é"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("unsafe dispatch"); return nil, nil })
		if _, e := c.Server(context.Background(), id); e == nil {
			t.Fatal(id)
		}
	}
}

func serverSingleReadError(t *testing.T) {
	b := &trackedBody{Reader: reviewFailReader{}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	_, e := c.Server(context.Background(), "selected")
	if e == nil || e.Error() != "transport status=200" || !b.closed {
		t.Fatalf("%v close=%v", e, b.closed)
	}
}
func serverSingleEarlierDeadline(t *testing.T) {
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
	if _, e := c.Server(ctx, "selected"); e != nil {
		t.Fatal(e)
	}
}

func serverSingleLateCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: cancel}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	p, e := c.Server(ctx, "selected")
	if p != nil || e == nil || e.Error() != "cancelled status=200" || !b.closed {
		t.Fatalf("accepted profile after cancellation: %+v error=%v closed=%v", p, e, b.closed)
	}
}

func serverSingleDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	b := &trackedBody{Reader: &reviewCancelReader{cancel: func() { <-ctx.Done() }}}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: b}, nil })
	profile, err := c.Server(ctx, "selected")
	if profile != nil || err == nil || err.Error() != "timeout status=200" || !b.closed {
		t.Fatalf("deadline: %v %v closed=%v", profile, err, b.closed)
	}
}

func TestServerBoundaries(t *testing.T) {
	t.Run("body-read", serverSingleReadError)
	t.Run("earlier-deadline", serverSingleEarlierDeadline)
	t.Run("late-cancel", serverSingleLateCancel)
	t.Run("deadline", serverSingleDeadline)
	for _, body := range []string{strings.Repeat(" ", responseLimit+1), `{"objects":[]}`} {
		c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > requestTimeout {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Encoding": []string{"gzip"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := c.Server(context.Background(), "selected"); err == nil {
			t.Fatal("encoding accepted")
		}
	}
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", responseLimit+1)))}, nil
	})
	if _, err := c.Server(context.Background(), "selected"); err == nil || !strings.Contains(err.Error(), "response_too_large") {
		t.Fatal(err)
	}
}
