package cloudsigma

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestAuditLogsSchemaRaw(t *testing.T) {
	fixtures := []string{
		// Retained rendered public schema, not inferred writable-log authorization.
		`{
"allowed_detail_http_methods": [],
"allowed_list_http_methods": [
"get"
],
"default_format": "application/json",
"default_limit": 20,
"fields": {
"action": {
"default": null,
"help_text": "Name of the executed action",
"readonly": false,
"required": true,
"type": "string"
},
"actor": {
"default": null,
"help_text": "User who executed action.",
"readonly": false,
"required": false,
"type": "related"
},
"category": {
"default": null,
"help_text": "Category the action belongs to.",
"readonly": false,
"required": true,
"type": "string"
},
"details": {
"default": null,
"help_text": "Details about the executed action.",
"readonly": false,
"required": false,
"type": "string"
},
"error_message": {
"default": null,
"help_text": "Error message, if applicable.",
"readonly": false,
"required": false,
"type": "string"
},
"error_point": {
"default": null,
"help_text": "The field that caused the error, if applicable.",
"readonly": false,
"required": false,
"type": "string"
},
"error_type": {
"default": null,
"help_text": "Type of error, if applicable.",
"readonly": false,
"required": false,
"type": "string"
},
"success": {
"default": true,
"help_text": "Whether the action was successful.",
"readonly": false,
"required": true,
"type": "boolean"
},
"timestamp": {
"default": null,
"help_text": "Time when entry was added.",
"readonly": false,
"required": true,
"type": "string"
},
"uuid": {
"default": null,
"help_text": "The uuid of resource the action is executed on. Can be empty, meaning no resource is referenced.",
"readonly": false,
"required": false,
"type": "string"
}
},
"filtering": {
"name": "exact",
"name__contains": "exact",
"tag": "exact",
"uuid": "exact"
}
}`,
		// Synthetic extensions and partial objects characterize retention, not valid schemas.
		`{"fields":{"future":{"default":9007199254740993,"order":[3,2,1]}},"extension":null}`, `{}`, `{"fields":null}`,
	}
	{
		for _, body := range fixtures {
			calls := 0
			c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
				calls++
				if q.Method != "GET" || q.URL.RequestURI() != "/api/2.0/logs/schema/" {
					t.Fatalf("wire: %s %s", q.Method, q.URL.RequestURI())
				}
				if q.Body != nil && q.Body != http.NoBody {
					b, e := io.ReadAll(q.Body)
					if e != nil || len(b) != 0 {
						t.Fatal("body", e)
					}
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
			})}))
			a, r, e := c.AuditLogs.SchemaRaw(context.Background())
			if e != nil || a == nil || r == nil || r.StatusCode != 200 || calls != 1 {
				t.Fatal("list", e, calls)
			}
			var expected map[string]json.RawMessage
			if err := json.Unmarshal([]byte(body), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(map[string]json.RawMessage(*a), expected) {
				t.Fatal("envelope/precision/null/order")
			}
			(*a)["injected"] = json.RawMessage(`true`)
			if raw := (*a)["fields"]; len(raw) > 0 {
				raw[0] = 'X'
				if (*a)["fields"][0] != 'X' {
					t.Fatal("mutation failed")
				}
			}
			b, _, e := c.AuditLogs.SchemaRaw(context.Background())
			if e != nil || calls != 2 || !reflect.DeepEqual(map[string]json.RawMessage(*b), expected) {
				t.Fatal("ownership", e)
			}
		}
	}
}

func TestAuditLogsSchemaErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `1`, `"x"`, `{`, ``} {
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: q}, nil
		})}))
		a, r, e := c.AuditLogs.SchemaRaw(context.Background())
		if a != nil || r != nil || e == nil {
			t.Fatal("malformed convention", body, e)
		}
	}
	sentinel := errors.New("synthetic transport")
	for _, code := range []int{403, 404, 500, 0} {
		calls := 0
		c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) {
			calls++
			if code == 0 {
				return nil, sentinel
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"error_type":"permission","error_message":"synthetic denied","error_point":null}]`)), Request: q}, nil
		})}))
		a, r, e := c.AuditLogs.SchemaRaw(context.Background())
		if a != nil || e == nil || calls != 1 {
			t.Fatal("error", e)
		}
		if code == 0 {
			if !errors.Is(e, sentinel) || r != nil {
				t.Fatal("transport", e)
			}
		} else {
			var api *ErrorResponse
			if !errors.As(e, &api) || r == nil || r.StatusCode != code || len(api.Errors) != 1 || api.Errors[0].Message != "synthetic denied" || api.Errors[0].Type != "permission" || api.Errors[0].Point != "" || api.Response != r {
				t.Fatal("typed error", e)
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: availabilityWire(func(q *http.Request) (*http.Response, error) { return nil, q.Context().Err() })}))
	if _, _, e := c.AuditLogs.SchemaRaw(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal("cancel", e)
	}
}
