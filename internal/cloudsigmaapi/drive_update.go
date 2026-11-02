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

// DriveUpdateRequest contains the documented writable drive definition fields.
// Response-only identity, ownership, runtime, status, attachment and snapshot
// projections are deliberately not representable here.
type DriveUpdateRequest struct {
	// AllowMultimount is a pointer so callers can distinguish omitted from an
	// explicit false value. This matters when clearing an existing setting.
	AllowMultimount *bool                  `json:"allow_multimount,omitempty"`
	Media           string                 `json:"media,omitempty"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Size            int                    `json:"size,omitempty"`
	StorageType     string                 `json:"storage_type,omitempty"`
	Tags            []Tag                  `json:"tags,omitempty"`
}

// MarshalJSON preserves explicit empty maps/slices for clearing metadata and
// tags while omitting nil values. The API treats these as distinct operations.
func (r DriveUpdateRequest) MarshalJSON() ([]byte, error) {
	type wire struct {
		AllowMultimount *bool                   `json:"allow_multimount,omitempty"`
		Media           string                  `json:"media,omitempty"`
		Meta            *map[string]interface{} `json:"meta,omitempty"`
		Name            string                  `json:"name,omitempty"`
		Size            int                     `json:"size,omitempty"`
		StorageType     string                  `json:"storage_type,omitempty"`
		Tags            *[]Tag                  `json:"tags,omitempty"`
	}
	out := wire{AllowMultimount: r.AllowMultimount, Media: r.Media, Name: r.Name, Size: r.Size, StorageType: r.StorageType}
	if r.Meta != nil {
		out.Meta = &r.Meta
	}
	if r.Tags != nil {
		out.Tags = &r.Tags
	}
	return json.Marshal(out)
}

// UpdateDrive performs one documented ordinary drive PUT. It does not invoke
// the resize action, stop/start servers, poll status, or retry an uncertain
// request. Those lifecycle semantics remain owned by the existing provider.
func (c *Client) UpdateDrive(ctx context.Context, uuid string, input DriveUpdateRequest) (*Drive, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, &Error{Category: "transport"}
	}
	transport := c.transport
	if h, ok := transport.(*http.Client); ok {
		clone := *h
		clone.Timeout = 0
		transport = &clone
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, driveUpdateContextError(ctx, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.base+"drives/"+uuid+"/", bytes.NewReader(payload))
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
		return nil, driveUpdateContextError(ctx, err)
	}
	if resp == nil {
		return nil, &Error{Category: "transport"}
	}
	failure := func(category string) error {
		return &Error{Category: category, Status: resp.StatusCode, RequestID: requestID(resp.Header.Get("X-REQUEST-ID"))}
	}
	if resp.StatusCode != http.StatusOK {
		category := "http_status"
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			category = "unexpected_status"
		}
		return nil, failure(category)
	}
	if encoding := resp.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		return nil, failure("unsupported_encoding")
	}
	if resp.Body == nil {
		return nil, failure("decode_error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, responseLimit+1))
	if err != nil {
		return nil, driveUpdateContextError(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, driveUpdateContextError(ctx, err)
	}
	if len(body) > responseLimit {
		return nil, failure("response_too_large")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var document json.RawMessage
	if err := decoder.Decode(&document); err != nil {
		return nil, failure("decode_error")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, failure("decode_error")
	}
	drive, err := decodeDrive(document)
	if err != nil || drive == nil || !driveSegment.MatchString(drive.UUID) || drive.UUID != uuid {
		return nil, failure("decode_error")
	}
	return drive, nil
}

func driveUpdateContextError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return &Error{Category: "cancelled"}
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return &Error{Category: "timeout"}
	}
	return &Error{Category: "transport"}
}
