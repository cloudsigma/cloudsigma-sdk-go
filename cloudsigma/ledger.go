package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// LedgerService provides the public account ledger read operation.
// Docs: https://docs.cloudsigma.com/en/latest/billing.html#ledger
// It does not expose mutations, transaction detail routes or automatic traversal.
type LedgerService service

// Ledger is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The documented collection envelope includes meta and objects with decimal
// amount, initial and end strings. Raw preservation does not validate currencies or imply that other
// representations are accepted by the service. Each Get returns independent data.
type Ledger map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *Ledger) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("ledger response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// Get reads one documented GET /ledger/ collection response without query parameters.
// No filters, singleton route or pagination traversal are inferred from the schema.
// Missing fields are preserved as missing, not defaulted or computed. Existing
// Client.Do error, status and single-JSON-decode semantics apply.
func (s *LedgerService) Get(ctx context.Context) (*Ledger, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "ledger/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(Ledger)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
