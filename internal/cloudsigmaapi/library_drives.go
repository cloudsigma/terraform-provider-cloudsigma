package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"regexp"
)

type LibraryDrive struct {
	AllowMultimount   bool                   `json:"allow_multimount,omitempty"`
	Arch              string                 `json:"arch,omitempty"`
	Category          []string               `json:"category,omitempty"`
	CloudInitService  string                 `json:"cloud_init_service,omitempty"`
	CreatedAt         string                 `json:"created_at,omitempty"`
	DefaultAuthMethod string                 `json:"default_auth_method,omitempty"`
	DefaultPassword   string                 `json:"default_pass,omitempty"`
	DefaultUser       string                 `json:"default_user,omitempty"`
	Deprecated        bool                   `json:"deprecated,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Distribution      string                 `json:"distribution,omitempty"`
	Favourite         bool                   `json:"favourite,omitempty"`
	ImageType         string                 `json:"image_type,omitempty"`
	InstallNotes      string                 `json:"install_notes,omitempty"`
	Licenses          []DriveLicense         `json:"licenses,omitempty"`
	Media             string                 `json:"media,omitempty"`
	Meta              map[string]interface{} `json:"meta,omitempty"`
	MountedOn         []ResourceLink         `json:"mounted_on,omitempty"`
	Name              string                 `json:"name,omitempty"`
	OS                string                 `json:"os,omitempty"`
	Paid              bool                   `json:"paid,omitempty"`
	RemoteSnapshots   []ResourceLink         `json:"remote_snapshots,omitempty"`
	ResourceURI       string                 `json:"resource_uri,omitempty"`
	Size              int                    `json:"size,omitempty"`
	Status            string                 `json:"status,omitempty"`
	StorageType       string                 `json:"storage_type,omitempty"`
	Tags              []Tag                  `json:"tags,omitempty"`
	URL               string                 `json:"url,omitempty"`
	UUID              string                 `json:"uuid"`
	Version           string                 `json:"version,omitempty"`
}

type DriveLicense struct {
	Amount  int           `json:"amount,omitempty"`
	License *License      `json:"license,omitempty"`
	User    *ResourceLink `json:"user,omitempty"`
}

type ResourceLink struct {
	ResourceURI string `json:"resource_uri,omitempty"`
	UUID        string `json:"uuid,omitempty"`
}
type Tag struct {
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Owner       *ResourceLink          `json:"owner,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Resources   []TagResource          `json:"resources,omitempty"`
	UUID        string                 `json:"uuid,omitempty"`
}

// TagResource represents a resource assigned to the tag.
type TagResource struct {
	Owner        *ResourceLink `json:"owner,omitempty"`
	ResourceType string        `json:"res_type,omitempty"`
	ResourceURI  string        `json:"resource_uri,omitempty"`
	UUID         string        `json:"uuid,omitempty"`
}

type driveLicenseResponse DriveLicense

func (r *driveLicenseResponse) UnmarshalJSON(data []byte) error {
	type alias DriveLicense
	decoded := alias(*r)
	var nested *licenseResponse
	if decoded.License != nil {
		value := licenseResponse(*decoded.License)
		nested = &value
	}
	wire := struct {
		*alias
		License *licenseResponse `json:"license"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.License = (*License)(wire.License)
	*r = driveLicenseResponse(decoded)
	return nil
}

type libraryDriveResponse LibraryDrive

func (r *libraryDriveResponse) UnmarshalJSON(data []byte) error {
	type alias LibraryDrive
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
	*r = libraryDriveResponse(decoded)
	return nil
}

var libraryDriveSegment = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// LibraryDrives preserves the registered datasource's explicit limit=0 request.
func (c *Client) LibraryDrives(ctx context.Context, name string) ([]LibraryDrive, error) {
	q := url.Values{"limit": []string{"0"}}
	if name != "" {
		q.Set("name", name)
	}
	v, e := c.getLibraryDrives(ctx, libraryDrivesEndpoint, "libdrives/?"+q.Encode())
	if e != nil {
		return nil, e
	}
	return v.([]LibraryDrive), nil
}
func (c *Client) LibraryDrive(ctx context.Context, uuid string) (*LibraryDrive, error) {
	if !libraryDriveSegment.MatchString(uuid) {
		return nil, &Error{Category: "transport"}
	}
	v, e := c.getLibraryDrives(ctx, libraryDriveEndpoint, "libdrives/"+uuid+"/")
	if e != nil {
		return nil, e
	}
	return v.(*LibraryDrive), nil
}
func decodeLibraryDrive(b []byte) (*LibraryDrive, error) {
	var v libraryDriveResponse
	if e := json.NewDecoder(bytes.NewReader(b)).Decode(&v); e != nil {
		return nil, e
	}
	return (*LibraryDrive)(&v), nil
}
func decodeLibraryDrives(b []byte) ([]LibraryDrive, error) {
	var root struct {
		Objects []libraryDriveResponse `json:"objects"`
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
	v := make([]LibraryDrive, len(root.Objects))
	for i := range v {
		v[i] = LibraryDrive(root.Objects[i])
	}
	return v, nil
}
