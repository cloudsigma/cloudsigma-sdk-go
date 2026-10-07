package cloudsigma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestLicenseUnmarshalQuantities(t *testing.T) {
	for _, field := range []string{"free", "initial_purchase_requirement", "subsequent_purchase_requirement"} {
		for _, tc := range []struct{ name, input, want string }{
			{"integer", "12", "12"}, {"zero", "0", "0"},
			{"large", "9007199254740993", "9007199254740993"},
			{"decimal", "1.250", "1.250"}, {"exponent", "1e+3", "1e+3"},
			{"negative", "-1", "-1"}, {"legacy", `"0012"`, "0012"},
			{"legacy_text", `"unlimited"`, "unlimited"}, {"empty", `""`, ""},
			{"null", "null", ""},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				var got licenseResponse
				if err := json.Unmarshal([]byte(fmt.Sprintf(`{"%s":%s,"name":"windows","burstable":true}`, field, tc.input)), &got); err != nil {
					t.Fatal(err)
				}
				values := map[string]string{"free": got.Free, "initial_purchase_requirement": got.InitialPurchaseRequirement, "subsequent_purchase_requirement": got.SubsequentPurchaseRequirement}
				if values[field] != tc.want || got.Name != "windows" || !got.Burstable {
					t.Fatalf("unexpected license: %+v", got)
				}
			})
		}
		for _, bad := range []string{"true", "false", "[]", "{}"} {
			t.Run(field+"/reject/"+bad, func(t *testing.T) {
				got := licenseResponse{Free: "old", Name: "old"}
				err := json.Unmarshal([]byte(fmt.Sprintf(`{"name":"new","%s":%s}`, field, bad)), &got)
				if err == nil || !strings.Contains(err.Error(), field) {
					t.Fatalf("expected field error, got %v", err)
				}
				if got.Free != "old" || got.Name != "old" {
					t.Fatalf("error changed receiver: %+v", got)
				}
			})
		}
	}
}

func TestLicenseUnmarshalCompatibility(t *testing.T) {
	initial := licenseResponse{Free: "old-free", InitialPurchaseRequirement: "old-initial", SubsequentPurchaseRequirement: "old-subsequent", Name: "old"}
	for _, input := range []string{`{}`, `null`, `{"free":null,"initial_purchase_requirement":null,"subsequent_purchase_requirement":null}`} {
		got := initial
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatal(err)
		}
		if got != initial {
			t.Fatalf("%s changed existing values: %+v", input, got)
		}
	}
	for _, input := range []string{`{"free":01}`, `{"free":1.}`, `{"name":12}`, `[]`} {
		var got licenseResponse
		if err := json.Unmarshal([]byte(input), &got); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	var got licenseResponse
	if err := json.Unmarshal([]byte(`{"free":0,"initial_purchase_requirement":2,"subsequent_purchase_requirement":"3","future_field":true}`), &got); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"free":"0","initial_purchase_requirement":"2","subsequent_purchase_requirement":"3","user_metric":""}` {
		t.Fatalf("changed JSON encoding: %s", encoded)
	}
}

func TestLicenseNestedDriveRegression(t *testing.T) {
	var drive driveResponse
	err := json.Unmarshal([]byte(`{"licenses":[{"license":{"free":0,"initial_purchase_requirement":2,"subsequent_purchase_requirement":1}}]}`), &drive)
	if err != nil {
		t.Fatal(err)
	}
	if len(drive.Licenses) != 1 || drive.Licenses[0].License == nil || drive.Licenses[0].License.Free != "0" {
		t.Fatalf("unexpected drive licenses: %+v", drive.Licenses)
	}
}

func TestLicenseNumericResponsePaths(t *testing.T) {
	setup()
	defer teardown()
	license := `{"name":"windows","free":0,"initial_purchase_requirement":2,"subsequent_purchase_requirement":1}`
	mux.HandleFunc("/licenses/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"objects":[%s],"meta":{"total_count":1}}`, license)
	})
	mux.HandleFunc("/drives/windows/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"uuid":"windows","licenses":[{"amount":1,"license":%s}]}`, license)
	})
	licenses, _, err := client.Licenses.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(licenses) != 1 || licenses[0].Free != "0" || licenses[0].InitialPurchaseRequirement != "2" || licenses[0].SubsequentPurchaseRequirement != "1" {
		t.Fatalf("unexpected licenses: %+v", licenses)
	}
	drive, _, err := client.Drives.Get(ctx, "windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(drive.Licenses) != 1 || drive.Licenses[0].License == nil || *drive.Licenses[0].License != licenses[0] {
		t.Fatalf("unexpected drive: %+v", drive)
	}
	var library libraryDriveResponse
	if err := json.Unmarshal([]byte(fmt.Sprintf(`{"licenses":[{"license":%s}]}`, license)), &library); err != nil {
		t.Fatal(err)
	}
	if len(library.Licenses) != 1 || library.Licenses[0].License == nil || *library.Licenses[0].License != licenses[0] {
		t.Fatalf("unexpected library licenses: %+v", library.Licenses)
	}
}

