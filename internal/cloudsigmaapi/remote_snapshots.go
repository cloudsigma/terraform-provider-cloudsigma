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

// RemoteSnapshotDriveRef is the minimal nested drive reference accepted by
// the remotesnapshots API. ResourceURI and response-only drive fields are
// deliberately not represented here.
type RemoteSnapshotDriveRef struct {
	UUID string `json:"uuid"`
}

// RemoteSnapshotCreate is the writable subset accepted by the remotesnapshots API.
// Drive may be a UUID string or RemoteSnapshotDriveRef. The request projector
// rejects every other representation so complete response objects cannot leak
// into a create request.
type RemoteSnapshotCreate struct {
	Drive    any
	Location string
}

type RemoteSnapshot struct {
	UUID        string `json:"uuid"`
	ResourceURI string `json:"resource_uri,omitempty"`
	Drive       *Drive `json:"drive,omitempty"`
	Location    string `json:"location,omitempty"`
	Status      string `json:"status,omitempty"`
}

// CreateRemoteSnapshot performs the API's single-object batch POST. It does not poll readiness.
func (c *Client) CreateRemoteSnapshot(ctx context.Context, input RemoteSnapshotCreate) (*RemoteSnapshot, error) {
	objects, err := c.createRemoteSnapshots(ctx, []RemoteSnapshotCreate{input}, true)
	if err != nil {
		return nil, err
	}
	return &objects[0], nil
}

// CreateRemoteSnapshots performs a batch POST and preserves the API response
// order. It does not poll readiness or infer completion from status.
func (c *Client) CreateRemoteSnapshots(ctx context.Context, inputs []RemoteSnapshotCreate) ([]RemoteSnapshot, error) {
	return c.createRemoteSnapshots(ctx, inputs, true)
}

type remoteSnapshotCreateWire struct {
	Drive    any    `json:"drive"`
	Location string `json:"location,omitempty"`
}

// marshalRemoteSnapshotRequest is kept separate from transport so both
// supported wire forms (a bare object and an objects batch) can be qualified
// without involving a live endpoint.
func marshalRemoteSnapshotRequest(inputs []RemoteSnapshotCreate, batch bool) ([]byte, error) {
	if len(inputs) == 0 || (!batch && len(inputs) != 1) {
		return nil, &Error{Category: "transport"}
	}
	wire := make([]remoteSnapshotCreateWire, len(inputs))
	for i, input := range inputs {
		drive, err := projectRemoteSnapshotDrive(input.Drive)
		if err != nil || (input.Location != "" && !locationLabel.MatchString(strings.ToLower(input.Location))) {
			return nil, &Error{Category: "transport"}
		}
		wire[i] = remoteSnapshotCreateWire{Drive: drive, Location: input.Location}
	}
	if batch {
		return json.Marshal(struct {
			Objects []remoteSnapshotCreateWire `json:"objects"`
		}{Objects: wire})
	}
	return json.Marshal(wire[0])
}

func projectRemoteSnapshotDrive(value any) (any, error) {
	switch drive := value.(type) {
	case string:
		if !driveSegment.MatchString(drive) {
			return nil, &Error{Category: "transport"}
		}
		return drive, nil
	case RemoteSnapshotDriveRef:
		if !driveSegment.MatchString(drive.UUID) {
			return nil, &Error{Category: "transport"}
		}
		return RemoteSnapshotDriveRef{UUID: drive.UUID}, nil
	case *RemoteSnapshotDriveRef:
		if drive == nil || !driveSegment.MatchString(drive.UUID) {
			return nil, &Error{Category: "transport"}
		}
		return RemoteSnapshotDriveRef{UUID: drive.UUID}, nil
	default:
		return nil, &Error{Category: "transport"}
	}
}

func (c *Client) createRemoteSnapshots(ctx context.Context, inputs []RemoteSnapshotCreate, batch bool) ([]RemoteSnapshot, error) {
	payload, err := marshalRemoteSnapshotRequest(inputs, batch)
	if err != nil {
		return nil, err
	}
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	if err := ctx.Err(); err != nil {
		return nil, remoteContextError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"remotesnapshots/", bytes.NewReader(payload))
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
	fail := func(cat string) ([]RemoteSnapshot, error) {
		return nil, &Error{Category: cat, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusCreated {
		cat := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			cat = "unexpected_status"
		}
		return fail(cat)
	}
	if enc := resp.Header.Get("Content-Encoding"); enc != "" && !strings.EqualFold(enc, "identity") {
		return fail("unsupported_encoding")
	}
	if resp.Body == nil {
		return fail("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fail(remoteContextError(ctx, ctxErr).(*Error).Category)
		}
		return fail("transport")
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fail(remoteContextError(ctx, ctxErr).(*Error).Category)
	}
	if len(body) > responseLimit {
		return fail("decode_error")
	}
	var doc struct {
		Objects []RemoteSnapshot `json:"objects"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&doc); err != nil {
		return fail("decode_error")
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return fail("decode_error")
	}
	if len(doc.Objects) != len(inputs) {
		return fail("decode_error")
	}
	for _, object := range doc.Objects {
		if !driveSegment.MatchString(object.UUID) {
			return fail("decode_error")
		}
	}
	return doc.Objects, nil
}

func remoteContextError(ctx context.Context, err error) error {
	cat := "transport"
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		cat = "cancelled"
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		cat = "timeout"
	}
	return &Error{Category: cat}
}
