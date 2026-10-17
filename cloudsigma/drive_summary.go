package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// DriveSummaryOptions selects field names for the documented brief listing.
// Nil/empty Fields omits projection. Names use ASCII letters/digits/underscore
// (SDK routing policy, not a server field whitelist); order/duplicates retained.
type DriveSummaryOptions struct{ Fields []string }

// DriveSummaryPage retains arbitrary projected object fields and metadata.
// RawMessage is necessary because projection can select fields outside Drive's
// typed model, and omission must not become a zero value. It is not redaction.
// Nested schemas are not validated; each call owns fresh decoded bytes.
type DriveSummaryPage map[string]json.RawMessage

// UnmarshalJSON requires an outer object, preserving missing/null nested fields.
func (p *DriveSummaryPage) UnmarshalJSON(b []byte) error {
	var v map[string]json.RawMessage
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("drive summary must be an object")
	}
	*p = v
	return nil
}

// ListSummary reads one GET /drives/ envelope, optionally projecting fields.
// Existing List remains the detailed typed reader. No pagination/filter options
// or traversal are inferred. Existing Client.Do status/error semantics apply.
// Docs: https://docs.cloudsigma.com/en/latest/drives.html#listing
func (s *DrivesService) ListSummary(ctx context.Context, opts *DriveSummaryOptions) (*DriveSummaryPage, *Response, error) {
	q := url.Values{}
	if opts != nil && len(opts.Fields) > 0 {
		for _, field := range opts.Fields {
			if field == "" {
				return nil, nil, fmt.Errorf("empty drive field")
			}
			for _, c := range field {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
					return nil, nil, fmt.Errorf("invalid drive field")
				}
			}
		}
		q.Set("fields", strings.Join(opts.Fields, ","))
	}
	path := "drives/"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	v := new(DriveSummaryPage)
	resp, err := s.client.Do(ctx, req, v)
	if err != nil {
		return nil, resp, err
	}
	return v, resp, nil
}
