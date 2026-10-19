package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// DailyBurstUsageService provides the public daily burst usage read operation.
// Docs: https://docs.cloudsigma.com/en/latest/burst_usage.html
// It does not expose mutations, transaction detail routes or automatic traversal.
type DailyBurstUsageService service

// DailyBurstUsage is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The complete collection envelope is retained. Raw preservation does not validate
// billing semantics or imply server acceptance. Each call returns independent data.
type DailyBurstUsage map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *DailyBurstUsage) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("dailyburstusage response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// DailyBurstUsageListOptions selects the documented date filters and optional
// pagination. Empty date strings omit their filters; nil ListOptions omits both
// paging parameters. Dates are forwarded unchanged, not parsed or calculated.
type DailyBurstUsageListOptions struct {
	DateGT string `url:"date__gt,omitempty"`
	DateLT string `url:"date__lt,omitempty"`
	*ListOptions
}

// ListRaw reads one GET /dailyburstusage/ collection envelope. It preserves meta,
// objects and unknown envelope fields without projecting them into Response.Meta.
// No pagination traversal, date-boundary calculation, billing interpretation or
// schema validation is performed. Existing Client.Do decoding/error rules apply.
func (s *DailyBurstUsageService) ListRaw(ctx context.Context, opts *DailyBurstUsageListOptions) (*DailyBurstUsage, *Response, error) {
	query := url.Values{}
	if opts != nil {
		if opts.DateGT != "" {
			query.Set("date__gt", opts.DateGT)
		}
		if opts.DateLT != "" {
			query.Set("date__lt", opts.DateLT)
		}
		if opts.ListOptions != nil {
			query.Set("limit", strconv.Itoa(opts.Limit))
			if opts.Offset != 0 {
				query.Set("offset", strconv.Itoa(opts.Offset))
			}
		}
	}
	path := "dailyburstusage/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(DailyBurstUsage)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
