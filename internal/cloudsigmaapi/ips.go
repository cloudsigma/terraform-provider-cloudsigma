package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"unicode"
)

type IP struct {
	Gateway     string                 `json:"gateway,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Nameservers []string               `json:"nameservers,omitempty"`
	Netmask     int                    `json:"netmask,omitempty"`
	Owner       *ResourceLink          `json:"owner,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Server      *ResourceLink          `json:"server,omitempty"`
	UUID        string                 `json:"uuid,omitempty"`
}

// safeIPSegment is route-integrity policy, not validation of an IP address family.
// Preserve opaque identifiers (including colon syntax) without interpreting them.
func safeIPSegment(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	for _, r := range id {
		if r > 127 || unicode.IsSpace(r) || unicode.IsControl(r) || strings.ContainsRune("/%?#\\", r) {
			return false
		}
	}
	return true
}
func (c *Client) IPs(ctx context.Context) ([]IP, error) {
	v, e := c.getLibraryDrives(ctx, ipsEndpoint, "ips/detail/")
	if e != nil {
		return nil, e
	}
	return v.([]IP), nil
}
func (c *Client) IP(ctx context.Context, id string) (*IP, error) {
	if !safeIPSegment(id) {
		return nil, &Error{Category: "transport"}
	}
	v, e := c.getLibraryDrives(ctx, ipEndpoint, "ips/"+id+"/")
	if e != nil {
		return nil, e
	}
	return v.(*IP), nil
}
func decodeIP(b []byte) (*IP, error) {
	var v IP
	e := json.NewDecoder(bytes.NewReader(b)).Decode(&v)
	return &v, e
}
func decodeIPs(b []byte) ([]IP, error) {
	var root struct {
		Objects []IP `json:"objects"`
		Meta    *struct {
			Limit      int `json:"limit"`
			Offset     int `json:"offset"`
			TotalCount int `json:"total_count"`
		} `json:"meta"`
	}
	e := json.NewDecoder(bytes.NewReader(b)).Decode(&root)
	return root.Objects, e
}
