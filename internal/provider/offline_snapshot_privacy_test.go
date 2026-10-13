package provider

import (
	"bytes"
	"context"
	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Every operation dispatches only to an in-memory transport. No snapshot exists.
func TestOfflineSnapshotPrivacy(t *testing.T) {
	for _, op := range []string{"create", "read", "update", "delete"} {
		t.Run(op, func(t *testing.T) {
			var logs bytes.Buffer
			ctx := tflogtest.RootLogger(context.Background(), &logs)
			calls := 0
			r := &snapshotResource{client: cs.NewClient(cs.NewTokenCredentialsProvider("PRIVATE-CANARY"), cs.WithLocation("next"), cs.WithHTTPClient(&http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
				calls++
				if q.URL.Host != "next.cloudsigma.com" {
					t.Fatal("origin")
				}
				body := `{"uuid":"fixture","drive":{"uuid":"drive"},"name":"PRIVATE-CANARY","status":"available"}`
				code := 200
				switch {
				case op == "create" && q.Method == "GET" && q.URL.Path == "/api/2.0/drives/drive/":
					body = `{"uuid":"drive","status":"unmounted"}`
				case op == "create" && q.Method == "POST" && q.URL.Path == "/api/2.0/snapshots/":
					body = `{"objects":[` + body + `]}`
					code = 201
				case q.Method == "GET" && q.URL.Path == "/api/2.0/snapshots/fixture/":
				case op == "update" && q.Method == "PUT" && q.URL.Path == "/api/2.0/snapshots/fixture/":
				case op == "delete" && q.Method == "DELETE" && q.URL.Path == "/api/2.0/snapshots/fixture/":
					body = ""
					code = 204
				default:
					t.Fatalf("unexpected wire %s %s", q.Method, q.URL.Path)
				}
				return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
			})}))}
			var sc resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &sc)
			model := snapshotResourceModel{ID: types.StringValue("fixture"), UUID: types.StringValue("fixture"), Drive: types.StringValue("drive"), Name: types.StringValue("PRIVATE-CANARY")}
			state := tfsdk.State{Schema: sc.Schema}
			if d := state.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			plan := tfsdk.Plan{Schema: sc.Schema, Raw: state.Raw}
			switch op {
			case "create":
				out := resource.CreateResponse{State: tfsdk.State{Schema: sc.Schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, &out)
				if out.Diagnostics.HasError() {
					t.Fatal(out.Diagnostics)
				}
			case "read":
				out := resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, &out)
				if out.Diagnostics.HasError() {
					t.Fatal(out.Diagnostics)
				}
			case "update":
				out := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{Plan: plan}, &out)
				if out.Diagnostics.HasError() {
					t.Fatal(out.Diagnostics)
				}
			case "delete":
				out := resource.DeleteResponse{State: state}
				r.Delete(ctx, resource.DeleteRequest{State: state}, &out)
				if out.Diagnostics.HasError() {
					t.Fatal(out.Diagnostics)
				}
			}
			want := 1
			if op == "create" {
				want = 4
			}
			if calls != want {
				t.Fatalf("calls %d want %d", calls, want)
			}
			if strings.Contains(logs.String(), "PRIVATE-CANARY") {
				t.Error("snapshot private data logged")
			}
		})
	}
}
