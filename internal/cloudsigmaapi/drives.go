package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"regexp"
)

type Drive struct {
	AllowMultimount bool                   `json:"allow_multimount,omitempty"`
	Licenses        []DriveLicense         `json:"licenses,omitempty"`
	Media           string                 `json:"media,omitempty"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
	MountedOn       []ResourceLink         `json:"mounted_on,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Owner           *ResourceLink          `json:"owner,omitempty"`
	RemoteSnapshots []ResourceLink         `json:"remote_snapshots,omitempty"`
	ResourceURI     string                 `json:"resource_uri,omitempty"`
	Runtime         *DriveRuntime          `json:"runtime,omitempty"`
	Size            int                    `json:"size,omitempty"`
	Snapshots       []ResourceLink         `json:"snapshots,omitempty"`
	Status          string                 `json:"status,omitempty"`
	StorageType     string                 `json:"storage_type,omitempty"`
	Tags            []Tag                  `json:"tags,omitempty"`
	UUID            string                 `json:"uuid,omitempty"`
}

// DriveRuntime represents a CloudSigma runtime information of the drive.
type DriveRuntime struct {
	IsSnapshotable         bool   `json:"is_snapshotable,omitempty"`
	SnapshotsAllocatedSize int    `json:"snapshots_allocated_size,omitempty"`
	StorageType            string `json:"storage_type,omitempty"`
}

type driveResponse Drive

func (r *driveResponse) UnmarshalJSON(data []byte) error {
	type alias Drive
	decoded := alias(*r)
	var nested []driveLicenseResponse
	if decoded.Licenses != nil {
		nested = make([]driveLicenseResponse, len(decoded.Licenses))
		for i := range nested {
			nested[i] = driveLicenseResponse(decoded.Licenses[i])
		}
	}
	wire := struct {
		*alias
		Licenses []driveLicenseResponse `json:"licenses"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Licenses = nil
	if wire.Licenses != nil {
		decoded.Licenses = make([]DriveLicense, len(wire.Licenses))
		for i := range wire.Licenses {
			decoded.Licenses[i] = DriveLicense(wire.Licenses[i])
		}
	}
	*r = driveResponse(decoded)
	return nil
}

var driveSegment = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// Drives preserves the registered datasource's explicit limit=0 request.
func (c *Client) Drives(ctx context.Context, name string) ([]Drive, error) {
	q := url.Values{"limit": []string{"0"}}
	if name != "" {
		q.Set("name", name)
	}
	v, e := c.getLibraryDrives(ctx, drivesEndpoint, "drives/detail/?"+q.Encode())
	if e != nil {
		return nil, e
	}
	return v.([]Drive), nil
}
func (c *Client) Drive(ctx context.Context, uuid string) (*Drive, error) {
	if !driveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, e := c.getLibraryDrives(ctx, driveEndpoint, "drives/"+uuid+"/")
	if e != nil {
		return nil, e
	}
	return v.(*Drive), nil
}
func decodeDrive(b []byte) (*Drive, error) {
	var v driveResponse
	if e := json.NewDecoder(bytes.NewReader(b)).Decode(&v); e != nil {
		return nil, e
	}
	return (*Drive)(&v), nil
}
func decodeDrives(b []byte) ([]Drive, error) {
	var root struct {
		Objects []driveResponse `json:"objects"`
		Meta    *struct {
			Limit      int `json:"limit"`
			Offset     int `json:"offset"`
			TotalCount int `json:"total_count"`
		} `json:"meta"`
	}
	if e := json.NewDecoder(bytes.NewReader(b)).Decode(&root); e != nil {
		return nil, e
	}
	if root.Objects == nil {
		return nil, nil
	}
	v := make([]Drive, len(root.Objects))
	for i := range v {
		v[i] = Drive(root.Objects[i])
	}
	return v, nil
}

// DriveWithCallerContext reads a drive for lifecycle status waiters without
// introducing a per-read timeout. It does not poll or interpret completion.
func (c *Client) DriveWithCallerContext(ctx context.Context, id string) (*Drive, error) {
	if !driveSegment.MatchString(id) {
		return nil, &Error{Category: "transport"}
	}
	v, err := c.getLibraryDrivesContext(ctx, driveEndpoint, "drives/"+id+"/", false)
	if err != nil {
		return nil, err
	}
	return v.(*Drive), nil
}
