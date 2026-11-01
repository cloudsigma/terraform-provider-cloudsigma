package cloudsigmaapi

import (
	"context"
	"net/http"
	"testing"
)

func TestJobsReadsCollectionAndDetail(t *testing.T) {
	c := remoteSnapshotClient(func(req *http.Request) (*http.Response, error) {
		if user, password, ok := req.BasicAuth(); !ok || user != "fixture-user" || password != "fixture-password" {
			t.Fatalf("authorization=%q", req.Header.Get("Authorization"))
		}
		switch req.URL.RequestURI() {
		case "/api/2.0/jobs/?limit=0":
			return remoteSnapshotResponse(http.StatusOK, `{"meta":{"offset":0,"total_count":1},"objects":[{"uuid":"job-1","state":"success","data":{"progress":100},"children":[{"uuid":"child-1","state":"success"}]}]}`), nil
		case "/api/2.0/jobs/job-1/":
			return remoteSnapshotResponse(http.StatusOK, `{"uuid":"job-1","operation":"drive_clone","state":"success","resources":["/api/2.0/drives/d-1/"]}`), nil
		default:
			t.Fatalf("route=%s", req.URL.RequestURI())
			return nil, nil
		}
	})
	jobs, err := c.Jobs(context.Background())
	if err != nil || len(jobs) != 1 || jobs[0].UUID != "job-1" || len(jobs[0].Children) != 1 || jobs[0].Children[0].UUID != "child-1" {
		t.Fatalf("jobs=%+v err=%v", jobs, err)
	}
	job, err := c.Job(context.Background(), "job-1")
	if err != nil || job.UUID != "job-1" || job.State != "success" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}

func TestJobsRejectMalformedResponsesAndIDs(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "null collection", body: `{"objects":null}`},
		{name: "missing collection", body: `{}`},
		{name: "invalid collection uuid", body: `{"meta":{"offset":0,"total_count":1},"objects":[{"uuid":"bad/id"}]}`},
		{name: "server capped collection", body: `{"meta":{"offset":0,"total_count":2},"objects":[{"uuid":"job-1"}]}`},
		{name: "trailing document", body: `{"meta":{"offset":0,"total_count":0},"objects":[]} {}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) {
				return remoteSnapshotResponse(http.StatusOK, tc.body), nil
			})
			if _, err := c.Jobs(context.Background()); err == nil {
				t.Fatal("malformed collection accepted")
			}
		})
	}
	c := remoteSnapshotClient(func(*http.Request) (*http.Response, error) { t.Fatal("invalid ID reached transport"); return nil, nil })
	if _, err := c.Job(context.Background(), "bad/id"); err == nil {
		t.Fatal("invalid job ID accepted")
	}
}
