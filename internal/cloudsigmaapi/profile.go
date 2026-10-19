package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
)

// Profile is deliberately independent of Terraform state and legacy SDK models.
type Profile struct{ Address, Company, FirstName, LastName, Title, UUID string }

func (c *Client) Profile(ctx context.Context) (*Profile, error) {
	v, err := c.get(ctx, profileEndpoint)
	if err != nil {
		return nil, err
	}
	return v.(*Profile), nil
}
func decodeProfile(b []byte) (*Profile, error) {
	fail := func() (*Profile, error) { return nil, &Error{Category: "decode_error"} }
	d := json.NewDecoder(bytes.NewReader(b))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return fail()
	}
	p := &Profile{}
	fields := map[string]*string{"address": &p.Address, "company": &p.Company, "first_name": &p.FirstName, "last_name": &p.LastName, "title": &p.Title, "uuid": &p.UUID}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return fail()
		}
		k, ok := key.(string)
		if !ok || seen[k] {
			return fail()
		}
		seen[k] = true
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return fail()
		}
		if dest, ok := fields[k]; ok {
			if json.Unmarshal(raw, dest) != nil {
				return fail()
			}
		}
	}
	if t, err = d.Token(); err != nil || t != json.Delim('}') {
		return fail()
	}
	if _, err = d.Token(); err != io.EOF {
		return fail()
	}
	return p, nil
}
