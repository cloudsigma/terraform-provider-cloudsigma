package provider

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/terraform"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Error bodies and resource IDs are intentionally excluded from diagnostics.
func requireDestroyed(r *cs.Response, err error) error {
	if err != nil && r != nil && r.StatusCode == http.StatusNotFound {
		return nil
	}
	return fmt.Errorf("resource absence not confirmed by HTTP 404")
}
func TestOfflineDestroyStatus(t *testing.T) {
	for _, code := range []int{200, 403, 404, 500} {
		r := &cs.Response{Response: &http.Response{StatusCode: code}}
		err := requireDestroyed(r, fmt.Errorf("PRIVATE-CANARY"))
		if (err == nil) != (code == 404) {
			t.Fatal(code)
		}
	}
	if requireDestroyed(nil, fmt.Errorf("transport")) == nil {
		t.Fatal("transport accepted")
	}
}

func TestOfflineSnapshotDependencyCleanup(t *testing.T) {
	t.Setenv("CLOUDSIGMA_LOCATION", "next")
	t.Setenv("CLOUDSIGMA_USERNAME", "synthetic")
	t.Setenv("CLOUDSIGMA_PASSWORD", "synthetic")
	for _, driveCode := range []int{404, 403, 500, 200} {
		old := http.DefaultClient
		paths := []string{}
		http.DefaultClient = &http.Client{Transport: fixtureRT(func(q *http.Request) (*http.Response, error) {
			paths = append(paths, q.URL.Path)
			code := 404
			if strings.Contains(q.URL.Path, "drives") {
				code = driveCode
			}
			body := `[]`
			if code == 200 {
				body = `{"uuid":"fixture"}`
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}
		state := &terraform.State{Modules: []*terraform.ModuleState{{Path: []string{"root"}, Resources: map[string]*terraform.ResourceState{"snapshot": {Type: "cloudsigma_snapshot", Primary: &terraform.InstanceState{ID: "fixture"}}, "drive": {Type: "cloudsigma_drive", Primary: &terraform.InstanceState{ID: "fixture"}}}}}}
		err := testAccCheckSnapshotDestroy(state)
		http.DefaultClient = old
		if (err == nil) != (driveCode == 404) {
			t.Fatalf("drive status %d result %v", driveCode, err)
		}
		if len(paths) != 2 {
			t.Fatalf("dependency not checked: %v", paths)
		}
	}
}
