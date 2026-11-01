package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These helpers run inside the existing offline assignment test so the
// fail-closed CI selector and its terminal-event count remain unchanged.
func schedulerAssignmentFixture(t *testing.T, transport fixtureRT) (*schedulerAssignmentResource, schema.Schema, func()) {
	t.Helper()
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = transport
	native, err := cloudsigmaapi.NewToken("zrh", "fixture-token", "fixture-agent")
	if err != nil {
		t.Fatal(err)
	}
	r := &schedulerAssignmentResource{}
	var configured resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: &ProviderClients{Auth: authToken, Native: native}}, &configured)
	if configured.Diagnostics.HasError() {
		t.Fatal(configured.Diagnostics)
	}
	var response resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &response)
	return r, response.Schema, func() { http.DefaultClient.Transport = old }
}

func schedulerAssignmentState(t *testing.T, definition schema.Schema, data schedulerAssignmentModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: definition}
	if diags := state.Set(context.Background(), &data); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func schedulerAssignmentResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}
}

func schedulerAssignmentReplacementRegression(t *testing.T) {
	t.Helper()
	calls := 0
	r, definition, restore := schedulerAssignmentFixture(t, fixtureRT(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("changed-drive update must not dispatch")
	}))
	defer restore()
	prior := schedulerAssignmentState(t, definition, schedulerAssignmentModel{ID: types.StringValue("old-drive"), DriveUUID: types.StringValue("old-drive"), SchedulerUUID: types.StringValue("scheduler-1")})
	plan := tfsdk.Plan{Schema: definition}
	if diags := plan.Set(context.Background(), &schedulerAssignmentModel{ID: types.StringValue("old-drive"), DriveUUID: types.StringValue("new-drive"), SchedulerUUID: types.StringValue("scheduler-1")}); diags.HasError() {
		t.Fatal(diags)
	}
	attribute := definition.Attributes["drive_uuid"].(schema.StringAttribute)
	if len(attribute.PlanModifiers) == 0 {
		t.Fatal("changing drive_uuid must plan replacement")
	}
	requiresReplace := false
	for _, modifier := range attribute.PlanModifiers {
		var modified planmodifier.StringResponse
		modifier.PlanModifyString(context.Background(), planmodifier.StringRequest{State: prior, Plan: plan, StateValue: types.StringValue("old-drive"), PlanValue: types.StringValue("new-drive")}, &modified)
		if modified.Diagnostics.HasError() {
			t.Fatal(modified.Diagnostics)
		}
		requiresReplace = requiresReplace || modified.RequiresReplace
	}
	if !requiresReplace {
		t.Fatal("changed drive_uuid did not require replacement")
	}
	updated := resource.UpdateResponse{State: prior}
	r.Update(context.Background(), resource.UpdateRequest{State: prior, Plan: plan}, &updated)
	if !updated.Diagnostics.HasError() || calls != 0 || !updated.State.Raw.Equal(prior.Raw) {
		t.Fatalf("changed drive update: calls=%d diagnostics=%v state=%s", calls, updated.Diagnostics, updated.State.Raw)
	}
}

func schedulerAssignmentImportReadRegression(t *testing.T) {
	t.Helper()
	calls := []string{}
	body := `{"uuid":"drive-1","backup_scheduler":"/api/2.0/backupschedulers/scheduler-1/"}`
	r, definition, restore := schedulerAssignmentFixture(t, fixtureRT(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method+" "+req.URL.RequestURI())
		if req.Method != http.MethodGet || req.URL.RequestURI() != "/api/2.0/drives/drive-1/" {
			t.Fatalf("import/refresh mutated or used the wrong route: %s %s", req.Method, req.URL)
		}
		return schedulerAssignmentResponse(req, http.StatusOK, body), nil
	}))
	defer restore()
	imported := resource.ImportStateResponse{State: schedulerAssignmentState(t, definition, schedulerAssignmentModel{})}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "drive-1"}, &imported)
	if imported.Diagnostics.HasError() || len(calls) != 0 {
		t.Fatalf("import dispatched: calls=%v diagnostics=%v", calls, imported.Diagnostics)
	}
	read := resource.ReadResponse{State: imported.State}
	r.Read(context.Background(), resource.ReadRequest{State: imported.State}, &read)
	var actual schedulerAssignmentModel
	if diags := read.State.Get(context.Background(), &actual); diags.HasError() || read.Diagnostics.HasError() || actual.ID.ValueString() != "drive-1" || actual.DriveUUID.ValueString() != "drive-1" || actual.SchedulerUUID.ValueString() != "scheduler-1" {
		t.Fatalf("literal URI import read: model=%+v diagnostics=%v %v", actual, read.Diagnostics, diags)
	}
	body = `{"uuid":"drive-1","backup_scheduler":"/api/2.0/backupschedulers/scheduler-2/"}`
	drift := resource.ReadResponse{State: read.State}
	r.Read(context.Background(), resource.ReadRequest{State: read.State}, &drift)
	if diags := drift.State.Get(context.Background(), &actual); diags.HasError() || drift.Diagnostics.HasError() || actual.SchedulerUUID.ValueString() != "scheduler-2" || actual.ID.ValueString() != "drive-1" {
		t.Fatalf("literal URI drift read: model=%+v diagnostics=%v %v", actual, drift.Diagnostics, diags)
	}
	for _, invalid := range []string{`"/api/2.0/drives/scheduler-2/"`, `"/api/2.0/backupschedulers/../"`, `""`, `{}`, `{"uuid":"scheduler-2","resource_uri":"/api/2.0/backupschedulers/different/"}`} {
		body = `{"uuid":"drive-1","backup_scheduler":` + invalid + `}`
		failed := resource.ReadResponse{State: drift.State}
		r.Read(context.Background(), resource.ReadRequest{State: drift.State}, &failed)
		if !failed.Diagnostics.HasError() || !failed.State.Raw.Equal(drift.State.Raw) {
			t.Fatalf("malformed association %s removed/changed state: diagnostics=%v state=%s", invalid, failed.Diagnostics, failed.State.Raw)
		}
	}
}

