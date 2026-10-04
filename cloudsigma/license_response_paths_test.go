package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestLicenseServiceResponsePaths(t *testing.T) {
	license := `{"name":"windows","free":0,"initial_purchase_requirement":2,"subsequent_purchase_requirement":1}`
	drive := `{"uuid":"id","name":"disk","licenses":[{"amount":1,"license":` + license + `}]}`
	serverBody := `{"uuid":"id","drives":[{"boot_order":1,"drive":` + drive + `}]}`
	snapshotBody := `{"uuid":"id","drive":` + drive + `}`
	for _, tc := range []struct {
		name, body string
		list       bool
		call       func() (*License, *Response, error)
	}{
		{"Drives/List", drive, true, func() (*License, *Response, error) {
			result, resp, err := client.Drives.List(ctx, nil)
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Licenses[0].License, resp, nil
		}},
		{"Drives/Get", drive, false, func() (*License, *Response, error) {
			result, resp, err := client.Drives.Get(ctx, "id")
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Licenses[0].License, resp, nil
		}},
		{"Drives/Create", drive, true, func() (*License, *Response, error) {
			result, resp, err := client.Drives.Create(ctx, &DriveCreateRequest{Drives: []Drive{{Name: "new"}}})
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Licenses[0].License, resp, nil
		}},
		{"Drives/Update", drive, false, func() (*License, *Response, error) {
			result, resp, err := client.Drives.Update(ctx, "id", &DriveUpdateRequest{Drive: &Drive{Name: "new"}})
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Licenses[0].License, resp, nil
		}},
		{"Drives/Resize", drive, true, func() (*License, *Response, error) {
			result, resp, err := client.Drives.Resize(ctx, "id", &DriveUpdateRequest{Drive: &Drive{Size: 536870912}})
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Licenses[0].License, resp, nil
		}},
		{"Drives/Clone", drive, true, func() (*License, *Response, error) {
			result, resp, err := client.Drives.Clone(ctx, "id", nil)
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Licenses[0].License, resp, nil
		}},
		{"LibraryDrives/List", drive, true, func() (*License, *Response, error) {
			result, resp, err := client.LibraryDrives.List(ctx, nil)
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Licenses[0].License, resp, nil
		}},
		{"LibraryDrives/Get", drive, false, func() (*License, *Response, error) {
			result, resp, err := client.LibraryDrives.Get(ctx, "id")
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Licenses[0].License, resp, nil
		}},
		{"LibraryDrives/Clone", drive, true, func() (*License, *Response, error) {
			result, resp, err := client.LibraryDrives.Clone(ctx, "id", nil)
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Licenses[0].License, resp, nil
		}},
		{"Servers/List", serverBody, true, func() (*License, *Response, error) {
			result, resp, err := client.Servers.List(ctx)
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Drives[0].Drive.Licenses[0].License, resp, nil
		}},
		{"Servers/Get", serverBody, false, func() (*License, *Response, error) {
			result, resp, err := client.Servers.Get(ctx, "id")
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Drives[0].Drive.Licenses[0].License, resp, nil
		}},
		{"Servers/Create", serverBody, true, func() (*License, *Response, error) {
			result, resp, err := client.Servers.Create(ctx, &ServerCreateRequest{Servers: []Server{{Name: "new"}}})
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Drives[0].Drive.Licenses[0].License, resp, nil
		}},
		{"Servers/Update", serverBody, false, func() (*License, *Response, error) {
			result, resp, err := client.Servers.Update(ctx, "id", &ServerUpdateRequest{Server: &Server{Name: "new"}})
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Drives[0].Drive.Licenses[0].License, resp, nil
		}},
		{"Snapshots/List", snapshotBody, true, func() (*License, *Response, error) {
			result, resp, err := client.Snapshots.List(ctx)
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Drive.Licenses[0].License, resp, nil
		}},
		{"Snapshots/Get", snapshotBody, false, func() (*License, *Response, error) {
			result, resp, err := client.Snapshots.Get(ctx, "id")
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Drive.Licenses[0].License, resp, nil
		}},
		{"Snapshots/Create", snapshotBody, true, func() (*License, *Response, error) {
			result, resp, err := client.Snapshots.Create(ctx, &SnapshotCreateRequest{Snapshots: []Snapshot{{Name: "new"}}})
			if err != nil {
				return nil, resp, err
			}
			if len(result) != 1 {
				return nil, resp, fmt.Errorf("unexpected length %d", len(result))
			}
			v := result[0]
			return v.Drive.Licenses[0].License, resp, nil
		}},
		{"Snapshots/Update", snapshotBody, false, func() (*License, *Response, error) {
			result, resp, err := client.Snapshots.Update(ctx, "id", &SnapshotUpdateRequest{Snapshot: &Snapshot{Name: "new"}})
			if err != nil {
				return nil, resp, err
			}
			v := result
			return v.Drive.Licenses[0].License, resp, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setup()
			defer teardown()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if tc.list {
					fmt.Fprintf(w, `{"objects":[%s],"meta":{"total_count":1}}`, tc.body)
				} else {
					fmt.Fprint(w, tc.body)
				}
			})
			got, resp, err := tc.call()
			if err != nil {
				t.Fatal(err)
			}
			if got == nil || got.Free != "0" || got.InitialPurchaseRequirement != "2" || got.SubsequentPurchaseRequirement != "1" {
				t.Fatalf("unexpected license: %+v", got)
			}
			if tc.name == "Drives/List" || tc.name == "LibraryDrives/List" || tc.name == "Servers/List" || tc.name == "Snapshots/List" {
				if resp.Meta == nil || resp.Meta.TotalCount != 1 {
					t.Fatalf("lost metadata: %+v", resp)
				}
			}
		})
	}
}

