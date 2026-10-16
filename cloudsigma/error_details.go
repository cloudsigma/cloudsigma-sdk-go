package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ErrorDetails retains an error response body without assuming a universal JSON
// envelope. It contains no headers. Bodies may contain sensitive data; callers
// must not log them indiscriminately. Accessors return independent copies.
type ErrorDetails struct {
	body    []byte
	readErr error
}

// RawBody returns the bytes actually read, including partial or non-JSON bodies.
func (d *ErrorDetails) RawBody() []byte { return append([]byte(nil), d.body...) }

// ReadError reports a body read failure independently of the legacy API error.
func (d *ErrorDetails) ReadError() error { return d.readErr }

// ErrorDetail is an opt-in representation of one JSON error object. Missing
// error_point is observable but is not a documented valid provider state.
type ErrorDetail struct {
	Message string `json:"error_message"`
	Type    string `json:"error_type"`
	point   *string
	present bool
}

// Point returns the nullable point and its presence. A nil value with present
// true means JSON null; a nonnil pointer can contain an empty string.
func (d ErrorDetail) Point() (value *string, present bool) {
	if d.point == nil {
		return nil, d.present
	}
	v := *d.point
	return &v, d.present
}

// DecodeErrorDetails retains caller-owned bytes without parsing an envelope.
func DecodeErrorDetails(body []byte) *ErrorDetails {
	return &ErrorDetails{body: append([]byte(nil), body...)}
}

// Object explicitly decodes the object form described by the public error docs.
// It does not normalize arrays, scalars, empty bodies or non-JSON into objects.
func (d *ErrorDetails) Object() (ErrorDetail, error) { return decodeErrorDetail(d.body) }

// Array explicitly decodes the array form illustrated for POST subscriptions.
// This does not assert that any other operation uses this envelope.
func (d *ErrorDetails) Array() ([]ErrorDetail, error) {
	b := bytes.TrimSpace(d.body)
	if len(b) == 0 || b[0] != '[' {
		return nil, fmt.Errorf("expected error array")
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	result := make([]ErrorDetail, len(raw))
	for i := range raw {
		detail, err := decodeErrorDetail(raw[i])
		if err != nil {
			return nil, err
		}
		result[i] = detail
	}
	return result, nil
}

func decodeErrorDetail(body []byte) (ErrorDetail, error) {
	var result ErrorDetail
	b := bytes.TrimSpace(body)
	if len(b) == 0 || b[0] != '{' {
		return result, fmt.Errorf("expected error object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return result, err
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return ErrorDetail{}, err
	}
	if point, ok := fields["error_point"]; ok {
		result.present = true
		if err := json.Unmarshal(point, &result.point); err != nil {
			return ErrorDetail{}, err
		}
	}
	return result, nil
}

// WithErrorDetails opts a single operation into retaining its error body. Use
// ErrorDetailsFromContext after an SDK call with the returned context. Create a
// fresh context per operation; reusing a capture context is unsupported.
// Capture covers Client.Do and standalone CheckResponse, not alternate transports.
// Client.Do clears its slot on entry, but service validation before Do does not.
// Reuse can therefore expose a prior operation's body; overlapping calls are not
// request-correlated. With a fresh context, failures before capture leave no details.
// The SDK's
// returned error, response, read count and body closure semantics are unchanged.
func WithErrorDetails(ctx context.Context) context.Context {
	return context.WithValue(ctx, errorDetailsKey{}, &errorDetailsSlot{})
}

// ErrorDetailsFromContext returns retained error details, or nil when absent.
func ErrorDetailsFromContext(ctx context.Context) *ErrorDetails {
	slot, _ := ctx.Value(errorDetailsKey{}).(*errorDetailsSlot)
	if slot == nil {
		return nil
	}
	slot.mu.Lock()
	defer slot.mu.Unlock()
	return slot.details
}

type errorDetailsKey struct{}
type errorDetailsSlot struct {
	mu      sync.Mutex
	details *ErrorDetails
}

func retainErrorDetailsContext(ctx context.Context, body []byte, readErr error) {
	slot, _ := ctx.Value(errorDetailsKey{}).(*errorDetailsSlot)
	if slot == nil {
		return
	}
	var details *ErrorDetails
	if body != nil || readErr != nil {
		details = DecodeErrorDetails(body)
		details.readErr = readErr
	}
	slot.mu.Lock()
	slot.details = details
	slot.mu.Unlock()
}
