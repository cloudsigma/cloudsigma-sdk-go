package cloudsigma_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// The public cloning contract permits an omitted body and optional definition
// fields. A 202 acknowledges acceptance, not clone completion. Token auth here
// characterizes existing client compatibility, not a new public auth contract.
func TestDriveCloneOptionsPublicContract(t *testing.T) {
	for _, mode := range []string{"nil", "empty", "false", "true", "legacytrue", "legacyfalse"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			c := sdk.NewClient(sdk.NewTokenCredentialsProvider("synthetic"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
				calls++
				var b []byte
				if q.Body != nil {
					b, _ = io.ReadAll(q.Body)
					_ = q.Body.Close()
				}
				if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/drives/fixture/action/?do=clone" || q.Header.Get("Authorization") != "Bearer synthetic" {
					t.Fatal("route/auth changed")
				}
				if strings.TrimSpace(string(b)) != drivecloneoptionsPreparseBody(mode) {
					t.Fatalf("body %s", b)
				}
				return &http.Response{StatusCode: 202, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objects":[{"uuid":"clone-id","status":"cloning_dst"}]}`)), Request: q}, nil
			})}))
			var options *sdk.DriveCloneOptions
			switch mode {
			case "empty":
				options = &sdk.DriveCloneOptions{}
			case "false", "true":
				v := mode == "true"
				options = &sdk.DriveCloneOptions{Definition: &sdk.Drive{Name: "fixture"}, AllowMultimount: &v}
			case "legacytrue", "legacyfalse":
				options = &sdk.DriveCloneOptions{Definition: &sdk.Drive{Name: "fixture", AllowMultimount: mode == "legacytrue"}}
			}
			out, r, e := c.Drives.CloneWithOptions(context.Background(), "fixture", options)
			if e != nil || r == nil || r.StatusCode != 202 || calls != 1 || out == nil || out.UUID != "clone-id" || out.Status != "cloning_dst" {
				t.Fatalf("ACK %v %v %v", out, r, e)
			}
			out, r, e = c.Drives.CloneWithOptions(context.Background(), "", options)
			if out != nil || r != nil || !errors.Is(e, sdk.ErrEmptyArgument) || calls != 1 {
				t.Fatal("empty identifier dispatched")
			}
		})
	}
}
