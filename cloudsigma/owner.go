package cloudsigma

// Owner represents a CloudSigma ownership of the resource.
type Owner struct {
	ResourceURI string `json:"resource_uri,omitempty"`
	UUID        string `json:"uuid"`
}
