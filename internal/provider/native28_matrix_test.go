package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	fp "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

type n28RT func(*http.Request) (*http.Response, error)

func (f n28RT) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type n28Partial struct {
	text string
	done bool
}

func (b *n28Partial) Close() error { return nil }
func (b *n28Partial) Read(p []byte) (int, error) {
	if b.done {
		return 0, io.EOF
	}
	b.done = true
	return copy(p, b.text), fmt.Errorf("partial-read-sentinel")
}
func TestNative28Matrix(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		for _, raw := range []bool{false, true} {
			phases := []string{"success", "pending", "wait-cancel", "wait-deadline", "final-cancel", "final-deadline", "wait-partial", "final-partial", "wait-malformed", "final-malformed", "unknown-post", "deadline-policy", "private-body", "private-status", "private-id", "wait-null", "final-null", "wait-status", "final-status", "empty-ack", "invalid-ack", "null-ack", "pre-cancel", "pre-deadline", "ack-cancel", "ack-deadline"}
			if bits != 3 {
				phases = []string{"success", "unknown-post", "wait-partial", "final-partial", "empty-ack", "invalid-ack", "null-ack", "ack-cancel"}
			}
			for _, phase := range phases {
				t.Run(fmt.Sprintf("credentials%d/raw%t/%s", bits, raw, phase), func(t *testing.T) {
					duration := 8 * time.Second
					if phase == "pending" {
						duration = 22 * time.Second
					}
					ctx, cancel := context.WithTimeout(context.Background(), duration)
					if phase == "deadline-policy" {
						cancel()
						ctx, cancel = context.WithCancel(context.Background())
					}
					defer cancel()
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
					old := http.DefaultClient
					defer func() { http.DefaultClient = old }()
					order := []string{}
					native, legacy, reads, posts := 0, 0, 0, 0
					blocked := false
					reconciling := false
					var stateMu sync.Mutex
					snapshotState := func() ([]string, int, int, int, int, bool, bool) {
						stateMu.Lock()
						defer stateMu.Unlock()
						return append([]string(nil), order...), native, legacy, reads, posts, blocked, reconciling
					}
					bodyDone := make(chan struct{}, 1)
					transport := func(n bool) n28RT {
						return func(req *http.Request) (*http.Response, error) {
							stateMu.Lock()
							if n {
								native++
							} else {
								legacy++
							}
							order = append(order, fmt.Sprintf("%t:%s:%s", n, req.Method, req.URL.Path))
							stateMu.Unlock()
							text := `{"uuid":"drive","status":"unmounted"}`
							code := http.StatusOK
							var body io.ReadCloser
							if strings.Contains(req.URL.Path, "snapshots") {
								text = `{"uuid":"created","drive":{"uuid":"drive"},"name":"done","status":"available","resource_uri":"uri","timestamp":"time","meta":{"exact":9007199254740993,"private":"PRIVATE_METADATA_CANARY"},"allocated_size":9007199254740993}`
								if req.Method == "POST" {
									stateMu.Lock()
									posts++
									stateMu.Unlock()
									if phase == "private-id" {
										text = `{"uuid":"PRIVATE_ID_CANARY","drive":{"uuid":"drive"},"status":"available"}`
									}
									if n {
										if bits != 3 {
											t.Fatal("native mutation with incomplete credentials")
										}
										code = http.StatusCreated
									}
									if phase == "unknown-post" {
										return nil, fmt.Errorf("commit outcome unknown")
									}
									if phase == "invalid-ack" {
										text = `{"uuid":"bad/id"}`
									}
									text = `{"objects":[` + text + `]}`
									if phase == "empty-ack" {
										text = `{"objects":[]}`
									}
									if phase == "null-ack" {
										text = `{"objects":[null]}`
									}
									if phase == "ack-cancel" {
										cancel()
									}
									if phase == "ack-deadline" {
										<-ctx.Done()
									}
								} else {
									stateMu.Lock()
									reads++
									currentReads, currentReconciling := reads, reconciling
									stateMu.Unlock()
									if phase != "deadline-policy" && !currentReconciling {
										want, _ := ctx.Deadline()
										got, ok := req.Context().Deadline()
										if !ok || !got.Equal(want) {
											t.Fatal("caller deadline changed", got, want)
										}
									}
									if phase == "deadline-policy" {
										deadline, ok := req.Context().Deadline()
										left := time.Until(deadline)
										t.Logf("DEADLINE native=%v exists=%v remaining=%s", n, ok, left)
										if ok {
											t.Fatal("introduced deadline", left)
										}
										if !n && ok {
											t.Fatal("legacy unexpectedly capped")
										}
									}
									if phase == "private-body" && !currentReconciling {
										text = `{"PRIVATE_BODY_CANARY":`
									}
									if phase == "private-status" && !currentReconciling {
										text = `{"uuid":"created","status":"PRIVATE_STATUS_CANARY"}`
									}
									if phase == "private-id" {
										text = `{`
									}
									target := !currentReconciling && (strings.HasPrefix(phase, "wait-") && currentReads == 1 || strings.HasPrefix(phase, "final-") && currentReads == 2)
									if phase == "pending" && currentReads == 1 {
										text = `{"uuid":"created","status":"creating"}`
									}
									if target && strings.HasSuffix(phase, "null") {
										text = `null`
									}
									if target && strings.HasSuffix(phase, "status") {
										return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
									}
									if target && strings.HasSuffix(phase, "malformed") {
										text = `{"PRIVATE_BODY_CANARY":`
									}
									if target && strings.HasSuffix(phase, "partial") {
										body = &n28Partial{text: text}
									}
									if target && (strings.HasSuffix(phase, "cancel") || strings.HasSuffix(phase, "deadline")) {
										stateMu.Lock()
										blocked = true
										stateMu.Unlock()
										reader, writer := io.Pipe()
										body = reader
										// Real pipe-backed Body.Read: no bytes are written; release only after context cancellation/deadline.
										go func() {
											if strings.HasSuffix(phase, "cancel") {
												time.Sleep(20 * time.Millisecond)
												cancel()
											}
											<-ctx.Done()
											writer.CloseWithError(ctx.Err())
											bodyDone <- struct{}{}
										}()
									}
								}
							}
							if body == nil {
								body = io.NopCloser(strings.NewReader(text))
							}
							return &http.Response{StatusCode: code, Header: make(http.Header), Body: body, Request: req}, nil
						}
					}
					http.DefaultClient = &http.Client{Transport: transport(false)}
					var data interface{}
					if !raw {
						provider := New("qualification")()
						var s fp.SchemaResponse
						provider.Schema(ctx, fp.SchemaRequest{}, &s)
						typ := s.Schema.Type().TerraformType(ctx)
						attrs := map[string]tftypes.Value{}
						for k, v := range typ.(tftypes.Object).AttributeTypes {
							attrs[k] = tftypes.NewValue(v, nil)
						}
						for k, v := range map[string]string{"location": "zrh", "username": u, "password": p, "token": tok} {
							attrs[k] = tftypes.NewValue(tftypes.String, v)
						}
						var out fp.ConfigureResponse
						provider.Configure(ctx, fp.ConfigureRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: tftypes.NewValue(typ, attrs)}}, &out)
						valid := bits == 3 || bits == 4
						if out.Diagnostics.HasError() == valid {
							t.Fatal("configuration oracle", out.Diagnostics)
						}
						if !valid {
							if len(order) != 0 || out.ResourceData != nil {
								t.Fatal("invalid config dispatched")
							}
							t.Log("RESULT configuration-refusal zero-dispatch")
							return
						}
						data = out.ResourceData
					} else {
						var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
						mode := authBasic
						if tok != "" {
							creds = cs.NewTokenCredentialsProvider(tok)
							mode = authToken
						}
						data = &ProviderClients{Legacy: cs.NewClient(creds, cs.WithHTTPClient(http.DefaultClient)), Auth: mode}
					}
					clients := data.(*ProviderClients)
					// Preserve framework selection: raw incomplete Basic must not gain a native client.
					if clients.Auth == authBasic && u != "" && p != "" {
						clients.Native = n28Client(t, transport(true))
					}
					r := &snapshotResource{}
					var conf resource.ConfigureResponse
					r.Configure(ctx, resource.ConfigureRequest{ProviderData: data}, &conf)
					if conf.Diagnostics.HasError() {
						t.Fatal(conf.Diagnostics)
					}
					var schema resource.SchemaResponse
					r.Schema(ctx, resource.SchemaRequest{}, &schema)
					plan := tfsdk.Plan{Schema: schema.Schema}
					if d := plan.Set(ctx, &snapshotResourceModel{Drive: types.StringValue("drive"), Name: types.StringValue("requested")}); d.HasError() {
						t.Fatal(d)
					}
					out := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
					if phase == "pre-cancel" {
						cancel()
					}
					if phase == "pre-deadline" {
						<-ctx.Done()
					}
					candidate := true
					// Pristine baseline panics on malformed acknowledgement; capture rather than conceal it.
					panicked := false
					func() {
						defer func() {
							if recover() != nil {
								panicked = true
							}
						}()
						r.Create(ctx, resource.CreateRequest{Plan: plan}, &out)
					}()
					currentOrder, _, _, currentReads, currentPosts, currentBlocked, _ := snapshotState()
					invalidAck := phase == "empty-ack" || phase == "null-ack" || phase == "invalid-ack"
					if invalidAck || phase == "pre-cancel" || phase == "pre-deadline" {
						if candidate && (panicked || !out.Diagnostics.HasError() || !out.State.Raw.IsNull()) {
							t.Fatal("invalid/unacknowledged state", panicked, out.Diagnostics, out.State.Raw)
						}
						if currentReads != 0 && candidate {
							t.Fatal("read after unusable acknowledgement", currentOrder)
						}
						if currentPosts > 1 {
							t.Fatal("retried POST")
						}
						t.Logf("RESULT boundary phase=%s panic=%v posts=%d reads=%d state=%s", phase, panicked, currentPosts, currentReads, out.State.Raw.String())
						return
					}
					if !candidate && phase == "final-null" {
						if !panicked || currentPosts != 1 || currentReads != 2 || !out.State.Raw.IsNull() {
							t.Fatal("baseline null final oracle", panicked, currentOrder, out.State.Raw)
						}
						t.Log("RESULT baseline final-null panic absent-state")
						return
					}
					if panicked {
						t.Fatal("unexpected callback panic")
					}
					if currentBlocked {
						select {
						case <-bodyDone:
						case <-time.After(time.Second):
							t.Fatal("body goroutine not terminal")
						}
					}
					if raw && bits < 3 {
						if !out.Diagnostics.HasError() || len(currentOrder) != 0 || !out.State.Raw.IsNull() {
							t.Fatal("SDK incomplete credential oracle", out.Diagnostics, currentOrder, out.State.Raw)
						}
						t.Log("RESULT raw-SDK-credential-refusal zero-dispatch")
						return
					}
					useNative := candidate && bits == 3
					partial := strings.HasSuffix(phase, "partial")
					wantErr := phase != "success" && phase != "pending" && phase != "deadline-policy" && !(partial && !useNative)
					if out.Diagnostics.HasError() != wantErr {
						t.Fatalf("diagnostic oracle wantError=%v actual=%v order=%v", wantErr, out.Diagnostics, currentOrder)
					}
					wantReads := 2
					if phase == "pending" {
						wantReads = 3
					}
					if phase == "unknown-post" || strings.HasPrefix(phase, "ack-") {
						wantReads = 0
					}
					if (strings.HasPrefix(phase, "wait-") || strings.HasPrefix(phase, "private-")) && wantErr {
						wantReads = 1
					}
					want := []string{fmt.Sprintf("%t:GET:/api/2.0/drives/drive/", useNative), fmt.Sprintf("%t:POST:/api/2.0/snapshots/", useNative)}
					for i := 0; i < wantReads; i++ {
						want = append(want, fmt.Sprintf("%t:GET:/api/2.0/snapshots/created/", useNative))
					}
					if phase == "private-id" && candidate {
						want = want[:2]
						wantReads = 0
					}
					if phase == "private-id" {
						for i := range want {
							want[i] = strings.ReplaceAll(want[i], "/created/", "/PRIVATE_ID_CANARY/")
						}
					}
					if !reflect.DeepEqual(currentOrder, want) || currentPosts != 1 || currentReads != wantReads {
						t.Fatalf("wire oracle got=%v want=%v posts=%d reads=%d", currentOrder, want, currentPosts, currentReads)
					}
					if wantErr {
						recoverable := candidate && phase != "unknown-post" && phase != "private-id" && !(useNative && strings.HasPrefix(phase, "ack-"))
						if recoverable {
							var got snapshotResourceModel
							if d := out.State.Get(context.Background(), &got); d.HasError() {
								t.Fatal(d)
							}
							expected := snapshotResourceModel{Drive: types.StringValue("drive"), Name: types.StringValue("requested"), ID: types.StringValue("created"), UUID: types.StringValue("created"), ResourceURI: types.StringNull(), Status: types.StringNull(), Timestamp: types.StringNull()}
							if !reflect.DeepEqual(got, expected) {
								t.Fatalf("recovery state got=%+v want=%+v", got, expected)
							}
							if !out.State.Raw.IsFullyKnown() {
								t.Fatal("unknown values cannot persist")
							}
							// Actual ordinary Read reconciles the saved identity without any mutation.
							stateMu.Lock()
							reconciling = true
							stateMu.Unlock()
							refreshed := resource.ReadResponse{State: out.State}
							r.Read(context.Background(), resource.ReadRequest{State: out.State}, &refreshed)
							if refreshed.Diagnostics.HasError() {
								t.Fatal(refreshed.Diagnostics)
							}
							currentOrder, _, _, _, currentPosts, _, _ = snapshotState()
							if d := refreshed.State.Get(context.Background(), &got); d.HasError() || got.ID.ValueString() != "created" || got.Name.ValueString() != "done" || got.Status.ValueString() != "available" || currentPosts != 1 {
								t.Fatal("refresh failed", got, d, currentOrder)
							}
						} else if !out.State.Raw.IsNull() {
							t.Fatal("fabricated state", out.State.Raw)
						}
					} else {
						var got snapshotResourceModel
						if d := out.State.Get(context.Background(), &got); d.HasError() {
							t.Fatal(d)
						}
						expected := snapshotResourceModel{Drive: types.StringValue("drive"), Name: types.StringValue("done"), ID: types.StringValue("created"), UUID: types.StringValue("created"), ResourceURI: types.StringValue("uri"), Status: types.StringValue("available"), Timestamp: types.StringValue("time")}
						if !reflect.DeepEqual(got, expected) {
							t.Fatalf("full state oracle got=%+v want=%+v", got, expected)
						}
					}
					if strings.HasPrefix(phase, "private-") {
						t.Logf("PRIVACY phase=%s exact=%v", phase, out.Diagnostics)
					}
					currentOrder, currentNative, currentLegacy, _, currentPosts, currentBlocked, _ := snapshotState()
					record := map[string]interface{}{"side": "candidate", "bits": bits, "raw": raw, "phase": phase, "order": currentOrder, "native": currentNative, "legacy": currentLegacy, "posts": currentPosts, "diagnostics": fmt.Sprint(out.Diagnostics), "state": out.State.Raw.String(), "blocked": currentBlocked}
					b, _ := json.Marshal(record)
					t.Log("RESULT " + string(b))
				})
			}
		}
	}
}

// Test-only reflection replaces the private HTTP client without exporting a
// production injection API. The real constructor and request/decode paths run.
func n28Client(t *testing.T, rt http.RoundTripper) *cloudsigmaapi.Client {
	t.Helper()
	c, err := cloudsigmaapi.New("zrh", "u", "p", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	field := reflect.ValueOf(c).Elem().FieldByName("transport")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(&http.Client{Transport: rt, Timeout: 60 * time.Second}))
	return c
}
