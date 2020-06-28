package cloudsigma

// Meta represents an object with meta information about the request.
type Meta struct {
	Limit      int `json:"limit,omitempty"`
	Offset     int `json:"offset,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
}

// Owner represents a CloudSigma ownership of the resource.
type Owner struct {
	ResourceURI string `json:"resource_uri,omitempty"`
	UUID        string `json:"uuid"`
}
