package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PricingService provides the public account pricing read operation.
// Docs: https://docs.cloudsigma.com/en/latest/billing.html#pricing
// It does not expose billing mutations, singleton prices or ledger operations.
type PricingService service

// Pricing is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The documented envelope includes current/next burst levels, meta and objects
// containing decimal price strings and currency. Raw preservation does not validate currencies or imply that other
// representations are accepted by the service. Each Get returns independent data.
type Pricing map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *Pricing) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("pricing response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// Get reads one public pricing response using GET /pricing/.
// Nil limit omits the query; a supplied nonnegative limit is explicit, including zero.
// No traversal, category filter or singleton price route is inferred.
// Missing fields are preserved as missing, not defaulted or computed. Existing
// Client.Do error, status and single-JSON-decode semantics apply.
func (s *PricingService) Get(ctx context.Context, limit *int) (*Pricing, *Response, error) {
	path := "pricing/"
	if limit != nil {
		if *limit < 0 {
			return nil, nil, fmt.Errorf("pricing limit must be nonnegative")
		}
		path += fmt.Sprintf("?limit=%d", *limit)
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(Pricing)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
