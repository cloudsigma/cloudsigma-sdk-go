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

type generateWire struct {
	t       *testing.T
	want    string
	body    string
	code    int
	calls   int
	failure error
}

func (w *generateWire) RoundTrip(r *http.Request) (*http.Response, error) {
	w.calls++
	if r.Method != "POST" || r.URL.RequestURI() != "/api/2.0/keypairs/" {
		w.t.Fatalf("route: %s %s", r.Method, r.URL)
	}
	b, e := io.ReadAll(r.Body)
	if e != nil {
		w.t.Fatal(e)
	}
	var got, want interface{}
	if json.Unmarshal(b, &got) != nil || json.Unmarshal([]byte(w.want), &want) != nil || !reflect.DeepEqual(got, want) {
		w.t.Fatalf("body: %s want %s", b, w.want)
	}
	if e = r.Context().Err(); e != nil {
		return nil, e
	}
	if w.failure != nil {
		return nil, w.failure
	}
	return &http.Response{StatusCode: w.code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(w.body)), Request: r}, nil
}
func generateClient(w *generateWire) *Client {
	return NewClient(NewTokenCredentialsProvider("synthetic"), WithHTTPClient(&http.Client{Transport: w}))
}
func TestGenerateKeypairAlgorithms(t *testing.T) {
	for _, algorithm := range []string{"rsa", "ed25519"} {
		t.Run(algorithm, func(t *testing.T) {
			w := &generateWire{t: t, want: `{"objects":[{"name":"fixture","algorithm_type":"` + algorithm + `"}]}`, code: 201, body: `{"objects":[{"uuid":"generated","name":"fixture","private_key":"synthetic-private","public_key":"synthetic-public","fingerprint":"fixture-fingerprint"}]}`}
			input := &KeypairGenerateRequest{Name: "fixture", AlgorithmType: algorithm}
			before := *input
			keys, r, e := generateClient(w).Keypairs.Generate(context.Background(), input)
			if e != nil || r == nil || r.StatusCode != 201 || len(keys) != 1 || keys[0].UUID != "generated" || keys[0].PrivateKey != "synthetic-private" || keys[0].PublicKey != "synthetic-public" || keys[0].Fingerprint != "fixture-fingerprint" || w.calls != 1 || *input != before {
				t.Fatalf("result: %v %v calls=%d", r, e, w.calls)
			}
		})
	}
}
func TestGenerateKeypairRefusal(t *testing.T) {
	for _, input := range []*KeypairGenerateRequest{nil, {}, {AlgorithmType: "RSA"}, {AlgorithmType: "unknown"}} {
		w := &generateWire{t: t}
		keys, r, e := generateClient(w).Keypairs.Generate(context.Background(), input)
		if e == nil || r != nil || keys != nil || w.calls != 0 {
			t.Fatal("invalid request dispatched or accepted")
		}
		if input == nil && !errors.Is(e, ErrEmptyPayloadNotAllowed) {
			t.Fatal(e)
		}
	}
}
func TestGenerateKeypairFailures(t *testing.T) {
	for _, code := range []int{401, 403, 500} {
		w := &generateWire{t: t, want: `{"objects":[{"algorithm_type":"rsa"}]}`, code: code, body: `[{"error_type":"permission","error_msg":"fixture"}]`}
		keys, r, e := generateClient(w).Keypairs.Generate(context.Background(), &KeypairGenerateRequest{AlgorithmType: "rsa"})
		var api *ErrorResponse
		if !errors.As(e, &api) || r == nil || r.StatusCode != code || keys != nil || w.calls != 1 {
			t.Fatalf("status %d: %v %v", code, r, e)
		}
	}
	for _, body := range []string{`{`, `[]`} {
		w := &generateWire{t: t, want: `{"objects":[{"algorithm_type":"rsa"}]}`, code: 201, body: body}
		keys, _, e := generateClient(w).Keypairs.Generate(context.Background(), &KeypairGenerateRequest{AlgorithmType: "rsa"})
		if e == nil || keys != nil || w.calls != 1 {
			t.Fatal("malformed response accepted")
		}
	}
	sentinel := errors.New("fixture transport")
	w := &generateWire{t: t, want: `{"objects":[{"algorithm_type":"rsa"}]}`, failure: sentinel}
	_, _, e := generateClient(w).Keypairs.Generate(context.Background(), &KeypairGenerateRequest{AlgorithmType: "rsa"})
	if !errors.Is(e, sentinel) || w.calls != 1 {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w = &generateWire{t: t, want: `{"objects":[{"algorithm_type":"rsa"}]}`}
	_, _, e = generateClient(w).Keypairs.Generate(ctx, &KeypairGenerateRequest{AlgorithmType: "rsa"})
	if !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
