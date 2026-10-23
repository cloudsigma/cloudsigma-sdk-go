package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CalculateExtension returns the price of the default extension of an existing
// subscription, without purchasing or extending it. It sends the documented
// empty object; overriding the extension period or end time is not supported.
// The complete raw singleton quote is preserved without local price calculation,
// schema validation or redaction. Unsafe identifiers are refused as SDK policy,
// not a UUID requirement (the public example uses a numeric identifier).
// One POST is made, without retries, polling or follow-up requests. Shared
// Client.Do success, decoding and error rules apply.
// Redirects are refused with ErrSubscriptionQuoteRedirect; the initial POST
// outcome is unknown. Callers must not blindly retry.
// Docs: https://docs.cloudsigma.com/en/latest/subscriptions.html#calculator
func (s *SubscriptionsService) CalculateExtension(ctx context.Context, id string) (map[string]json.RawMessage, *Response, error) {
	if err := validateServerDefinitionID(id); err != nil {
		return nil, nil, err
	}
	req, err := s.client.NewRequest(http.MethodPost, "subscriptioncalculator/"+id+"/action/?do=extend", struct{}{})
	if err != nil {
		return nil, nil, err
	}
	var value map[string]json.RawMessage
	resp, err := s.client.doResourceAction(ctx, req, &value, ErrSubscriptionQuoteRedirect)
	if err != nil {
		return nil, resp, err
	}
	if value == nil {
		return nil, resp, fmt.Errorf("expected subscription extension quote response object")
	}
	return value, resp, nil
}