func schedulerAssignmentDeleteRegression(t *testing.T) {
	t.Helper()
	for _, scenario := range []struct {
		name         string
		initial      int
		action       int
		actionErr    error
		final        int
		finalRef     string
		wantErr      bool
		wantRequests int
	}{
		{name: "unknown transport then absent", actionErr: errors.New("synthetic uncertain response"), final: 200, finalRef: "null", wantRequests: 3},
		{name: "unknown timeout then absent", actionErr: context.DeadlineExceeded, final: 200, finalRef: "null", wantRequests: 3},
		{name: "unknown transport then replacement", actionErr: errors.New("synthetic uncertain response"), final: 200, finalRef: `"/api/2.0/backupschedulers/replacement/"`, wantRequests: 3},
		{name: "unknown transport then forbidden", actionErr: errors.New("synthetic uncertain response"), final: 403, wantErr: true, wantRequests: 3},
		{name: "unknown transport still associated", actionErr: errors.New("synthetic uncertain response"), final: 200, finalRef: `"/api/2.0/backupschedulers/scheduler-1/"`, wantErr: true, wantRequests: 3},
		{name: "action forbidden", action: 403, wantErr: true, wantRequests: 2},
		{name: "initial read forbidden", initial: 403, wantErr: true, wantRequests: 1},
		{name: "accepted then drive deleted", action: 202, final: 404, wantRequests: 3},
		{name: "accepted then forbidden", action: 202, final: 403, wantErr: true, wantRequests: 3},
		{name: "absent action then drive deleted", action: 404, final: 404, wantRequests: 3},
	} {
		func() {
			calls := []string{}
			reads := 0
			r, definition, restore := schedulerAssignmentFixture(t, fixtureRT(func(req *http.Request) (*http.Response, error) {
				calls = append(calls, req.Method+" "+req.URL.RequestURI())
				if req.Method == http.MethodPost {
					if req.URL.RequestURI() != "/api/2.0/drives/drive-1/action/?do=remove_scheduler" {
						t.Fatalf("%s unexpected mutation: %s", scenario.name, req.URL)
					}
					if scenario.actionErr != nil {
						return nil, scenario.actionErr
					}
					return schedulerAssignmentResponse(req, scenario.action, `{"uuid":"drive-1","backup_scheduler":null}`), nil
				}
				if req.Method != http.MethodGet || req.URL.RequestURI() != "/api/2.0/drives/drive-1/" {
					t.Fatalf("%s unexpected read: %s %s", scenario.name, req.Method, req.URL)
				}
				reads++
				if reads == 1 {
					status := scenario.initial
					if status == 0 {
						status = http.StatusOK
					}
					return schedulerAssignmentResponse(req, status, `{"uuid":"drive-1","backup_scheduler":"/api/2.0/backupschedulers/scheduler-1/"}`), nil
				}
				return schedulerAssignmentResponse(req, scenario.final, `{"uuid":"drive-1","backup_scheduler":`+scenario.finalRef+`}`), nil
			}))
			defer restore()
			prior := schedulerAssignmentState(t, definition, schedulerAssignmentModel{ID: types.StringValue("drive-1"), DriveUUID: types.StringValue("drive-1"), SchedulerUUID: types.StringValue("scheduler-1")})
			deleted := resource.DeleteResponse{State: prior}
			r.Delete(context.Background(), resource.DeleteRequest{State: prior}, &deleted)
			if deleted.Diagnostics.HasError() != scenario.wantErr || len(calls) != scenario.wantRequests {
				t.Fatalf("%s calls=%v diagnostics=%v", scenario.name, calls, deleted.Diagnostics)
			}
			if scenario.wantErr && !deleted.State.Raw.Equal(prior.Raw) {
				t.Fatalf("%s changed state on failure", scenario.name)
			}
		}()
	}
}
