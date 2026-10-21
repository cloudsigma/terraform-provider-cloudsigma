package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/provider/drive"
	fp "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Same portable assertions run on the pinned baseline with routing explicitly
// selected by N29_BASELINE. No polling clocks or waiter durations are replaced.
func TestNative29DriveReadiness(t *testing.T) {
	baseline := os.Getenv("N29_BASELINE") == "1"
	for bits := 0; bits < 8; bits++ {
		phases := []string{"success"}
		if bits == 3 {
			phases = []string{"success", "pending", "deadline", "cancel", "404", "malformed", "null", "partial", "privateprecision", "unavailable", "provided-deadline"}
		}
		for _, phase := range phases {
			t.Run(fmt.Sprintf("configured%d/%s", bits, phase), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if phase == "deadline" || phase == "provided-deadline" {
					var end context.CancelFunc
					ctx, end = context.WithTimeout(ctx, 7*time.Second)
					defer end()
				}
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
				order := []string{}
				dr, posts, sr, nc, lc := 0, 0, 0, 0, 0
				done := make(chan struct{}, 1)
				blocked := false
				transport := func(native bool) n28RT {
					return func(req *http.Request) (*http.Response, error) {
						if native {
							nc++
						} else {
							lc++
						}
						order = append(order, fmt.Sprintf("%t:%s:%s", native, req.Method, req.URL.Path))
						text := `{"uuid":"drive","status":"unmounted"}`
						code := 200
						var body io.ReadCloser
						if strings.Contains(req.URL.Path, "drives/") {
							dr++
							want, has := ctx.Deadline()
							got, ok := req.Context().Deadline()
							if has != ok || has && !got.Equal(want) {
								t.Fatal("caller deadline altered", has, ok, want, got)
							}
							if phase == "pending" && dr == 1 {
								text = `{"status":"creating"}`
							}
							if phase == "404" {
								code = 404
								text = `{}`
							}
							if phase == "malformed" {
								text = `{"PRIVATE_DRIVE_CANARY":`
							}
							if phase == "null" {
								text = `null`
							}
							if phase == "unavailable" {
								text = `{"status":"unavailable"}`
							}
							if phase == "privateprecision" {
								text = `{"uuid":"drive","status":"mounted","size":9007199254740993,"meta":{"private":"PRIVATE_DRIVE_CANARY","number":9007199254740993}}`
							}
							if phase == "partial" {
								body = &n28Partial{text: text}
							}
							if phase == "deadline" || phase == "cancel" {
								blocked = true
								reader, writer := io.Pipe()
								body = reader
								go func() {
									if phase == "cancel" {
										time.Sleep(20 * time.Millisecond)
										cancel()
									}
									<-ctx.Done()
									_ = writer.CloseWithError(ctx.Err())
									done <- struct{}{}
								}()
							}
						} else {
							text = `{"uuid":"created","drive":{"uuid":"drive"},"name":"done","status":"available","resource_uri":"uri","timestamp":"time"}`
							if req.Method == "POST" {
								posts++
								if native {
									t.Fatal("native mutation")
								}
								text = `{"objects":[` + text + `]}`
							} else {
								sr++
							}
						}
						if body == nil {
							body = io.NopCloser(strings.NewReader(text))
						}
						return &http.Response{StatusCode: code, Header: make(http.Header), Body: body, Request: req}, nil
					}
				}
				old := http.DefaultClient
				http.DefaultClient = &http.Client{Transport: transport(false)}
				defer func() { http.DefaultClient = old }()
				provider := New("n29")()
				var ps fp.SchemaResponse
				provider.Schema(ctx, fp.SchemaRequest{}, &ps)
				typ := ps.Schema.Type().TerraformType(ctx)
				attrs := map[string]tftypes.Value{}
				for k, v := range typ.(tftypes.Object).AttributeTypes {
					attrs[k] = tftypes.NewValue(v, nil)
				}
				for k, v := range map[string]string{"location": "zrh", "username": u, "password": p, "token": tok} {
					attrs[k] = tftypes.NewValue(tftypes.String, v)
				}
				var configured fp.ConfigureResponse
				provider.Configure(ctx, fp.ConfigureRequest{Config: tfsdk.Config{Schema: ps.Schema, Raw: tftypes.NewValue(typ, attrs)}}, &configured)
				valid := bits == 3 || bits == 4
				if configured.Diagnostics.HasError() == valid {
					t.Fatal(configured.Diagnostics)
				}
				if !valid {
					if nc+lc != 0 || configured.ResourceData != nil {
						t.Fatal("invalid credentials dispatched")
					}
					return
				}
				clients := configured.ResourceData.(*ProviderClients)
				if bits == 3 {
					clients.Native = n28Client(t, transport(true))
				}
				r := &snapshotResource{}
				var conf resource.ConfigureResponse
				r.Configure(ctx, resource.ConfigureRequest{ProviderData: clients}, &conf)
				if conf.Diagnostics.HasError() {
					t.Fatal(conf.Diagnostics)
				}
				var sc resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &sc)
				plan := tfsdk.Plan{Schema: sc.Schema}
				if d := plan.Set(ctx, &snapshotResourceModel{Drive: types.StringValue("drive"), Name: types.StringValue("requested")}); d.HasError() {
					t.Fatal(d)
				}
				out := resource.CreateResponse{State: tfsdk.State{Schema: sc.Schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, &out)
				failure := phase == "deadline" || phase == "cancel" || phase == "404" || phase == "malformed" || phase == "null" || phase == "unavailable" || phase == "partial" && !baseline
				if out.Diagnostics.HasError() != failure {
					t.Fatal("diagnostics", out.Diagnostics, order)
				}
				if blocked {
					select {
					case <-done:
					case <-time.After(time.Second):
						t.Fatal("body not terminal")
					}
				}
				driveNative := bits == 3 && !baseline
				want := []string{fmt.Sprintf("%t:GET:/api/2.0/drives/drive/", driveNative)}
				if phase == "pending" {
					want = append(want, want[0])
				}
				if !failure {
					want = append(want, "false:POST:/api/2.0/snapshots/", fmt.Sprintf("%t:GET:/api/2.0/snapshots/created/", bits == 3), fmt.Sprintf("%t:GET:/api/2.0/snapshots/created/", bits == 3))
				}
				if !reflect.DeepEqual(order, want) || nc+lc != len(want) {
					t.Fatal("wire order", order, want)
				}
				if failure {
					if posts != 0 || sr != 0 || !out.State.Raw.IsNull() {
						t.Fatal("created snapshot after preflight failure", posts, sr, out.State.Raw)
					}
				} else {
					var got snapshotResourceModel
					if d := out.State.Get(ctx, &got); d.HasError() || got.ID.ValueString() != "created" || got.UUID.ValueString() != "created" || got.Status.ValueString() != "available" || posts != 1 || sr != 2 {
						t.Fatal(got, d, order)
					}
				}
				if strings.Contains(fmt.Sprint(out.Diagnostics), "PRIVATE_DRIVE_CANARY") || strings.Contains(out.State.Raw.String(), "PRIVATE_DRIVE_CANARY") {
					t.Fatal("private drive field leaked")
				}
			})
		}
	}
	// Raw helper deliberately receives no Native argument; token takes precedence.
	for bits := 0; bits < 8; bits++ {
		t.Run(fmt.Sprintf("raw%d", bits), func(t *testing.T) {
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
			var creds cs.CredentialsProvider = cs.NewUsernamePasswordCredentialsProvider(u, p)
			if tok != "" {
				creds = cs.NewTokenCredentialsProvider(tok)
			}
			calls := 0
			c := cs.NewClient(creds, cs.WithHTTPClient(&http.Client{Transport: n28RT(func(req *http.Request) (*http.Response, error) {
				calls++
				if _, ok := req.Context().Deadline(); ok {
					t.Fatal("raw deadline added")
				}
				if tok != "" && !strings.Contains(req.Header.Get("Authorization"), tok) {
					t.Fatal("token precedence")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"status":"unmounted"}`)), Request: req}, nil
			})}))
			err := drive.WaitDriveStatusMountedOrUnmounted(context.Background(), c, "drive")
			valid := bits == 3 || bits >= 4
			if (err == nil) != valid || valid && calls != 1 || !valid && calls != 0 {
				t.Fatal(err, calls)
			}
		})
	}
}
