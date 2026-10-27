package provider

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func profileFixture(t *testing.T, code int, body string, calls *int, auth string) *cloudsigmaapi.Client {
	t.Helper()
	old := http.DefaultTransport
	tr := old.(*http.Transport).Clone()
	tr.Proxy = nil
	tr.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() {
			defer b.Close()
			r, err := http.ReadRequest(bufio.NewReader(b))
			if err != nil {
				return
			}
			*calls++
			if r.Method != http.MethodGet || r.URL.Path != "/api/2.0/profile/" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != auth {
				return
			}
			_, _ = fmt.Fprintf(b, "HTTP/1.1 %d response\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", code, len(body), body)
		}()
		return a, nil
	}
	tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	http.DefaultTransport = tr
	c, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "fixture-agent")
	http.DefaultTransport = old
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestOfflineNativeTokenProfile(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
		want string
	}{
		{name: "success", code: http.StatusOK, body: `{"address":"a","company":"c","first_name":"f","last_name":"l","title":"t","uuid":"profile"}`},
		{name: "status", code: http.StatusUnauthorized, body: `PRIVATE`, want: "Unable to fetch user profile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			native := profileFixture(t, tc.code, tc.body, &calls, "Bearer fixture-token")
			d := &profileDataSource{}
			ctx := context.Background()
			var conf datasource.ConfigureResponse
			d.Configure(ctx, datasource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native, Legacy: nil}}, &conf)
			if conf.Diagnostics.HasError() {
				t.Fatal(conf.Diagnostics)
			}
			var s datasource.SchemaResponse
			d.Schema(ctx, datasource.SchemaRequest{}, &s)
			state := tfsdk.State{Schema: s.Schema}
			prior := profileDataSourceModel{ID: types.StringValue("old"), UUID: types.StringValue("old")}
			if diags := state.Set(ctx, &prior); diags.HasError() {
				t.Fatal(diags)
			}
			out := datasource.ReadResponse{State: state}
			d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
			if tc.want != "" {
				if !out.Diagnostics.HasError() || out.Diagnostics[0].Summary() != tc.want || calls != 1 || !out.State.Raw.Equal(state.Raw) {
					t.Fatalf("diagnostics=%v calls=%d", out.Diagnostics, calls)
				}
				return
			}
			var got profileDataSourceModel
			if diags := out.State.Get(ctx, &got); diags.HasError() || out.Diagnostics.HasError() || got.ID.ValueString() != "profile" || got.Address.ValueString() != "a" {
				t.Fatalf("state=%+v diagnostics=%v", got, out.Diagnostics)
			}
			if calls != 1 {
				t.Fatalf("calls=%d", calls)
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "fixture-agent")
	if err != nil {
		t.Fatal(err)
	}
	d := &profileDataSource{clients: &ProviderClients{Auth: authToken, Native: native}}
	var s datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &s)
	state := tfsdk.State{Schema: s.Schema}
	if diags := state.Set(context.Background(), &profileDataSourceModel{ID: types.StringValue("old")}); diags.HasError() {
		t.Fatal(diags)
	}
	out := datasource.ReadResponse{State: state}
	d.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{Schema: s.Schema, Raw: state.Raw}}, &out)
	if !out.Diagnostics.HasError() || !strings.Contains(out.Diagnostics[0].Detail(), "cancelled") || !out.State.Raw.Equal(state.Raw) {
		t.Fatal(errors.New("token profile cancellation"))
	}
}
