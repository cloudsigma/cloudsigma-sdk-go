package cloudsigma_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
)

func TestRemoteSnapshotQualifiedConstructorOwnership(t *testing.T) {
	drive := &sdk.RemoteSnapshotDriveReference{UUID: "d1", ResourceURI: "/drives/d1/"}
	input := []sdk.RemoteSnapshotCreate{{Drive: drive, Location: "l1", Name: "first"}, {Drive: drive, Name: "second"}}
	req, e := sdk.NewRemoteSnapshotCreateRequest(input...)
	if e != nil {
		t.Fatal(e)
	}
	input[0].Name = "changed"
	drive.UUID = "changed"
	drive.ResourceURI = "changed"
	if input[0].Name != "changed" || drive.UUID != "changed" {
		t.Fatal("mutation did not occur")
	}
	if req.Objects[0].Name != "first" || req.Objects[0].Drive.UUID != "d1" || req.Objects[0].Drive.ResourceURI != "/drives/d1/" {
		t.Fatal("input aliases request")
	}
	req.Objects[0].Drive.UUID = "result-change"
	if req.Objects[0].Drive.UUID != "result-change" || req.Objects[1].Drive.UUID != "d1" || drive.UUID != "changed" {
		t.Fatal("object ownership")
	}
}
func TestRemoteSnapshotQualifiedInvalidLateBatch(t *testing.T) {
	var calls int32
	c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(*http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errors.New("unexpected dispatch")
	})}))
	for _, bad := range []sdk.RemoteSnapshotCreate{{}, {Drive: &sdk.RemoteSnapshotDriveReference{}}, {Drive: &sdk.RemoteSnapshotDriveReference{ResourceURI: "/drive/only/"}}} {
		objects := []sdk.RemoteSnapshotCreate{{Drive: &sdk.RemoteSnapshotDriveReference{UUID: "valid"}}, bad}
		if _, e := sdk.NewRemoteSnapshotCreateRequest(objects...); e == nil {
			t.Fatal("constructor accepted invalid late member")
		}
		v, r, e := c.RemoteSnapshots.CreateDefinitions(context.Background(), &sdk.RemoteSnapshotCreateDefinitionsRequest{Objects: objects})
		if e == nil || v != nil || r != nil {
			t.Fatal("manual request accepted")
		}
	}
	v, r, e := c.RemoteSnapshots.CreateDefinitions(context.Background(), nil)
	if !errors.Is(e, sdk.ErrEmptyPayloadNotAllowed) || v != nil || r != nil || calls != 0 {
		t.Fatal("dispatch or nil policy")
	}
}
func TestRemoteSnapshotQualified201OrderedResponse(t *testing.T) {
	for _, construction := range []string{"constructor", "literal"} {
		t.Run(construction, func(t *testing.T) {
			var calls, closed int32
			hc := &http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
				atomic.AddInt32(&calls, 1)
				b, _ := io.ReadAll(q.Body)
				_ = q.Body.Close()
				want := `{"objects":[{"drive":{"uuid":"d1","resource_uri":"/drives/d1/"},"location":"l1","name":"first"},{"drive":{"uuid":"d2"},"name":"second"}]}`
				if q.Method != "POST" || q.URL.RequestURI() != "/api/2.0/remotesnapshots/" || strings.TrimSpace(string(b)) != want {
					t.Errorf("wire %s %s %s", q.Method, q.URL, b)
				}
				return &http.Response{StatusCode: 201, Header: make(http.Header), Request: q, Body: &resourceRedirectBody{Reader: strings.NewReader(`{"objects":[{"uuid":"r1","resource_uri":"/remotesnapshots/r1/","status":"creating","location":"l1","drive":{"uuid":"d1"},"drive_meta":{"name":"first","size":4096}},{"uuid":"r2","resource_uri":"/remotesnapshots/r2/","status":"available","drive":{"uuid":"d2"},"drive_meta":{"name":"second"}}]}`), closed: &closed}}, nil
			})}
			c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithHTTPClient(hc))
			objects := []sdk.RemoteSnapshotCreate{{Drive: &sdk.RemoteSnapshotDriveReference{UUID: "d1", ResourceURI: "/drives/d1/"}, Location: "l1", Name: "first"}, {Drive: &sdk.RemoteSnapshotDriveReference{UUID: "d2"}, Name: "second"}}
			req := &sdk.RemoteSnapshotCreateDefinitionsRequest{Objects: objects}
			if construction == "constructor" {
				var e error
				req, e = sdk.NewRemoteSnapshotCreateRequest(objects...)
				if e != nil {
					t.Fatal(e)
				}
			}
			v, r, e := c.RemoteSnapshots.CreateDefinitions(context.Background(), req)
			if e != nil || r == nil || r.StatusCode != 201 || len(v) != 2 {
				t.Fatalf("response %v %v %v", v, r, e)
			}
			if v[0].UUID != "r1" || v[1].UUID != "r2" || v[0].Drive.UUID != "d1" || v[1].Drive.UUID != "d2" || v[0].Status != "creating" || v[1].Status != "available" || v[0].ResourceURI != "/remotesnapshots/r1/" || v[1].ResourceURI != "/remotesnapshots/r2/" || v[0].Location != "l1" || v[0].RemoteSnapshotDriveMetadata.Size != 4096 || v[1].RemoteSnapshotDriveMetadata.Name != "second" || calls != 1 || closed != 1 {
				t.Fatalf("ordered response %v", v)
			}
		})
	}
}
func TestRemoteSnapshotQualifiedHTTPError(t *testing.T) {
	for _, action := range []string{"legacy", "typed"} {
		for _, payload := range []string{`[{"error_type":"invalid","error_point":"drive","error_message":"bad drive"}]`, `not-json`} {
			var closed int32
			c := sdk.NewClient(logoutCookieCredentials{}, sdk.WithHTTPClient(&http.Client{Transport: resourceRedirectTransport(func(q *http.Request) (*http.Response, error) {
				_ = q.Body.Close()
				return &http.Response{StatusCode: 400, Header: make(http.Header), Request: q, Body: &resourceRedirectBody{Reader: strings.NewReader(payload), closed: &closed}}, nil
			})}))
			v, r, e := remoteSnapshotPreparseCall(c, action)
			var he *sdk.ErrorResponse
			if v || r == nil || r.StatusCode != 400 || e == nil || closed != 1 {
				t.Fatalf("HTTP error %v %v", r, e)
			}
			if strings.HasPrefix(payload, "[") && (!errors.As(e, &he) || he.Response != r || len(he.Errors) != 1 || he.Errors[0].Type != "invalid" || he.Errors[0].Point != "drive" || he.Errors[0].Message != "bad drive") {
				t.Fatal("typed errors lost")
			}
			if payload == "not-json" {
				var syntax *json.SyntaxError
				if !errors.As(e, &syntax) {
					t.Fatal("raw malformed error changed")
				}
			}
		}
	}
}
func TestRemoteSnapshotQualifiedLegacyFullGolden(t *testing.T) {
	req := sdk.RemoteSnapshotCreateRequest{RemoteSnapshots: []sdk.RemoteSnapshot{{Location: "legacy-omitted", RemoteSnapshotDriveMetadata: &sdk.RemoteSnapshotDriveMetadata{Name: "legacy-omitted"}, Snapshot: sdk.Snapshot{Drive: &sdk.Drive{UUID: "d", Name: "full-drive", Size: 4096, Status: "mounted", Meta: map[string]interface{}{"key": "value"}}, Meta: map[string]interface{}{"key": "value"}, Name: "remote", Tags: []sdk.Tag{{UUID: "tag"}}, UUID: "r"}}}}
	b, e := json.Marshal(req)
	if e != nil {
		t.Fatal(e)
	}
	want := `{"objects":[{"drive":{"meta":{"key":"value"},"name":"full-drive","size":4096,"status":"mounted","uuid":"d"},"meta":{"key":"value"},"name":"remote","tags":[{"uuid":"tag"}],"uuid":"r"}]}`
	if string(b) != want {
		t.Fatalf("legacy bytes %s", b)
	}
}
