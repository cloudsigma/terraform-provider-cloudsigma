package cloudsigma

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Each cohort retains the production five-second waiter and the stage cap.
func TestNative31AttachmentFaults(t *testing.T)    { native31AttachmentFaults(t, false, true) }
func TestNative31AttachmentCreate(t *testing.T)    { native31AttachmentFaults(t, false, false) }
func TestNative31AttachmentRawClone(t *testing.T)  { native31AttachmentFaults(t, true, true) }
func TestNative31AttachmentRawCreate(t *testing.T) { native31AttachmentFaults(t, true, false) }

// The paired pristine baseline changes only this test oracle, never SDK source.
const native31Baseline = false

type native31AckBody struct {
	data      []byte
	finish    func()
	readError bool
}

func (b *native31AckBody) Read(p []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	if b.finish != nil {
		b.finish()
		b.finish = nil
	}
	if b.readError {
		return n, io.ErrUnexpectedEOF
	}
	return n, io.EOF
}
func (b *native31AckBody) Close() error { return nil }

func native31AttachmentFaults(t *testing.T, raw, clone bool) {
	for bits := 0; bits < 8; bits++ {
		for _, fault := range []string{"401", "404", "500", "empty", "malformed", "disconnect", "cancel-body", "deadline-body", "ack-cancel-before", "ack-cancel-after", "ack-deadline-before", "ack-deadline-after", "ack-read-error"} {
			t.Run(fmt.Sprintf("%d/%s", bits, fault), func(t *testing.T) {
				old := http.DefaultTransport
				defer func() { http.DefaultTransport = old }()
				http.DefaultTransport = &http.Transport{}
				u, p, tok := "", "", ""
				if bits&1 != 0 {
					u = "u"
				}
				if bits&2 != 0 {
					p = "p"
				}
				if bits&4 != 0 {
					tok = "t"
				}
				incomplete := tok == "" && (u == "" || p == "")
				conf := &Config{Username: u, Password: p, Token: tok, Location: "zrh"}
				conf.loadAndValidate(context.Background(), "fixture")
				clients, e := conf.clients()
				if e != nil {
					t.Fatal(e)
				}
				ctx, cancel := context.WithCancel(context.Background())
				// A real caller deadline spans the unchanged five-second initial waiter.
				// It expires only when the selected PUT acknowledgment boundary waits for it.
				if strings.Contains(fault, "deadline") && strings.HasPrefix(fault, "ack-") {
					ctx, cancel = context.WithTimeout(context.Background(), 5100*time.Millisecond)
				}
				defer cancel()
				var end context.CancelFunc
				routes := []string{}
				counts := map[string]int{}
				native, legacy, gets, posts, puts := 0, 0, 0, 0, 0
				handler := func(r *http.Request) (*http.Response, error) {
					if tok != "" {
						if r.Header.Get("Authorization") != "Bearer "+tok {
							t.Error("token")
						}
					} else {
						a, b, _ := r.BasicAuth()
						if a != u || b != p {
							t.Error("basic")
						}
					}
					body := `{"uuid":"created","status":"unmounted"}`
					code := 200
					if r.Method == "POST" {
						posts++
						code = 202
						if !clone {
							code = 201
						}
						body = `{"objects":[{"uuid":"created"}]}`
					}
					if r.Method == "PUT" {
						puts++
					}
					if r.Method == "GET" {
						gets++
					}
					if r.Method == "PUT" {
						{
							switch fault {
							case "ack-cancel-before", "ack-cancel-after", "ack-deadline-before", "ack-deadline-after", "ack-read-error":
								finish := cancel
								if strings.Contains(fault, "deadline") {
									finish = func() { <-ctx.Done() }
								}
								ack := &native31AckBody{data: []byte(`{"uuid":"created"}`), readError: fault == "ack-read-error"}
								if strings.HasSuffix(fault, "before") {
									finish()
								} else if strings.HasSuffix(fault, "after") {
									ack.finish = finish
								}
								return &http.Response{StatusCode: 200, Header: make(http.Header), Body: ack, Request: r}, nil
							case "401":
								code = 401
							case "404":
								code = 404
							case "500":
								code = 500
							case "empty":
								body = ""
							case "malformed":
								body = "{"
							case "disconnect":
								return nil, fmt.Errorf("synthetic disconnect")
							default:
								// Return actual HTTP response headers then block inside Body.Read using net.Pipe.
								if fault == "deadline-body" {
									var child context.Context
									child, end = context.WithTimeout(ctx, 40*time.Millisecond)
									ctx = child
								} else {
									go func() { time.Sleep(20 * time.Millisecond); cancel() }()
								}
								tr := &http.Transport{DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
									a, b := net.Pipe()
									go func() {
										defer b.Close()
										_, err := http.ReadRequest(bufio.NewReader(b))
										if err != nil {
											return
										}
										fmt.Fprint(b, "HTTP/1.1 200 OK\r\nContent-Length: 999\r\nConnection: close\r\n\r\n{")
										<-ctx.Done()
									}()
									return a, nil
								}}
								rr := r.Clone(ctx)
								return (&http.Client{Transport: tr}).Do(rr)
							}
						}
					}
					return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}
				if clients.native != nil {
					clients.native = native31Transport(t, func(r *http.Request) (*http.Response, error) {
						native++
						counts["native/"+r.Method]++
						routes = append(routes, r.Method+" "+r.URL.RequestURI())
						return handler(r)
					})
				}
				var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
				if tok != "" {
					creds = cs.NewTokenCredentialsProvider(tok)
				}
				clients.legacy = cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
					legacy++
					counts["legacy/"+r.Method]++
					routes = append(routes, r.Method+" "+r.URL.RequestURI())
					return handler(r)
				})}))
				d := schema.TestResourceDataRaw(t, resourceCloudSigmaDrive().Schema, map[string]interface{}{"name": "before", "size": 1073741824, "media": "disk", "clone_drive_id": "source", "tags": []interface{}{"tag"}})
				if !clone {
					_ = d.Set("clone_drive_id", "")
				}
				var meta interface{} = clients
				if raw {
					meta = clients.legacy
				}
				ds := resourceCloudSigmaDriveCreate(ctx, d, meta)
				if end != nil {
					end()
				}
				// Pinned SDK decodes a complete acknowledgment despite cancellation or
				// bytes+read-error; native attachment deliberately rejects these boundaries.
				wantError := (!raw && bits == 3 && native31Baseline && fault != "ack-read-error") || incomplete || !strings.HasPrefix(fault, "ack-") || (!raw && bits == 3 && !native31Baseline)
				if ds.HasError() != wantError || puts != 1 && !incomplete {
					t.Fatal(ds, puts)
				}
				if incomplete {
					if native+legacy != 0 || d.Id() != "" {
						t.Fatal(native, legacy, d.Id())
					}
					return
				}
				extraRead := strings.HasPrefix(fault, "ack-") && (raw || bits != 3 || native31Baseline && fault == "ack-read-error")
				totalCalls, wantGets := 4, 2
				if extraRead {
					totalCalls++
					wantGets++
				}
				wn := 0
				if bits == 3 && !raw {
					wn = 4
					if native31Baseline {
						wn = 3
						if extraRead && fault == "ack-read-error" {
							wn++
						}
					}
				}
				if native != wn || legacy != totalCalls-wn || posts != 1 || gets != wantGets || d.Id() != "created" {
					t.Fatal(native, legacy, posts, gets, d.Id())
				}
				postRoute := "POST /api/2.0/drives/"
				if clone {
					postRoute = "POST /api/2.0/drives/source/action/?do=clone"
				}
				expected := []string{postRoute, "GET /api/2.0/drives/created/", "GET /api/2.0/drives/created/", "PUT /api/2.0/drives/created/"}
				if extraRead {
					expected = append(expected, "GET /api/2.0/drives/created/")
				}
				if fmt.Sprint(routes) != fmt.Sprint(expected) {
					t.Fatalf("routes=%v want=%v", routes, expected)
				}
				for _, method := range []string{"POST", "GET", "PUT"} {
					total := 1
					if method == "GET" {
						total = wantGets
					}
					n := 0
					if !raw && bits == 3 {
						n = total
						if native31Baseline && method == "PUT" {
							n = 0
						}
					}
					if counts["native/"+method] != n || counts["legacy/"+method] != total-n {
						t.Fatal(counts)
					}
				}
				t.Logf("native=%d legacy=%d POST=%d GET=%d PUT=%d retained=%s", native, legacy, posts, gets, puts, d.Id())
			})
		}
	}
}

// Preserve body-read failures across the real HTTP bridge: an incomplete
// declared length makes net/http deliver valid bytes plus unexpected EOF.
func native31Transport(t *testing.T, handler func(*http.Request) (*http.Response, error)) *cloudsigmaapi.Client {
	t.Helper()
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	http.DefaultTransport = &http.Transport{DialTLSContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			req, e := http.ReadRequest(bufio.NewReader(b))
			if e != nil {
				return
			}
			res, e := handler(req.WithContext(ctx))
			if e != nil {
				return
			}
			defer res.Body.Close()
			if ack, ok := res.Body.(*native31AckBody); ok && ack.readError {
				fmt.Fprintf(b, "HTTP/1.1 %d fixture\r\nContent-Length: 999\r\nConnection: close\r\n\r\n", res.StatusCode)
			} else {
				fmt.Fprintf(b, "HTTP/1.1 %d fixture\r\nConnection: close\r\n\r\n", res.StatusCode)
			}
			_, _ = io.Copy(b, res.Body)
		}()
		return a, nil
	}}
	c, e := cloudsigmaapi.New("zrh", "u", "p", "fixture")
	if e != nil {
		t.Fatal(e)
	}
	return c
}
