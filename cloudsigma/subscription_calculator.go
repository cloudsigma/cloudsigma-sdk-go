package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Calculate returns a quote for the supplied subscription definitions, without
// purchasing subscriptions. The documented request is an objects envelope.
// Each definition must be a non-null JSON object; fields are forwarded unchanged,
// not checked for server writability, resource applicability or valid durations.
// Nil/empty batches are refused as SDK policy. Callers must not mutate inputs
// concurrently. The complete response object preserves the aggregate price and
// per-object quotes without numeric conversion or local price calculation.
// This performs one POST, with no retry or polling. Shared HTTP success/error
// rules apply; raw preservation is not full billing-schema qualification.
// Docs: https://docs.cloudsigma.com/en/latest/subscriptions.html#calculator
func (s *SubscriptionsService) Calculate(ctx context.Context, definitions []json.RawMessage) (map[string]json.RawMessage, *Response, error) {
	if len(definitions) == 0 {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	objects := make([]json.RawMessage, len(definitions))
	for i, definition := range definitions {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(definition, &object); err != nil || object == nil {
			return nil, nil, fmt.Errorf("subscription quote definition must be a JSON object")
		}
		objects[i] = append(json.RawMessage(nil), definition...)
	}
	body := struct {
		Objects []json.RawMessage `json:"objects"`
	}{objects}
	req, err := s.client.NewRequest(http.MethodPost, "subscriptioncalculator/", body)
	if err != nil {
		return nil, nil, err
	}
	var out map[string]json.RawMessage
	resp, err := s.client.Do(ctx, req, &out)
	if err != nil {
		return nil, resp, err
	}
	if out == nil {
		return nil, resp, fmt.Errorf("expected subscription quote response object")
	}
	return out, resp, nil
}
