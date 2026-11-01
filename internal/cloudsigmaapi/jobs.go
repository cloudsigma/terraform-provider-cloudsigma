package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
)

// Job is the read-only asynchronous-operation model documented by CloudSigma.
// Data and nested children remain dynamic because deployments may add fields.
type Job struct {
	Children     []Job                      `json:"children,omitempty"`
	Created      string                     `json:"created,omitempty"`
	Data         map[string]json.RawMessage `json:"data,omitempty"`
	LastModified string                     `json:"last_modified,omitempty"`
	Operation    string                     `json:"operation,omitempty"`
	ResourceURI  string                     `json:"resource_uri,omitempty"`
	Resources    []string                   `json:"resources,omitempty"`
	State        string                     `json:"state,omitempty"`
	UUID         string                     `json:"uuid"`
}

// Jobs requests all jobs using limit=0 and refuses an incomplete response.
// The response-size limit still applies; callers may need pagination support.
func (c *Client) Jobs(ctx context.Context) ([]Job, error) {
	v, err := c.getLibraryDrives(ctx, jobsEndpoint, "jobs/?limit=0")
	if err != nil {
		return nil, err
	}
	return v.([]Job), nil
}

// Job reads one job by UUID without polling or interpreting terminal state.
func (c *Client) Job(ctx context.Context, uuid string) (*Job, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrives(ctx, jobEndpoint, "jobs/"+uuid+"/")
	if err != nil {
		return nil, err
	}
	job := v.(*Job)
	if job.UUID != uuid {
		return nil, &Error{Category: "decode_error"}
	}
	return job, nil
}

func decodeJobs(body []byte) ([]Job, error) {
	var root struct {
		Objects []Job `json:"objects"`
		Meta    *struct {
			Offset     *int `json:"offset"`
			TotalCount *int `json:"total_count"`
		} `json:"meta"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&root); err != nil || root.Objects == nil {
		return nil, &Error{Category: "decode_error"}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, &Error{Category: "decode_error"}
	}
	if root.Meta == nil || root.Meta.Offset == nil || root.Meta.TotalCount == nil || *root.Meta.Offset != 0 || *root.Meta.TotalCount != len(root.Objects) {
		return nil, &Error{Category: "incomplete_response"}
	}
	for _, job := range root.Objects {
		if !driveSegment.MatchString(job.UUID) {
			return nil, &Error{Category: "decode_error"}
		}
	}
	return root.Objects, nil
}

func decodeJob(body []byte) (*Job, error) {
	var job Job
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&job); err != nil || !driveSegment.MatchString(job.UUID) {
		return nil, &Error{Category: "decode_error"}
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, &Error{Category: "decode_error"}
	}
	return &job, nil
}
