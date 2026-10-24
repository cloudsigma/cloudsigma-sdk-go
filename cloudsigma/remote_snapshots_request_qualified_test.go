package cloudsigma

import (
	"encoding/json"
	"testing"
)

func TestRemoteSnapshotCreateRequestTypedWireAndValidation(t *testing.T) {
	req, err := NewRemoteSnapshotCreateRequest(RemoteSnapshotCreate{
		Drive: &RemoteSnapshotDriveReference{UUID: "d-1"}, Location: "alias", Name: "n",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"objects":[{"drive":{"uuid":"d-1"},"location":"alias","name":"n"}]}`
	if string(got) != want {
		t.Fatalf("wire = %s, want %s", got, want)
	}
	if _, err := NewRemoteSnapshotCreateRequest(RemoteSnapshotCreate{}); err == nil {
		t.Fatal("expected missing drive error")
	}
}

func TestRemoteSnapshotDriveReferenceAcceptsWireForms(t *testing.T) {
	for _, wire := range []string{`"d-1"`, `{"uuid":"d-1"}`, `{"uuid":"d-1","resource_uri":"/api/2.0/drives/d-1/"}`} {
		var ref RemoteSnapshotDriveReference
		if err := json.Unmarshal([]byte(wire), &ref); err != nil {
			t.Fatalf("%s: %v", wire, err)
		}
		if ref.UUID != "d-1" {
			t.Fatalf("%s: uuid=%q", wire, ref.UUID)
		}
	}
}

func TestRemoteSnapshotCreateRequestLegacyWireIncludesFullSnapshot(t *testing.T) {
	req := RemoteSnapshotCreateRequest{RemoteSnapshots: []RemoteSnapshot{{Location: "ZRH", Snapshot: Snapshot{UUID: "r-1", Name: "n", Meta: map[string]interface{}{"x": "y"}}}}}
	got, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"objects":[{"meta":{"x":"y"},"name":"n","tags":[],"uuid":"r-1"}]}` {
		t.Fatalf("legacy wire = %s", got)
	}
}
