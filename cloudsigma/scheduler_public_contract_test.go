package cloudsigma_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

// Expectations come from the public scheduler-binding example, not SDK encoders.
func TestSchedulerPreparsePublicContract(t *testing.T) {
	for _, code := range []int{200, 201, 202} {
		calls := 0
		c := sdk.NewClient(sdk.NewTokenCredentialsProvider("contract-token"), sdk.WithEndpoint("https://example.invalid/custom/api/2.0/"), sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
			calls++
			if q.Method != "POST" || q.URL.RequestURI() != "/custom/api/2.0/drives/34f9c721-ec23-4f5c-86ab-2839f687f14e/action/?do=set_scheduler" {
				t.Error("public route changed")
			}
			if q.Header.Get("Authorization") != "Bearer contract-token" {
				t.Errorf("token auth changed: %q", q.Header.Get("Authorization"))
			}
			var body map[string]json.RawMessage
			if e := json.NewDecoder(q.Body).Decode(&body); e != nil {
				t.Fatal(e)
			}
			_ = q.Body.Close()
			if len(body) != 1 || string(body["backup_policy_uuid"]) != `"103de42c-17a1-40ea-bc0f-521789168bbf"` {
				t.Fatal("public payload changed", body)
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"uuid":"drive-ack","backup_scheduler":{"uuid":"policy-ack","resource_uri":"/api/2.0/backupschedulers/policy-ack/"},"licenses":[{"license":{"free":0}}]}`)), Request: q}, nil
		})}))
		v, r, e := c.Drives.SetScheduler(context.Background(), "34f9c721-ec23-4f5c-86ab-2839f687f14e", "103de42c-17a1-40ea-bc0f-521789168bbf")
		if e != nil || r == nil || r.StatusCode != code || v == nil || v.Drive == nil || v.UUID != "drive-ack" || v.BackupScheduler == nil || v.BackupScheduler.UUID != "policy-ack" || v.BackupScheduler.ResourceURI != "/api/2.0/backupschedulers/policy-ack/" || len(v.Licenses) != 1 || v.Licenses[0].License.Free != "0" || calls != 1 {
			t.Fatalf("public acknowledgement changed: %v %v %v calls=%d", v, r, e, calls)
		}
	}
}
