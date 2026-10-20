package cloudsigmaapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestKeypairUpdate(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 205, 301, 302, 307, 308, 400, 401, 403, 404, 409, 429, 500} {
		for _, input := range []KeypairUpdateRequest{{Name: "name", UUID: "fixture"}, {Name: "name", UUID: "fixture", PrivateKey: "SYNTHETIC-PRIVATE", PublicKey: "\nSYNTHETIC-PUBLIC\n"}, {}} {
			calls := 0
			var wires []map[string]string
			var bodies []*trackedBody
			rt := func(q *http.Request) (*http.Response, error) {
				calls++
				if q.Method != "PUT" || q.URL.Path != "/api/2.0/keypairs/fixture/" || q.URL.RawQuery != "" {
					t.Fatal(q.URL)
				}
				raw, e := io.ReadAll(q.Body)
				if e != nil {
					t.Fatal(e)
				}
				var wire map[string]string
				if e = json.Unmarshal(raw, &wire); e != nil {
					t.Fatal(e)
				}
				wires = append(wires, wire)
				b := &trackedBody{Reader: strings.NewReader(`{"uuid":"returned","name":"returned-name","private_key":"SYNTHETIC-PRIVATE","public_key":"SYNTHETIC-PUBLIC"}`)}
				bodies = append(bodies, b)
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: b, Request: q}, nil
			}
			c := reviewNativeNew(rt)
			v, e := c.UpdateKeypair(context.Background(), "fixture", input)
			if (e == nil) != (code == 200) {
				t.Fatal(code, e)
			}
			if e == nil && (v.UUID != "returned" || v.PrivateKey != "SYNTHETIC-PRIVATE") {
				t.Fatal("projection")
			}
			old := cs.NewClient(cs.NewTokenCredentialsProvider("fixture"), cs.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}))
			_, _, oe := old.Keypairs.Update(context.Background(), "fixture", &cs.KeypairUpdateRequest{Keypair: &cs.Keypair{Name: input.Name, UUID: input.UUID, PrivateKey: input.PrivateKey, PublicKey: input.PublicKey}})
			if (oe == nil) != (code >= 200 && code < 300) || calls != 2 || !reflect.DeepEqual(wires[0], wires[1]) {
				t.Fatal(code, oe, wires)
			}
			expected := map[string]string{}
			for k, v := range map[string]string{"name": input.Name, "uuid": input.UUID, "private_key": input.PrivateKey, "public_key": input.PublicKey} {
				if v != "" {
					expected[k] = v
				}
			}
			if !reflect.DeepEqual(wires[0], expected) {
				t.Fatal(wires)
			}
			for _, b := range bodies {
				if !b.closed {
					t.Fatal("closure")
				}
			}
		}
	}
}
func TestKeypairUpdateBoundaries(t *testing.T) {
	for _, id := range []string{"", "../x", "a/b", "a?b", "a#b", "a%2fb", "a\n"} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("dispatch"); return nil, nil })
		if _, e := c.UpdateKeypair(context.Background(), id, KeypairUpdateRequest{}); e == nil {
			t.Fatal(id)
		}
	}
	for _, expired := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if expired {
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
		}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("dispatch"); return nil, nil })
		if _, e := c.UpdateKeypair(ctx, "fixture", KeypairUpdateRequest{}); e == nil {
			t.Fatal("context")
		}
	}
	for _, body := range []string{"", `{"private_key":123}`, `SYNTHETIC-PRIVATE`, strings.Repeat("x", responseLimit+1)} {
		b := &trackedBody{Reader: strings.NewReader(body)}
		c := reviewNativeNew(func(q *http.Request) (*http.Response, error) {
			if _, ok := q.Context().Deadline(); !ok {
				t.Fatal("deadline")
			}
			if q.Header.Get("Accept-Encoding") != "identity" || q.Header.Get("Cookie") != "" {
				t.Fatal("headers")
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: b}, nil
		})
		if _, e := c.UpdateKeypair(context.Background(), "fixture", KeypairUpdateRequest{}); e == nil || strings.Contains(e.Error(), "PRIVATE") {
			t.Fatal(e)
		}
		if !b.closed {
			t.Fatal("closure")
		}
	}
	for _, body := range []string{`null`, `{}`, `{"objects":[]}`, `{"uuid":null,"private_key":null}`, `{"uuid":"fixture"} trailing`} {
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, e := c.UpdateKeypair(context.Background(), "fixture", KeypairUpdateRequest{}); e != nil {
			t.Fatal("existing permissive decode", e)
		}
	}
	in := KeypairUpdateRequest{PrivateKey: "SYNTHETIC-PRIVATE"}
	if strings.Contains(fmt.Sprintf("%v %+v %#v", in, in, in), "PRIVATE") {
		t.Fatal("redaction")
	}
}
func TestKeypairUpdateFailures(t *testing.T) {
	for _, kind := range []string{"transport", "nil", "nilbody", "read", "encoding", "cancel", "late-read"} {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		b := &trackedBody{Reader: strings.NewReader(`{}`)}
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			calls++
			switch kind {
			case "transport":
				return nil, io.ErrUnexpectedEOF
			case "nil":
				return nil, nil
			case "cancel":
				cancel()
			case "read":
				b.Reader = updateBadReader{}
			case "late-read":
				b.Reader = updateCancelReader{cancel: cancel}
			}
			r := &http.Response{StatusCode: 200, Header: make(http.Header), Body: b}
			if kind == "nilbody" {
				r.Body = nil
			}
			if kind == "encoding" {
				r.Header.Set("Content-Encoding", "gzip")
			}
			return r, nil
		})
		if _, e := c.UpdateKeypair(ctx, "fixture", KeypairUpdateRequest{}); e == nil || calls != 1 {
			t.Fatal(kind, e, calls)
		}
		cancel()
		if kind != "transport" && kind != "nil" && kind != "nilbody" && !b.closed {
			t.Fatal("closure")
		}
	}
}

type updateBadReader struct{}

func (updateBadReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type updateCancelReader struct{ cancel context.CancelFunc }

func (r updateCancelReader) Read(p []byte) (int, error) { r.cancel(); return copy(p, `{}`), io.EOF }
