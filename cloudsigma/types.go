package cloudsigma

// DriveLicense represents a CloudSigma license attached to the drive.
type DriveLicense struct {
	Amount  int           `json:"amount,omitempty"`
	License *License      `json:"license,omitempty"`
	User    *ResourceLink `json:"user,omitempty"`
}

// Meta represents an object with meta information about the request.
type Meta struct {
	Limit      int `json:"limit,omitempty"`
	Offset     int `json:"offset,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
}

// ResourceLink represents a link to other CloudSigma resource.
type ResourceLink struct {
	ResourceURI string `json:"resource_uri,omitempty"`
	UUID        string `json:"uuid,omitempty"`
}
