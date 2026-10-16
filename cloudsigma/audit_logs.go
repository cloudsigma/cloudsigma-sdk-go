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

// AuditLogsService provides read access to public audit logs.
// Docs: https://docs.cloudsigma.com/en/latest/audit_logs.html
type AuditLogsService service

// AuditLogs is a raw JSON object preserving decimal strings, numeric lexemes,
// unknown fields and absent-versus-null values without float conversion.
// The complete collection envelope is retained. Raw preservation does not validate
// audit-log semantics or imply server acceptance. Each call returns independent data.
type AuditLogs map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (b *AuditLogs) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("logs response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*b = fields
	return nil
}

// ListRaw reads one GET /logs/ collection envelope, preserving meta, objects
// and unknown members without projecting them into Response.Meta. It does not
// traverse pages, interpret action success as resource completion, or validate
// the log schema. Existing Client.Do decoding/error rules apply.
func (s *AuditLogsService) ListRaw(ctx context.Context, opts *ListOptions) (*AuditLogs, *Response, error) {
	query := url.Values{}
	if opts != nil {
		query.Set("limit", strconv.Itoa(opts.Limit))
		if opts.Offset != 0 {
			query.Set("offset", strconv.Itoa(opts.Offset))
		}
	}
	path := "logs/"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(AuditLogs)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}

// SchemaRaw reads GET /logs/schema/ as a complete raw object. It preserves
// field descriptions, defaults, method lists and unknown members without
// interpreting them as permission to write logs or as validation rules.
// Existing Client.Do decoding/error rules apply.
func (s *AuditLogsService) SchemaRaw(ctx context.Context) (*AuditLogs, *Response, error) {
	req, err := s.client.NewRequest(http.MethodGet, "logs/schema/", nil)
	if err != nil {
		return nil, nil, err
	}
	value := new(AuditLogs)
	resp, err := s.client.Do(ctx, req, value)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
