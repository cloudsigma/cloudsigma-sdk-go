package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// BalanceService provides the public account balance read operation.
// Docs: https://docs.cloudsigma.com/en/latest/billing.html#balance
// It does not expose billing mutations, pricing or ledger operations.
type BalanceService service

// Balance is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The documented example uses a string balance, null credit_limit and string
// currency. Raw preservation does not validate currencies or imply that other
// representations are accepted by the service. Each Get returns independent data.
type Balance map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *Balance) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("balance response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// Get reads the authenticated account's balance using GET /balance/.
// Missing fields are preserved as missing, not defaulted or computed. Existing
// Client.Do error, status and single-JSON-decode semantics apply.
func (s *BalanceService) Get(ctx context.Context) (*Balance, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "balance/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(Balance)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
