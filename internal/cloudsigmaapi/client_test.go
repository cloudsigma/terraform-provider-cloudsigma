package cloudsigmaapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fakeDoer func(*http.Request) (*http.Response, error)

func (f fakeDoer) Do(r *http.Request) (*http.Response, error) { return f(r) }

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }
func TestClient(t *testing.T) {
	for location, host := range map[string]string{"ZRH": "zrh.cloudsigma.com", "dub": "ec.servecentric.com", "ruh": "ruh.cld.v2.sa", "adb": "siaflex.cloud", "wdc": "wdc.alpha3cloud.com"} {
		c, err := New(location, "fixture-user", "fixture-pass", "fixture-agent")
		if err != nil {
			t.Fatal(err)
		}
		h := c.transport.(*http.Client)
		tr := h.Transport.(*http.Transport)
		if h.Timeout != 60*time.Second || !tr.DisableCompression || tr.Proxy == nil || h.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
			t.Fatal("transport policy")
		}
		calls := 0
		body := &trackedBody{Reader: strings.NewReader(`{"uuid":"fixture","first_name":"name","unknown":{"a":[1]}}`)}
		c.transport = fakeDoer(func(r *http.Request) (*http.Response, error) {
			calls++
			u, p, ok := r.BasicAuth()
			if !ok || u != "fixture-user" || p != "fixture-pass" || r.URL.String() != "https://"+host+"/api/2.0/profile/" || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("User-Agent") != "fixture-agent" {
				t.Fatal("request contract")
			}
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) > 60*time.Second {
				t.Fatal("deadline")
			}
			return &http.Response{StatusCode: 200, Body: body}, nil
		})
		p, err := c.Profile(context.Background())
		if err != nil || p.UUID != "fixture" || p.FirstName != "name" || calls != 1 || !body.closed {
			t.Fatalf("profile %v %v", p, err)
		}
	}
	for _, location := range []string{"", "https://evil", "a/b", "a?x", "a#x", "a@b", "%2f", "..", "a.b"} {
		if _, err := New(location, "", "", ""); err == nil {
			t.Fatal(location)
		}
	}
}

func TestTokenClient(t *testing.T) {
	c, err := NewToken("zrh", "fixture-token", "fixture-agent")
	if err != nil {
		t.Fatal(err)
	}
	body := &trackedBody{Reader: strings.NewReader(`{"uuid":"fixture"}`)}
	c.transport = fakeDoer(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", got)
		}
		if _, _, ok := r.BasicAuth(); ok {
			t.Fatal("unexpected basic auth")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: body, Header: make(http.Header), Request: r}, nil
	})
	if _, err := c.Profile(context.Background()); err != nil || !body.closed {
		t.Fatalf("profile=%v closed=%v", err, body.closed)
	}
	if _, err := NewToken("zrh", "", "fixture-agent"); err == nil {
		t.Fatal("accepted an empty bearer token")
	}
}
func TestProfile(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{{`{}`, true}, {`{"uuid":null,"address":null}`, true}, {`{"uuid":""}`, true}, {`{"uuid":"x"} `, true}, {``, false}, {`null`, false}, {`[]`, false}, {`1`, false}, {`"x"`, false}, {`{"uuid":1}`, false}, {`{"uuid":true}`, false}, {`{"uuid":{}}`, false}, {`{"uuid":[]}`, false}, {`{"uuid":"x","uuid":"y"}`, false}, {`{"unknown":1,"unknown":2}`, false}, {`{} {}`, false}, {`{"unknown":}`, false}} {
		_, err := decodeProfile([]byte(tc.body))
		if (err == nil) != tc.valid {
			t.Fatalf("%s: %v", tc.body, err)
		}
	}
	for _, tc := range []struct {
		status                       int
		body, encoding, id, category string
	}{
		{200, `{}`, "", "safe-ID:1", ""}, {200, strings.Repeat(" ", responseLimit-2) + `{}`, "", "", ""}, {200, strings.Repeat(" ", responseLimit-1) + `{}`, "", "", "response_too_large"}, {200, `null`, "", "safe", "decode_error"}, {200, `{}`, "gzip", "", "unsupported_encoding"}, {204, "SECRET", "", "", "unexpected_status"}, {302, "SECRET", "", "", "http_status"}, {401, "SECRET", "", "", "http_status"}, {403, "SECRET", "", "", "http_status"}, {404, "SECRET", "", "", "http_status"}, {500, "SECRET", "", "invalid secret", "http_status"}} {
		c, _ := New("zrh", "u", "p", "")
		b := &trackedBody{Reader: strings.NewReader(tc.body)}
		calls := 0
		c.transport = fakeDoer(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: tc.status, Body: b, Header: http.Header{"Content-Encoding": []string{tc.encoding}, "X-Request-Id": []string{tc.id}, "Location": []string{"SECRET"}}}, nil
		})
		_, err := c.Profile(context.Background())
		if calls != 1 || !b.closed {
			t.Fatal("dispatch/close")
		}
		if tc.category == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			e, ok := err.(*Error)
			if !ok || e.Category != tc.category || e.Status != tc.status || strings.Contains(e.Error(), "SECRET") || strings.Contains(e.Error(), "invalid secret") {
				t.Fatalf("unsafe error %v", err)
			}
		}
	}
	for _, id := range []string{"bad value", "a\n", strings.Repeat("x", 129), "é"} {
		if requestID(id) != "" {
			t.Fatal("request id")
		}
	}
	for _, tc := range []struct {
		err      error
		category string
	}{{errors.New("SECRET https://secret"), "transport"}, {context.Canceled, "cancelled"}, {context.DeadlineExceeded, "timeout"}} {
		c, _ := New("zrh", "u", "p", "")
		b := &trackedBody{Reader: strings.NewReader("")}
		c.transport = fakeDoer(func(*http.Request) (*http.Response, error) { return &http.Response{Body: b}, tc.err })
		_, err := c.Profile(context.Background())
		if err.Error() != tc.category || !b.closed {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, _ := New("zrh", "u", "p", "")
	c.transport = fakeDoer(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	if _, err := c.Profile(ctx); err.Error() != "cancelled" {
		t.Fatal(err)
	}
}

func TestClientRedirect(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		c, _ := New("zrh", "fixture", "fixture", "")
		h := c.transport.(*http.Client)
		calls := 0
		b := &trackedBody{Reader: strings.NewReader("SECRET")}
		h.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: code, Header: http.Header{"Location": []string{"https://other.invalid/SECRET"}}, Body: b, Request: r}, nil
		})
		_, err := c.Profile(context.Background())
		if err == nil || calls != 1 || !b.closed || strings.Contains(err.Error(), "SECRET") {
			t.Fatalf("redirect %d: %v", code, err)
		}
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
