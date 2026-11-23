package cloudsigma

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptions_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"auto_renew":true,"uuid":"long-uuid"}],"meta":{"total_count":1}}`)
	})
	expected := []Subscription{
		{
			AutoRenew: true,
			UUID:      "long-uuid",
		},
	}

	subscriptions, resp, err := client.Subscriptions.List(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, subscriptions)
	assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestSubscriptions_ListHTTPError(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/subscriptions/", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `[{"error_type":"synthetic","error_message":"denied","error_point":"list"}]`)
	})

	subscriptions, resp, err := client.Subscriptions.List(ctx)
	var apiErr *ErrorResponse
	assert.ErrorAs(t, err, &apiErr)
	assert.Nil(t, subscriptions)
	if assert.NotNil(t, resp) {
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	}
	if assert.NotNil(t, apiErr) {
		assert.Same(t, resp, apiErr.Response)
		assert.Len(t, apiErr.Errors, 1)
		assert.Equal(t, "synthetic", apiErr.Errors[0].Type)
		assert.Equal(t, "denied", apiErr.Errors[0].Message)
		assert.Equal(t, "list", apiErr.Errors[0].Point)
	}
}

func TestSubscriptions_ListContextCancellation(t *testing.T) {
	setup()
	defer teardown()
	canceled, cancel := context.WithCancel(ctx)
	cancel()

	subscriptions, resp, err := client.Subscriptions.List(canceled)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, subscriptions)
	assert.Nil(t, resp)
}

func TestSubscriptions_Create(t *testing.T) {
	setup()
	defer teardown()

	input := &SubscriptionCreateRequest{
		Subscriptions: []Subscription{
			{Amount: "3000", Resource: "dssd"},
		},
	}
	mux.HandleFunc("/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
		v := new(SubscriptionCreateRequest)
		_ = json.NewDecoder(r.Body).Decode(v)
		assert.Equal(t, input, v)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = fmt.Fprint(w, `{"objects":[{"amount":"3000","resource":"dssd","uuid":"long-uuid"}]}`)
	})
	expected := []Subscription{
		{Amount: "3000", Resource: "dssd", UUID: "long-uuid"},
	}

	subscriptions, _, err := client.Subscriptions.Create(ctx, input)

	assert.NoError(t, err)
	assert.Equal(t, expected, subscriptions)
}

func TestSubscriptions_Create_emptyPayload(t *testing.T) {
	_, _, err := client.Subscriptions.Create(ctx, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPayloadNotAllowed.Error(), err.Error())
}
