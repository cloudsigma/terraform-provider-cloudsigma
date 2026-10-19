package cloudsigmaapi

import "fmt"

// Error contains only safe diagnostic metadata, never response or credential values.
type Error struct {
	Category  string
	Status    int
	RequestID string
}

func (e *Error) Error() string {
	s := e.Category
	if e.Status != 0 {
		s += fmt.Sprintf(" status=%d", e.Status)
	}
	if e.RequestID != "" {
		s += " request_id=" + e.RequestID
	}
	return s
}
func requestID(s string) string {
	if len(s) > 128 {
		return ""
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return ""
		}
	}
	return s
}
