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

func TestOfflineBearerNativeSnapshotCreate(t *testing.T) {
	ctx := context.Background()
	var requestBody map[string]any
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "fixture")
	if err != nil {
		t.Fatal(err)
	}
	nativeTransport := fixtureRT(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/2.0/snapshots/" {
			t.Fatalf("request=%s %s", req.Method, req.URL)
		}
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(req.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"created","drive":{"uuid":"drive"},"name":"requested"}]}`)), Request: req}, nil
	})
	field := reflect.ValueOf(native).Elem().FieldByName("transport")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(&http.Client{Transport: nativeTransport, Timeout: time.Minute}))
	legacyTransport := fixtureRT(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatalf("legacy authorization=%q", req.Header.Get("Authorization"))
		}
		body := `{"uuid":"drive","status":"unmounted"}`
		if strings.Contains(req.URL.Path, "/snapshots/") {
			body = `{"uuid":"created","drive":{"uuid":"drive"},"name":"requested","status":"available","timestamp":"time","resource_uri":"uri"}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	legacy := cs.NewClient(cs.NewTokenCredentialsProvider("fixture-token"), cs.WithHTTPClient(&http.Client{Transport: legacyTransport}))
	r := &snapshotResource{}
	var configured resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Legacy: legacy, Native: native}}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	plan := tfsdk.Plan{Schema: schemaResponse.Schema}
	if diags := plan.Set(ctx, &snapshotResourceModel{Drive: types.StringValue("drive"), Name: types.StringValue("requested")}); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	want := map[string]any{"objects": []any{map[string]any{"drive": map[string]any{"uuid": "drive"}, "name": "requested"}}}
	if !reflect.DeepEqual(requestBody, want) {
		t.Fatalf("native create projection=%#v", requestBody)
	}
	var got snapshotResourceModel
	if diags := response.State.Get(ctx, &got); diags.HasError() || got.ID.ValueString() != "created" || got.Name.ValueString() != "requested" || got.Status.ValueString() != "available" {
		t.Fatalf("state=%+v diagnostics=%v", got, diags)
	}
}
