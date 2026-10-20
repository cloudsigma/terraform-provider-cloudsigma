package cloudsigma

import "errors"

// serverReadMetadata validates the complete dynamic projection before any state
// writes. Empty containers retain prior metadata, as in the legacy read path.
func serverReadMetadata(values map[string]interface{}) (map[string]interface{}, error) {
	metadata := make(map[string]interface{}, len(values))
	for key, value := range values {
		// This field is managed separately by ssh_keys, not Terraform metadata.
		if key == "ssh_public_key" {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, errors.New("invalid server metadata response: expected string values")
		}
		metadata[key] = text
	}
	return metadata, nil
}
