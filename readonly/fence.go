package readonly

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Fence struct {
	Base     http.RoundTripper
	Account  string
	Paths    map[string]bool
	Calls    int
	Admitted bool
	Deadline time.Time
}

func New(account string, paths map[string]bool) *Fence {
	return &Fence{Base: &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DisableKeepAlives: true}, Account: account, Paths: paths, Deadline: time.Now().Add(30 * time.Second)}
}
func (f *Fence) RoundTrip(q *http.Request) (res *http.Response, err error) {
	defer func() {
		if err != nil {
			f.Admitted = false
		}
	}()
	profile := q.URL.Path == "/api/2.0/profile/"
	if profile {
		f.Admitted = false
	}
	if q.Method != "GET" || q.URL.Scheme != "https" || q.URL.Host != "next.cloudsigma.com" || q.Host != "next.cloudsigma.com" || q.URL.RawPath != "" || q.URL.User != nil || q.URL.RawQuery != "" || q.URL.Fragment != "" || q.URL.Opaque != "" || (q.Body != nil && q.Body != http.NoBody) || q.URL.ForceQuery || f.Calls >= 4 || !time.Now().Before(f.Deadline) || (!profile && (!f.Admitted || !f.Paths[q.URL.Path])) {
		return nil, fmt.Errorf("readonly fence refusal")
	}
	ctx, cancel := context.WithDeadline(q.Context(), f.Deadline)
	defer cancel()
	f.Calls++
	r, e := f.Base.RoundTrip(q.Clone(ctx))
	if e != nil {
		if r != nil && r.Body != nil {
			r.Body.Close()
		}
		return nil, fmt.Errorf("transport")
	}
	if r == nil || r.Body == nil {
		return nil, fmt.Errorf("response")
	}
	defer r.Body.Close()
	b, e := io.ReadAll(io.LimitReader(r.Body, 262145))
	if e != nil || len(b) > 262144 || ctx.Err() != nil || r.StatusCode >= 300 && r.StatusCode < 400 {
		return nil, fmt.Errorf("response bound")
	}
	if profile {
		var v struct {
			UUID string `json:"uuid"`
		}
		if r.StatusCode != 200 || json.Unmarshal(b, &v) != nil || v.UUID != f.Account || f.Account == "" {
			return nil, fmt.Errorf("identity")
		}
		f.Admitted = true
	}
	r.Body = io.NopCloser(strings.NewReader(string(b)))
	return r, nil
}
