package cloudsigmaapi

import (
	"bytes"
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

// This oracle uses the actual SDK service decoder, including its private nested
// license decoder. Public-model Unmarshal is intentionally not the oracle.
func TestServerCreate(t *testing.T) {
	rich := `{"uuid":"created","auto_start":true,"context":true,"cpu":2000,"cpu_type":"amd","cpus_instead_of_cores":true,"enable_numa":true,"epcs":[{"size":8}],"hypervisor":"kvm","mem":536870912,"meta":{"n":9007199254740993,"nested":{"x":null}},"name":"created","owner":{"uuid":"owner"},"resource_uri":"/servers/created/","smp":2,"status":"stopped","vnc_password":"synthetic","jobs":[{"uuid":"job"}],"drives":[{"boot_order":2,"dev_channel":"0:2","device":"virtio","drive":{"uuid":"drive","size":9007199254740993,"licenses":[{"amount":3,"license":{"free":9007199254740993,"initial_purchase_requirement":1.2300,"subsequent_purchase_requirement":"42"}}],"meta":{"n":9007199254740993},"runtime":{"storage_type":"nvme"}}}],"nics":[{"boot_order":1,"mac":"aa:bb","model":"virtio","firewall_policy":{"uuid":"fw","rules":[{"action":"accept","dst_port":"443"}],"tags":[{"uuid":"tag"}]},"ip_v4_conf":{"conf":"static","ip":{"uuid":"192.0.2.1","netmask":24}},"ip_v6_conf":{"conf":"manual"},"vlan":{"uuid":"vlan","subscription":{"id":123}}}],"pubkeys":[{"uuid":"key","public_key":"synthetic","private_key":"synthetic","meta":{"nested":true}}],"tags":[{"uuid":"tag","name":"tag"}],"runtime":{"nics":[{"interface_type":"public","ip_v4":{"uuid":"192.0.2.1"},"ip_v6":{"uuid":"::1"}}]},"unknown":{"ignored":true}}`
	for _, body := range []string{rich, `{"uuid":"created"}`, `{"uuid":"created","drives":[null,{"drive":null}],"jobs":null,"nics":[],"pubkeys":[],"tags":[]}`} {
		input := ServerCreateRequest{CPU: 2000, Memory: 536870912, Name: "input", SMP: 2, VNCPassword: "synthetic", Meta: map[string]interface{}{"keep": "value"}, NICs: []ServerNIC{{IP4Configuration: &ServerIPConfiguration{Type: "dhcp"}}}, PublicKeys: []Keypair{{UUID: "key"}}, Tags: []Tag{{UUID: "tag"}}}
		var requests [][]byte
		rt := func(r *http.Request) (*http.Response, error) {
			if r.Method != "POST" || r.URL.RequestURI() != "/api/2.0/servers/" {
				t.Fatal(r.Method, r.URL)
			}
			b, e := io.ReadAll(r.Body)
			if e != nil {
				t.Fatal(e)
			}
			requests = append(requests, bytes.TrimSpace(b))
			return &http.Response{StatusCode: 201, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[` + body + `]}`))}, nil
		}
		native, e := reviewNativeNew(rt).CreateServer(context.Background(), input)
		if e != nil {
			t.Fatal(e)
		}
		wire, _ := json.Marshal(input)
		var request cs.Server
		if e = json.Unmarshal(wire, &request); e != nil {
			t.Fatal(e)
		}
		sdk := cs.NewClient(cs.NewTokenCredentialsProvider("synthetic"), cs.WithHTTPClient(&http.Client{Transport: subscriptionOracleRT(rt)}))
		old, _, e := sdk.Servers.Create(context.Background(), &cs.ServerCreateRequest{Servers: []cs.Server{request}})
		if e != nil {
			t.Fatal(e)
		}
		a, _ := json.Marshal(native)
		b, _ := json.Marshal(old[0])
		var av, bv interface{}
		_ = json.Unmarshal(a, &av)
		_ = json.Unmarshal(b, &bv)
		if !reflect.DeepEqual(av, bv) || !bytes.Equal(native.Jobs, old[0].Jobs) {
			t.Fatalf("projection\n%s\n%s", a, b)
		}
		if len(requests) != 2 || !bytes.Equal(requests[0], requests[1]) {
			t.Fatal("request parity", requests)
		}
		// Full response attachment PUT compatibility, preserving legacy metadata rounding.
		var projected cs.Server
		if e = json.Unmarshal(a, &projected); e != nil {
			t.Fatal(e)
		}
		added := cs.ServerDrive{BootOrder: len(old[0].Drives) + 1, Device: "virtio", Drive: &cs.Drive{UUID: "added"}}
		projected.Drives = append(projected.Drives, added)
		old[0].Drives = append(old[0].Drives, added)
		a, _ = json.Marshal(&cs.ServerUpdateRequest{Server: &projected})
		b, _ = json.Marshal(&cs.ServerUpdateRequest{Server: &old[0]})
		if !bytes.Equal(a, b) {
			t.Fatalf("PUT mismatch\n%s\n%s", a, b)
		}
	}
}

func TestServerCreateBoundaries(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 301, 401, 409, 429, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			c := reviewNativeNew(func(r *http.Request) (*http.Response, error) {
				calls++
				u, p, ok := r.BasicAuth()
				if !ok || u != "synthetic" || p != "synthetic" {
					t.Fatal("auth")
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"created"}]}`))}, nil
			})
			v, e := c.CreateServer(context.Background(), ServerCreateRequest{})
			if calls != 1 || (e == nil) != (code == 201) || (v != nil) != (code == 201) {
				t.Fatal(v, e, calls)
			}
		})
	}
	for _, body := range []string{`{}`, `null`, `{"objects":[]}`, `{"objects":[null]}`, `{"objects":[{"uuid":"../bad"}]}`, `{"objects":[{"uuid":"ok"},{"uuid":"other"}]}`, `{"objects":[{"uuid":"ok","cpu":"bad"}]}`, `{"objects":[{"uuid":"ok"}]} garbage`, `{"objects":[{"uuid":"ok"}]`, strings.Repeat("x", responseLimit+1)} {
		calls := 0
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if v, e := c.CreateServer(context.Background(), ServerCreateRequest{}); v != nil || e == nil || calls != 1 {
			t.Fatal(v, e, calls)
		}
	}
	for _, body := range []string{`{"objects":[{"uuid":"created"}]}`, `{"objects":[{"uuid":"created"}]`} {
		ctx, cancel := context.WithCancel(context.Background())
		b := &reviewBody{Reader: strings.NewReader(body), cancel: cancel}
		calls := 0
		c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 201, Body: b}, nil
		})
		v, e := c.CreateServer(ctx, ServerCreateRequest{})
		if e == nil || calls != 1 || !b.closed {
			t.Fatal(v, e, calls)
		}
		if (v != nil) != (strings.HasSuffix(body, "}")) {
			t.Fatal("identity from incomplete ack", v)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := reviewNativeNew(func(*http.Request) (*http.Response, error) { t.Fatal("cancelled dispatch"); return nil, nil })
	if v, e := c.CreateServer(ctx, ServerCreateRequest{}); v != nil || e == nil {
		t.Fatal(v, e)
	}
	// Unknown commit: transport failure cannot be repaired by another POST or GET.
	calls := 0
	c = reviewNativeNew(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, fmt.Errorf("synthetic unknown commit")
	})
	if v, e := c.CreateServer(context.Background(), ServerCreateRequest{}); v != nil || e == nil || calls != 1 {
		t.Fatal(v, e, calls)
	}
}

// The body, not RoundTrip, blocks until cancellation. This exercises Read's
// error boundary after a 201 header without inventing an acknowledged identity.
type createBlockedBody struct {
	ctx     context.Context
	entered chan struct{}
	closed  bool
}

func (b *createBlockedBody) Read([]byte) (int, error) {
	close(b.entered)
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}
func (b *createBlockedBody) Close() error { b.closed = true; return nil }
func TestServerCreateBlockedRead(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
			}
			defer cancel()
			body := &createBlockedBody{ctx: ctx, entered: make(chan struct{})}
			calls := 0
			c := reviewNativeNew(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 201, Body: body}, nil
			})
			if !deadline {
				go func() { <-body.entered; cancel() }()
			}
			v, e := c.CreateServer(ctx, ServerCreateRequest{})
			if v != nil || e == nil || calls != 1 || !body.closed {
				t.Fatal(v, e, calls, body.closed)
			}
		})
	}
}
