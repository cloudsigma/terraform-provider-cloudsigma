package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// License preserves all legacy filterable fields independently of the SDK.
type License struct {
	Burstable                     bool   `json:"burstable,omitempty"`
	Free                          string `json:"free,omitempty"`
	InitialPurchaseRequirement    string `json:"initial_purchase_requirement,omitempty"`
	LongName                      string `json:"long_name,omitempty"`
	Name                          string `json:"name,omitempty"`
	ResourceURI                   string `json:"resource_uri,omitempty"`
	SubsequentPurchaseRequirement string `json:"subsequent_purchase_requirement,omitempty"`
	Type                          string `json:"type,omitempty"`
	UserMetric                    string `json:"user_metric"`
}

func (c *Client) Licenses(ctx context.Context) ([]License, error) {
	v, err := c.get(ctx, licensesEndpoint)
	if err != nil {
		return nil, err
	}
	return v.([]License), nil
}
func decodeLicenses(b []byte) ([]License, error) {
	var root struct {
		Objects []licenseResponse `json:"objects"`
		Meta    *struct {
			Limit      int `json:"limit"`
			Offset     int `json:"offset"`
			TotalCount int `json:"total_count"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&root); err != nil {
		return nil, err
	}
	if root.Objects == nil {
		return nil, nil
	}
	result := make([]License, len(root.Objects))
	for i := range result {
		result[i] = License(root.Objects[i])
	}
	return result, nil
}

// Response-only numeric quantity compatibility preserves the pinned SDK wire semantics.
type licenseResponse License

// licenseQuantity decodes each occurrence immediately so duplicate keys cannot
// hide invalid values and null preserves the most recently decoded value.
type licenseQuantity struct {
	name  string
	value string
}

func (q *licenseQuantity) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &q.value); err != nil {
			return fmt.Errorf("license.%s: %w", q.name, err)
		}
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("license.%s: %w", q.name, err)
	}
	q.value = number.String()
	return nil
}

func (l *licenseResponse) UnmarshalJSON(data []byte) error {
	type licenseAlias License
	decoded := licenseAlias(*l)
	wire := struct {
		*licenseAlias
		Free       licenseQuantity `json:"free"`
		Initial    licenseQuantity `json:"initial_purchase_requirement"`
		Subsequent licenseQuantity `json:"subsequent_purchase_requirement"`
	}{
		licenseAlias: &decoded,
		Free:         licenseQuantity{"free", l.Free},
		Initial:      licenseQuantity{"initial_purchase_requirement", l.InitialPurchaseRequirement},
		Subsequent:   licenseQuantity{"subsequent_purchase_requirement", l.SubsequentPurchaseRequirement},
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Free = wire.Free.value
	decoded.InitialPurchaseRequirement = wire.Initial.value
	decoded.SubsequentPurchaseRequirement = wire.Subsequent.value
	*l = licenseResponse(decoded)
	return nil
}