func TestLicenseWireNestedSemantics(t *testing.T) {
	for _, input := range []string{
		`{}`, `null`,
		`{"licenses":[{"license":{"free":"new"},"LICENSE":{"free":null,"name":"new-name"}}]}`,
		`{"licenses":null}`, `{"licenses":[]}`,
		`{"licenses":[{"license":null}]}`,
		`{"licenses":[{"license":{"free":"first"}}],"LICENSES":[{"license":{"name":"next"}}]}`,
	} {
		t.Run(input, func(t *testing.T) {
			seed := func() Drive {
				return Drive{UUID: "id", Licenses: []DriveLicense{{Amount: 2, License: &License{Free: "old", Name: "old-name"}}}}
			}
			want := seed()
			got := driveResponse(seed())
			if err := json.Unmarshal([]byte(input), &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(input), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(Drive(got), want) {
				t.Fatalf("wire differs from public baseline: got %+v want %+v", got, want)
			}
		})
	}
	for _, input := range []string{`{}`, `null`, `{"drive":null}`, `{"drive":{"name":"new"},"DRIVE":{"uuid":"new-id"}}`} {
		seed := func() Snapshot {
			return Snapshot{Drive: &Drive{UUID: "id", Name: "old", Licenses: []DriveLicense{{License: &License{Free: "old"}}}}}
		}
		want := seed()
		got := snapshotResponse(seed())
		if err := json.Unmarshal([]byte(input), &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(Snapshot(got), want) {
			t.Fatalf("snapshot differs for %s", input)
		}
	}
	for _, input := range []string{
		`{"licenses":[{"license":{"free":false,"free":1}}]}`,
		`{"licenses":[{"license":{"free":1,"free":false}}]}`,
	} {
		var got driveResponse
		if err := json.Unmarshal([]byte(input), &got); err == nil {
			t.Fatalf("accepted invalid nested occurrence: %s", input)
		}
	}
}

func TestLicensePublicMethodSetsUnchanged(t *testing.T) {
	for _, v := range []interface{}{&License{}, &DriveLicense{}, &Drive{}, &LibraryDrive{}, &ServerDrive{}, &Server{}, &Snapshot{}} {
		if _, ok := v.(json.Unmarshaler); ok {
			t.Fatalf("public type %T acquired UnmarshalJSON", v)
		}
	}
	var license License
	if err := json.Unmarshal([]byte(`{"free":0}`), &license); err == nil {
		t.Fatal("direct public decode must retain original string-only behavior")
	}
}

func TestLicenseResponseEnvelopePresence(t *testing.T) {
	for _, input := range []string{`{}`, `{"objects":null}`, `{"objects":[]}`} {
		var root drivesRoot
		if err := json.Unmarshal([]byte(input), &root); err != nil {
			t.Fatal(err)
		}
		values := root.values()
		if (values == nil) != (input != `{"objects":[]}`) {
			t.Fatalf("lost nil/empty distinction for %s", input)
		}
	}
}

func TestLicenseHTTPRejectsInvalidOccurrences(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/drives/id/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"licenses":[{"license":{"free":false,"FREE":1}}]}`)
	})
	got, _, err := client.Drives.Get(ctx, "id")
	if err == nil || got != nil {
		t.Fatalf("expected failed response, got %+v, %v", got, err)
	}
}
