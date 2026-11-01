package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// Snapshot is the read-only snapshot projection; it does not authorize writes.
type Snapshot struct {
	AllocatedSize int                    `json:"allocated_size,omitempty"`
	Drive         *Drive                 `json:"drive,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Owner         *ResourceLink          `json:"owner,omitempty"`
	ResourceURI   string                 `json:"resource_uri,omitempty"`
	Status        string                 `json:"status,omitempty"`
	Tags          []Tag                  `json:"tags"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	UUID          string                 `json:"uuid,omitempty"`
}

// SnapshotDriveRef is the minimal writable drive reference accepted by the
// snapshots API. Response-only drive fields are deliberately not represented.
type SnapshotDriveRef struct {
	UUID string `json:"uuid"`
}

// SnapshotCreateRequest is the provider-owned writable projection for a
// snapshot create. It intentionally excludes response-only fields (including
// status, timestamp, resource_uri, tags and drive metadata).
type SnapshotCreateRequest struct {
	Drive SnapshotDriveRef `json:"drive"`
	Name  string           `json:"name,omitempty"`
}

// SnapshotUpdateRequest is the provider-owned writable projection for a
// snapshot update. The API accepts the same minimal drive/name subset as
// create; the target UUID is carried in the URL, never in the body.
type SnapshotUpdateRequest struct {
	Drive SnapshotDriveRef `json:"drive"`
	Name  string           `json:"name,omitempty"`
}

type snapshotResponse Snapshot

func (r *snapshotResponse) UnmarshalJSON(data []byte) error {
	type alias Snapshot
	decoded := alias(*r)
	var nested *driveResponse
	if decoded.Drive != nil {
		value := driveResponse(*decoded.Drive)
		nested = &value
	}
	wire := struct {
		*alias
		Drive *driveResponse `json:"drive"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Drive = (*Drive)(wire.Drive)
	*r = snapshotResponse(decoded)
	return nil
}

func (c *Client) Snapshot(ctx context.Context, uuid string) (*Snapshot, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrives(ctx, snapshotEndpoint, "snapshots/"+uuid+"/")
	if err != nil {
		return nil, err
	}
	return v.(*Snapshot), nil
}

// SnapshotWithCallerContext reads lifecycle state without imposing a per-read
// deadline. Other Snapshot readers retain their existing bounded policy.
func (c *Client) SnapshotWithCallerContext(ctx context.Context, uuid string) (*Snapshot, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrivesContext(ctx, snapshotEndpoint, "snapshots/"+uuid+"/", false)
	if err != nil {
		return nil, err
	}
	return v.(*Snapshot), nil
}

// CreateSnapshot performs the documented snapshot POST without using the SDK
// request model. The API contract requires HTTP 201; all other statuses,
// including other 2xx responses, are rejected. No readiness polling is
// performed here.
func (c *Client) CreateSnapshot(ctx context.Context, input SnapshotCreateRequest) (*Snapshot, error) {
	if !driveSegment.MatchString(input.Drive.UUID) {
		return nil, &Error{Category: "transport"}
	}
	payload, err := json.Marshal(struct {
		Objects []SnapshotCreateRequest `json:"objects"`
	}{Objects: []SnapshotCreateRequest{input}})
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	return c.snapshotWrite(ctx, http.MethodPost, "snapshots/", payload, true, http.StatusCreated)
}

// UpdateSnapshot performs a minimal PUT for an existing snapshot. The
// response identity must match the requested UUID before it is admitted.
func (c *Client) UpdateSnapshot(ctx context.Context, uuid string, input SnapshotUpdateRequest) (*Snapshot, error) {
	if !driveSegment.MatchString(uuid) || !driveSegment.MatchString(input.Drive.UUID) {
		return nil, &Error{Category: "transport"}
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	snapshot, err := c.snapshotWrite(ctx, http.MethodPut, "snapshots/"+uuid+"/", payload, false, http.StatusOK)
	if err != nil {
		return nil, err
	}
	if snapshot.UUID != uuid {
		return nil, &Error{Category: "decode_error", Status: http.StatusOK}
	}
	return snapshot, nil
}

// snapshotWrite is the bounded transport used by native snapshot mutations.
// It admits exactly one complete JSON document and never follows redirects or
// retries a request whose outcome is unknown.
func (c *Client) snapshotWrite(ctx context.Context, method, route string, payload []byte, envelope bool, statuses ...int) (*Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	if err := ctx.Err(); err != nil {
		return nil, remoteContextError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+route, bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", c.userAgent)
	c.authenticate(req)
	resp, err := transport.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, remoteContextError(ctx, err)
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	fail := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	allowed := false
	for _, status := range statuses {
		if resp.StatusCode == status {
			allowed = true
			break
		}
	}
	if !allowed {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return nil, fail(category)
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return nil, fail("unsupported_encoding")
	}
	if resp.Body == nil {
		return nil, fail("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, remoteContextError(ctx, ctxErr)
		}
		return nil, fail("transport")
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, remoteContextError(ctx, ctxErr)
	}
	if len(body) > responseLimit {
		return nil, fail("response_too_large")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var document json.RawMessage
	if err := decoder.Decode(&document); err != nil {
		return nil, fail("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fail("decode_error")
	}
	var snapshot *Snapshot
	if envelope {
		var root struct {
			Objects []snapshotResponse `json:"objects"`
		}
		if err := json.Unmarshal(document, &root); err != nil || len(root.Objects) != 1 {
			return nil, fail("decode_error")
		}
		snapshot = (*Snapshot)(&root.Objects[0])
	} else {
		var decoded snapshotResponse
		if err := json.Unmarshal(document, &decoded); err != nil {
			return nil, fail("decode_error")
		}
		snapshot = (*Snapshot)(&decoded)
	}
	if snapshot == nil || !driveSegment.MatchString(snapshot.UUID) {
		return nil, fail("decode_error")
	}
	if err := ctx.Err(); err != nil {
		category := "cancelled"
		if errors.Is(err, context.DeadlineExceeded) {
			category = "timeout"
		}
		return nil, fail(category)
	}
	return snapshot, nil
}

func decodeSnapshot(b []byte) (*Snapshot, error) {
	var v snapshotResponse
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&v); err != nil {
		return nil, err
	}
	return (*Snapshot)(&v), nil
}
