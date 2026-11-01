package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func snapshotUpdateNativeFixture(t *testing.T, code int, body string, requestBody *map[string]any) *cloudsigmaapi.Client {
	t.Helper()
	native, err := cloudsigmaapi.New("zrh", "fixture", "fixture", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	transport := fixtureRT(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPut || req.URL.Path != "/api/2.0/snapshots/prior/" {
			t.Fatalf("request=%s %s", req.Method, req.URL)
		}
		if requestBody != nil {
			if err := json.NewDecoder(req.Body).Decode(requestBody); err != nil {
				t.Fatal(err)
			}
		}
		return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	field := reflect.ValueOf(native).Elem().FieldByName("transport")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(&http.Client{Transport: transport, Timeout: time.Minute}))
	return native
}

func TestOfflineNativeSnapshotUpdate(t *testing.T) {
	ctx := context.Background()
	var requestBody map[string]any
	native := snapshotUpdateNativeFixture(t, http.StatusOK, `{"uuid":"prior","drive":{"uuid":"drive"},"name":"updated","status":"available","resource_uri":"uri","timestamp":"time","drive_meta":{"private":"ignored"}}`, &requestBody)
	legacy := cs.NewClient(cs.NewTokenCredentialsProvider("legacy"))
	r := &snapshotResource{}
	var configured resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authBasic, Legacy: legacy, Native: native}}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	prior := snapshotResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("prior"), Drive: types.StringValue("drive"), Name: types.StringValue("old")}
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diags := plan.Set(ctx, &snapshotResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("prior"), Drive: types.StringValue("drive"), Name: types.StringValue("updated")}); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if requestBody["drive_meta"] != nil || requestBody["drive"].(map[string]any)["uuid"] != "drive" || requestBody["name"] != "updated" {
		t.Fatalf("native update projection=%#v", requestBody)
	}
	var got snapshotResourceModel
	if diags := response.State.Get(ctx, &got); diags.HasError() || got.ID.ValueString() != "prior" || got.Name.ValueString() != "updated" || got.Status.ValueString() != "available" {
		t.Fatalf("state=%+v diagnostics=%v", got, diags)
	}
}

func TestOfflineNativeSnapshotUpdateErrorPreservesState(t *testing.T) {
	ctx := context.Background()
	native := snapshotUpdateNativeFixture(t, http.StatusForbidden, `{}`, nil)
	r := &snapshotResource{native: native}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	prior := snapshotResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("prior"), Drive: types.StringValue("drive"), Name: types.StringValue("old")}
	state := tfsdk.State{Schema: schemaResponse.Schema}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diags := plan.Set(ctx, &snapshotResourceModel{ID: types.StringValue("prior"), UUID: types.StringValue("prior"), Drive: types.StringValue("drive"), Name: types.StringValue("updated")}); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &response)
	if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) {
		t.Fatalf("diagnostics=%v state=%s prior=%s", response.Diagnostics, response.State.Raw, state.Raw)
	}
}
