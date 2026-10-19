package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
)

type VLAN struct {
	Meta         map[string]interface{} `json:"meta,omitempty"`
	Owner        *ResourceLink          `json:"owner,omitempty"`
	ResourceURI  string                 `json:"resource_uri,omitempty"`
	Servers      []ResourceLink         `json:"servers,omitempty"`
	Subscription *VLANSubscription      `json:"subscription,omitempty"`
	Tags         []Tag                  `json:"tags,omitempty"`
	UUID         string                 `json:"uuid,omitempty"`
}

// VLANSubscription represents a CloudSigma subscription reference.
type VLANSubscription struct {
	ID          int    `json:"id"`
	ResourceURI string `json:"resource_uri,omitempty"`
}

// vlanSegment is local route-integrity policy, not a public UUID grammar.
var vlanSegment = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

func (c *Client) VLANs(ctx context.Context) ([]VLAN, error) {
	v, e := c.getLibraryDrives(ctx, vlansEndpoint, "vlans/detail/")
	if e != nil {
		return nil, e
	}
	return v.([]VLAN), nil
}
func (c *Client) VLAN(ctx context.Context, id string) (*VLAN, error) {
	if !vlanSegment.MatchString(id) {
		return nil, &Error{Category: "transport"}
	}
	v, e := c.getLibraryDrives(ctx, vlanEndpoint, "vlans/"+id+"/")
	if e != nil {
		return nil, e
	}
	return v.(*VLAN), nil
}
func decodeVLAN(b []byte) (*VLAN, error) {
	var v VLAN
	e := json.NewDecoder(bytes.NewReader(b)).Decode(&v)
	return &v, e
}
func decodeVLANs(b []byte) ([]VLAN, error) {
	var root struct {
		Objects []VLAN `json:"objects"`
		Meta    *struct {
			Limit      int `json:"limit"`
			Offset     int `json:"offset"`
			TotalCount int `json:"total_count"`
		} `json:"meta"`
	}
	e := json.NewDecoder(bytes.NewReader(b)).Decode(&root)
	return root.Objects, e
}
