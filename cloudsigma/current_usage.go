package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CurrentUsageService provides the public current usage read operation.
// Docs: https://docs.cloudsigma.com/en/latest/current_usage.html
// It does not expose mutations or calculate billing or resource usage.
type CurrentUsageService service

// CurrentUsage is a raw JSON object preserving the complete response, including
// nested balance and usage dictionaries, decimal strings, numeric lexemes and
// unknown fields. No schema or billing semantics are inferred from raw values.
// Each call returns independent data.
type CurrentUsage map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *CurrentUsage) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("currentusage response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// GetRaw reads GET /currentusage/ without a query or request body. The complete
// object is retained without typed projection or float conversion. No filters,
// pagination, metering calculations or schema validation are performed.
// Existing Client.Do decoding/error rules apply.
func (s *CurrentUsageService) GetRaw(ctx context.Context) (*CurrentUsage, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "currentusage/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(CurrentUsage)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
