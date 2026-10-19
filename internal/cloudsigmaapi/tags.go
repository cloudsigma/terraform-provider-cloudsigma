package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
)

// Tags preserves the existing first-page tags/ request without adding query parameters.
func (c *Client) Tags(ctx context.Context) ([]Tag, error) {
	v, e := c.get(ctx, tagsEndpoint)
	if e != nil {
		return nil, e
	}
	return v.([]Tag), nil
}
func decodeTags(b []byte) ([]Tag, error) {
	var root struct {
		Objects []Tag `json:"objects"`
		Meta    *struct {
			Limit      int `json:"limit"`
			Offset     int `json:"offset"`
			TotalCount int `json:"total_count"`
		} `json:"meta"`
	}
	e := json.NewDecoder(bytes.NewReader(b)).Decode(&root)
	return root.Objects, e
}
