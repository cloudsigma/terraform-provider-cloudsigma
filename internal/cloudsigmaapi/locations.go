package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
)

// Location preserves all legacy filterable fields independently of the SDK.
type Location struct {
	AlternativeFrontendURL   string `json:"alternative_frontend_url,omitempty"`
	APIEndpoint              string `json:"api_endpoint,omitempty"`
	CountryCode              string `json:"country_code,omitempty"`
	DefaultFrontendSignupURL string `json:"default_frontend_signup_url,omitempty"`
	DefaultFrontendURL       string `json:"default_frontend_url,omitempty"`
	DisplayName              string `json:"display_name,omitempty"`
	DocumentationURL         string `json:"documentation_url,omitempty"`
	ID                       string `json:"id,omitempty"`
	UploadURL                string `json:"upload_url,omitempty"`
	WebsocketURL             string `json:"websocket_url,omitempty"`
}

func (c *Client) Locations(ctx context.Context) ([]Location, error) {
	v, err := c.get(ctx, locationsEndpoint)
	if err != nil {
		return nil, err
	}
	return v.([]Location), nil
}
func decodeLocations(b []byte) ([]Location, error) {
	var root struct {
		Objects []Location `json:"objects"`
		Meta    *struct {
			Limit      int `json:"limit"`
			Offset     int `json:"offset"`
			TotalCount int `json:"total_count"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&root); err != nil {
		return nil, err
	}
	return root.Objects, nil
}
