package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func schedulerDefinitionFixture() string {
	return `{"name":"terraform-snapshot","type":"snapshot","is_enabled":false,"is_default":false,"manual_incremental_backup":"","user_timezone":"Etc/UTC","orchestrator_timezone":"","incremental_backup":{"minute":"","hour":"","day_of_month":"","month":"","day_of_week":[""],"start_time":{"hour":0,"minute":0},"end_time":{"hour":23,"minute":59},"repeat":{"hour":"","minute":"*/30"}},"initial_backup":{"start_time":"","end_time":"","day_of_week":""},"retention_policy":{"name":"keep-three","rules":[{"period":"items","quantity":3}]},"enable_backup_policy_notifications":false,"enable_retention_policy_notifications":false,"remote_location":"","meta":{},"tags":[]}`
}

func TestOfflineSchedulerResourceCreateReadUpdateDelete(t *testing.T) {
	calls := []string{}
	old := http.DefaultTransport
	oldClientTransport := http.DefaultClient.Transport
	http.DefaultTransport = fixtureRT(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		var body string
		switch req.Method {
		case http.MethodPost:
			body = `{"objects":[{"uuid":"scheduler-1","name":"terraform-snapshot","type":"snapshot"}]}`
			return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
		case http.MethodPut:
			body = `{"uuid":"scheduler-1","name":"terraform-snapshot","type":"snapshot"}`
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
		case http.MethodDelete:
			return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
		default:
			body = `{"uuid":"scheduler-1","name":"terraform-snapshot","type":"snapshot","owner":{"uuid":"server"},"permissions":[],"resource_uri":"/api/2.0/backupschedulers/scheduler-1/"}`
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
		}
	})
	http.DefaultClient.Transport = http.DefaultTransport
	defer func() { http.DefaultTransport = old; http.DefaultClient.Transport = oldClientTransport }()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "fixture-agent")
	if err != nil {
		t.Fatal(err)
	}
	r := NewSchedulerResource()
	var conf resource.ConfigureResponse
	r.(resource.ResourceWithConfigure).Configure(context.Background(), resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native}}, &conf)
	if conf.Diagnostics.HasError() {
		t.Fatal(conf.Diagnostics)
	}
	var sr resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &sr)
	plan := tfsdk.Plan{Schema: sr.Schema}
	if d := plan.Set(context.Background(), &schedulerResourceModel{DefinitionJSON: types.StringValue(schedulerDefinitionFixture())}); d.HasError() {
		t.Fatal(d)
	}
	create := resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &create)
	if create.Diagnostics.HasError() {
		t.Fatal(create.Diagnostics)
	}
	var state schedulerResourceModel
	if d := create.State.Get(context.Background(), &state); d.HasError() || state.ID.ValueString() != "scheduler-1" {
		t.Fatal(state, d)
	}
	read := resource.ReadResponse{State: create.State}
	r.Read(context.Background(), resource.ReadRequest{State: create.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	update := resource.UpdateResponse{State: read.State}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: read.State}, &update)
	if update.Diagnostics.HasError() {
		t.Fatal(update.Diagnostics, calls)
	}
	del := resource.DeleteResponse{State: update.State}
	r.Delete(context.Background(), resource.DeleteRequest{State: update.State}, &del)
	if del.Diagnostics.HasError() {
		t.Fatal(del.Diagnostics, calls)
	}
	if len(calls) != 4 || calls[0] != "POST /api/2.0/backupschedulers/" || calls[1] != "GET /api/2.0/backupschedulers/scheduler-1/" || calls[2] != "PUT /api/2.0/backupschedulers/scheduler-1/" || calls[3] != "DELETE /api/2.0/backupschedulers/scheduler-1/" {
		t.Fatalf("calls=%v", calls)
	}
}

func TestOfflineSchedulerAssignmentReconcilesAndDoesNotBlindlyReplay(t *testing.T) {
	calls := []string{}
	old := http.DefaultTransport
	oldClientTransport := http.DefaultClient.Transport
	http.DefaultTransport = fixtureRT(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method+" "+req.URL.RequestURI())
		if req.Method == http.MethodGet {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"drive-1","backup_scheduler":{"uuid":"scheduler-1"}}`)), Request: req}, nil
		}
		return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"drive-1","backup_scheduler":{"uuid":"scheduler-1"}}`)), Request: req}, nil
	})
	http.DefaultClient.Transport = http.DefaultTransport
	defer func() { http.DefaultTransport = old; http.DefaultClient.Transport = oldClientTransport }()
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "fixture-agent")
	if err != nil {
		t.Fatal(err)
	}
	r := NewSchedulerAssignmentResource()
	var conf resource.ConfigureResponse
	r.(resource.ResourceWithConfigure).Configure(context.Background(), resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native}}, &conf)
	if conf.Diagnostics.HasError() {
		t.Fatal(conf.Diagnostics)
	}
	var sr resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &sr)
	plan := tfsdk.Plan{Schema: sr.Schema}
	if d := plan.Set(context.Background(), &schedulerAssignmentModel{DriveUUID: types.StringValue("drive-1"), SchedulerUUID: types.StringValue("scheduler-1")}); d.HasError() {
		t.Fatal(d)
	}
	out := resource.CreateResponse{State: tfsdk.State{Schema: sr.Schema}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &out)
	if out.Diagnostics.HasError() {
		t.Fatal(out.Diagnostics)
	}
	// Existing association converges without replaying the action.
	if len(calls) != 1 || !strings.HasPrefix(calls[0], "GET ") {
		t.Fatalf("calls=%v", calls)
	}
	var state schedulerAssignmentModel
	if d := out.State.Get(context.Background(), &state); d.HasError() || state.ID.ValueString() != "drive-1" {
		t.Fatal(state, d)
	}
}
