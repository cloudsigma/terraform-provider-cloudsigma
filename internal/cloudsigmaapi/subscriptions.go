package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
)

// Subscription preserves all legacy filterable fields independently of the SDK.
type Subscription struct {
	Amount           string `json:"amount,omitempty"`
	AutoRenew        bool   `json:"auto_renew,omitempty"`
	EndTime          string `json:"end_time,omitempty"`
	FreeTier         bool   `json:"free_tier,omitempty"`
	ID               string `json:"id,omitempty"`
	Period           string `json:"period,omitempty"`
	Price            string `json:"price,omitempty"`
	Remaining        string `json:"remaining,omitempty"`
	Resource         string `json:"resource,omitempty"`
	ResourceURI      string `json:"resource_uri,omitempty"`
	StartTime        string `json:"start_time,omitempty"`
	Status           string `json:"status,omitempty"`
	SubscribedObject string `json:"subscribed_object,omitempty"`
	UUID             string `json:"uuid,omitempty"`
}

func (c *Client) Subscriptions(ctx context.Context) ([]Subscription, error) {
	v, err := c.get(ctx, subscriptionsEndpoint)
	if err != nil {
		return nil, err
	}
	return v.([]Subscription), nil
}

// Decode only the existing single response; do not invent paging or numeric coercion.
func decodeSubscriptions(b []byte) ([]Subscription, error) {
	var root struct {
		Objects []Subscription `json:"objects"`
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
