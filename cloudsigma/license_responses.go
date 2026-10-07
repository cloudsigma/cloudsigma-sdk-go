package cloudsigma

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// These private response types add numeric license compatibility only to SDK
// service responses. Public model method sets and request encoding are unchanged.
// Only licenseResponse (scalar fields) promises error atomicity. Containing
// wrappers seed shallow aliases to preserve normal merge semantics; maps and
// unrelated pointers can be partially mutated on error, as with encoding/json.
// Service methods decode fresh destinations and do not return them on failure.
type licenseResponse License

// licenseQuantity decodes each occurrence immediately so duplicate keys cannot
// hide invalid values and null preserves the most recently decoded value.
type licenseQuantity struct {
	name  string
	value string
}

func (q *licenseQuantity) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &q.value); err != nil {
			return fmt.Errorf("license.%s: %w", q.name, err)
		}
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("license.%s: %w", q.name, err)
	}
	q.value = number.String()
	return nil
}

func (l *licenseResponse) UnmarshalJSON(data []byte) error {
	type licenseAlias License
	decoded := licenseAlias(*l)
	wire := struct {
		*licenseAlias
		Free       licenseQuantity `json:"free"`
		Initial    licenseQuantity `json:"initial_purchase_requirement"`
		Subsequent licenseQuantity `json:"subsequent_purchase_requirement"`
	}{
		licenseAlias: &decoded,
		Free:         licenseQuantity{"free", l.Free},
		Initial:      licenseQuantity{"initial_purchase_requirement", l.InitialPurchaseRequirement},
		Subsequent:   licenseQuantity{"subsequent_purchase_requirement", l.SubsequentPurchaseRequirement},
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Free = wire.Free.value
	decoded.InitialPurchaseRequirement = wire.Initial.value
	decoded.SubsequentPurchaseRequirement = wire.Subsequent.value
	*l = licenseResponse(decoded)
	return nil
}

type driveLicenseResponse DriveLicense

func (r *driveLicenseResponse) UnmarshalJSON(data []byte) error {
	type alias DriveLicense
	decoded := alias(*r)
	var nested *licenseResponse
	if decoded.License != nil {
		value := licenseResponse(*decoded.License)
		nested = &value
	}
	wire := struct {
		*alias
		License *licenseResponse `json:"license"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.License = (*License)(wire.License)
	*r = driveLicenseResponse(decoded)
	return nil
}

type driveResponse Drive

func (r *driveResponse) UnmarshalJSON(data []byte) error {
	type alias Drive
	decoded := alias(*r)
	var nested []driveLicenseResponse
	if decoded.Licenses != nil {
		nested = make([]driveLicenseResponse, len(decoded.Licenses))
		for i := range nested {
			nested[i] = driveLicenseResponse(decoded.Licenses[i])
		}
	}
	wire := struct {
		*alias
		Licenses []driveLicenseResponse `json:"licenses"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Licenses = nil
	if wire.Licenses != nil {
		decoded.Licenses = make([]DriveLicense, len(wire.Licenses))
		for i := range wire.Licenses {
			decoded.Licenses[i] = DriveLicense(wire.Licenses[i])
		}
	}
	*r = driveResponse(decoded)
	return nil
}

type libraryDriveResponse LibraryDrive

func (r *libraryDriveResponse) UnmarshalJSON(data []byte) error {
	type alias LibraryDrive
	decoded := alias(*r)
	var nested []driveLicenseResponse
	if decoded.Licenses != nil {
		nested = make([]driveLicenseResponse, len(decoded.Licenses))
		for i := range nested {
			nested[i] = driveLicenseResponse(decoded.Licenses[i])
		}
	}
	wire := struct {
		*alias
		Licenses []driveLicenseResponse `json:"licenses"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Licenses = nil
	if wire.Licenses != nil {
		decoded.Licenses = make([]DriveLicense, len(wire.Licenses))
		for i := range wire.Licenses {
			decoded.Licenses[i] = DriveLicense(wire.Licenses[i])
		}
	}
	*r = libraryDriveResponse(decoded)
	return nil
}

type serverDriveResponse ServerDrive

func (r *serverDriveResponse) UnmarshalJSON(data []byte) error {
	type alias ServerDrive
	decoded := alias(*r)
	var nested *driveResponse
	if decoded.Drive != nil {
		value := driveResponse(*decoded.Drive)
		nested = &value
	}
	wire := struct {
		*alias
		Drive *driveResponse `json:"drive"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Drive = (*Drive)(wire.Drive)
	*r = serverDriveResponse(decoded)
	return nil
}

type serverResponse Server

func (r *serverResponse) UnmarshalJSON(data []byte) error {
	type alias Server
	decoded := alias(*r)
	var nested []serverDriveResponse
	if decoded.Drives != nil {
		nested = make([]serverDriveResponse, len(decoded.Drives))
		for i := range nested {
			nested[i] = serverDriveResponse(decoded.Drives[i])
		}
	}
	wire := struct {
		*alias
		Drives []serverDriveResponse `json:"drives"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Drives = nil
	if wire.Drives != nil {
		decoded.Drives = make([]ServerDrive, len(wire.Drives))
		for i := range wire.Drives {
			decoded.Drives[i] = ServerDrive(wire.Drives[i])
		}
	}
	*r = serverResponse(decoded)
	return nil
}

type snapshotResponse Snapshot

func (r *snapshotResponse) UnmarshalJSON(data []byte) error {
	type alias Snapshot
	decoded := alias(*r)
	var nested *driveResponse
	if decoded.Drive != nil {
		value := driveResponse(*decoded.Drive)
		nested = &value
	}
	wire := struct {
		*alias
		Drive *driveResponse `json:"drive"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Drive = (*Drive)(wire.Drive)
	*r = snapshotResponse(decoded)
	return nil
}
