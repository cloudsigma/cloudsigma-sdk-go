package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CapabilitiesRaw preserves the complete dynamic public capabilities object.
// Raw values retain numeric lexemes and distinguish missing fields from null.
// This representation does not validate limits or infer feature availability.
type CapabilitiesRaw map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (c *CapabilitiesRaw) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("capabilities response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*c = fields
	return nil
}

// GetRaw reads the complete GET /capabilities/ response without query parameters.
// Unlike Get's historical hosts/hypervisors projection, it retains documented
// drive, snapshot and other dynamic entries. Each call owns independent values.
// It performs no limit arithmetic, caching, pagination or mutations. Existing
// Client.Do status, error and single-JSON-decode semantics apply.
// Docs: https://docs.cloudsigma.com/en/latest/capabilities.html
func (s *CapabilitiesService) GetRaw(ctx context.Context) (*CapabilitiesRaw, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "capabilities/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(CapabilitiesRaw)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
