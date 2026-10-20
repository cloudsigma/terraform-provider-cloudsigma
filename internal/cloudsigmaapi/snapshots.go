package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
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

func decodeSnapshot(b []byte) (*Snapshot, error) {
	var v snapshotResponse
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&v); err != nil {
		return nil, err
	}
	return (*Snapshot)(&v), nil
}
