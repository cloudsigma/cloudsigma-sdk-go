package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// AccountCreateRequest contains only the documented writable account creation
// fields. Email is always sent; nil Promo omits the optional promo code, while a
// non-nil pointer preserves even an empty string. Validation belongs to the server.
type AccountCreateRequest struct {
	Email string  `json:"email"`
	Promo *string `json:"promo,omitempty"`
}

// AccountCreateResult retains the complete response object, including unknown
// fields and numeric lexemes. A response does not establish account activation.
type AccountCreateResult map[string]json.RawMessage

// UnmarshalJSON requires an object and replaces the receiver only on success.
func (a *AccountCreateResult) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return fmt.Errorf("account creation response must be an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*a = fields
	return nil
}

// ErrAccountCreateRedirect indicates an unconfirmed account creation outcome.
// The server may have processed the initial POST; callers must not blindly retry.
var ErrAccountCreateRedirect = errors.New("account creation redirect refused; outcome unknown")

// CreateAccountRaw requests account creation and retains the response object.
// The documented HTTP 200 acknowledgment does not guarantee confirmation email
// delivery, activation, a password or a session. No retries or follow-up occur.
// Existing client authentication is preserved, not silently disabled. For an
// unauthenticated request, callers can provide a CredentialsProvider returning
// empty Credentials (no Authorization header); Basic/Bearer acceptance is not
// implied. This method does not acquire cookies or change caller credentials.
// Docs: https://docs.cloudsigma.com/en/latest/accounts.html
func (c *Client) CreateAccountRaw(ctx context.Context, account AccountCreateRequest) (*AccountCreateResult, *Response, error) {
	req, err := c.NewRequest(http.MethodPost, "accounts/action/?do=create", account)
	if err != nil {
		return nil, nil, err
	}
	value := new(AccountCreateResult)
	resp, err := c.doResourceAction(ctx, req, value, ErrAccountCreateRedirect)
	if err != nil {
		return nil, resp, err
	}
	return value, resp, nil
}
