package readonly

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fixture func(*http.Request) (*http.Response, error)

func (f fixture) RoundTrip(q *http.Request) (*http.Response, error) { return f(q) }
func TestFence(t *testing.T) {
	f := New("synthetic", map[string]bool{"/api/2.0/tags/fixture/": true})
	f.Base = fixture(func(q *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"uuid":"synthetic"}`)), Header: make(http.Header)}, nil
	})
	q, _ := http.NewRequest("GET", "https://next.cloudsigma.com/api/2.0/tags/fixture/", nil)
	if _, e := f.RoundTrip(q); e == nil || f.Calls != 0 {
		t.Fatal("admission")
	}
	p, _ := http.NewRequest("GET", "https://next.cloudsigma.com/api/2.0/profile/", nil)
	if _, e := f.RoundTrip(p); e != nil {
		t.Fatal(e)
	}
	if _, e := f.RoundTrip(q); e != nil {
		t.Fatal(e)
	}
	for _, kind := range []string{"origin", "method", "query", "raw", "deadline"} {
		x := *f
		r := q.Clone(q.Context())
		u := *q.URL
		r.URL = &u
		switch kind {
		case "origin":
			r.URL.Host = "evil"
		case "method":
			r.Method = "POST"
		case "query":
			r.URL.RawQuery = "limit=0"
		case "raw":
			r.URL.RawPath = "/encoded"
		case "deadline":
			x.Deadline = time.Now().Add(-time.Second)
		}
		n := x.Calls
		if _, e := x.RoundTrip(r); e == nil || x.Calls != n {
			t.Fatal(kind)
		}
	}
	for f.Calls < 4 {
		if _, e := f.RoundTrip(q); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := f.RoundTrip(q); e == nil {
		t.Fatal("budget")
	}
}