func TestLicenseDuplicateQuantities(t *testing.T) {
	for _, field := range []string{"free", "initial_purchase_requirement", "subsequent_purchase_requirement"} {
		for _, keys := range [][2]string{{field, field}, {strings.ToUpper(field), field}, {field, strings.ToUpper(field)}} {
			for _, tc := range []struct {
				first, second, want string
				fail                bool
			}{
				{`"new"`, `null`, "new", false}, {`12`, `null`, "12", false},
				{`false`, `"new"`, "", true}, {`"new"`, `false`, "", true},
			} {
				input := fmt.Sprintf(`{"%s":%s,"%s":%s}`, keys[0], tc.first, keys[1], tc.second)
				t.Run(input, func(t *testing.T) {
					original := licenseResponse{Free: "old", InitialPurchaseRequirement: "old", SubsequentPurchaseRequirement: "old", Name: "old", Burstable: true}
					got := original
					err := json.Unmarshal([]byte(input), &got)
					if tc.fail {
						if err == nil || !strings.Contains(err.Error(), field) {
							t.Fatalf("expected field error, got %v", err)
						}
						if got != original {
							t.Fatalf("error mutated receiver: %+v", got)
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					values := map[string]string{"free": got.Free, "initial_purchase_requirement": got.InitialPurchaseRequirement, "subsequent_purchase_requirement": got.SubsequentPurchaseRequirement}
					if values[field] != tc.want {
						t.Fatalf("got %+v, want %q", got, tc.want)
					}
				})
			}
		}
	}
}

func TestLicenseUnmarshalAtomicErrors(t *testing.T) {
	original := licenseResponse{Free: "old", InitialPurchaseRequirement: "initial", SubsequentPurchaseRequirement: "subsequent", Name: "original", Burstable: true}
	for _, input := range []string{
		`{"free":12,"name":false}`, `{"name":"new","free":12,"initial_purchase_requirement":false}`,
		`{"free":12,"subsequent_purchase_requirement":[]}`, `{"burstable":"wrong","free":12}`,
	} {
		got := original
		if err := json.Unmarshal([]byte(input), &got); err == nil {
			t.Fatalf("accepted %s", input)
		}
		if got != original {
			t.Fatalf("%s mutated receiver: %+v", input, got)
		}
	}
}

// Public model decoding must retain its original method set and behavior.
func TestLicensePublicDecodingCompatibility(t *testing.T) {
	type Extended struct {
		License
		Extra string `json:"extra"`
	}
	var extended Extended
	if err := json.Unmarshal([]byte(`{"name":"n","extra":"x"}`), &extended); err != nil {
		t.Fatal(err)
	}
	if extended.Name != "n" || extended.Extra != "x" {
		t.Fatalf("unexpected public embedding behavior: %+v", extended)
	}
	var license License
	decoder := json.NewDecoder(strings.NewReader(`{"unknown":true}`))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&license); err == nil {
		t.Fatal("public License must retain strict decoding")
	}

}

func TestLicenseFixtureCredentialsRequest(t *testing.T) {
	// The provider regression uses this exact deterministic provider. Request
	// construction must succeed before its in-memory transport can run.
	fixtureClient := NewClient(NewTokenCredentialsProvider("fixture-only-not-a-real-token"))
	req, err := fixtureClient.NewRequest(http.MethodGet, "drives/windows/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != "Bearer fixture-only-not-a-real-token" {
		t.Fatal("fixture credential was not applied")
	}
}
