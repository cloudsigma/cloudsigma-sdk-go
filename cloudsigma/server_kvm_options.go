package cloudsigma

import "context"

// ServerKVMOptions provides presence-aware KVM CPU topology settings for edits.
// Nil fields are omitted; nonnil fields send true or false explicitly.
// Public KVM documentation recommends CPUsInsteadOfCores=false for Windows.
// Changes to these fields on a running server are ignored by the API.
//
// CloudSigma API docs: https://docs.cloudsigma.com/en/latest/servers_kvm.html
// https://docs.cloudsigma.com/en/latest/servers.html#editing
// This separate type leaves the legacy Server model and its encoding unchanged.
type ServerKVMOptions struct {
	CPUsInsteadOfCores *bool
	EnableNuma         *bool
}

// UpdateKVMOptions edits only the specified KVM CPU topology settings.
// Nil options are rejected; empty options send an empty object. The latter is
// representation behavior, not a guarantee that an empty edit is useful.
// The returned server is the response snapshot, not proof of runtime application.
func (s *ServersService) UpdateKVMOptions(ctx context.Context, id string, options *ServerKVMOptions) (*Server, *Response, error) {
	if options == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}
	definition := ServerDefinition{}
	for key, value := range map[string]*bool{
		"cpus_instead_of_cores": options.CPUsInsteadOfCores,
		"enable_numa":           options.EnableNuma,
	} {
		if value != nil {
			if *value {
				definition[key] = []byte("true")
			} else {
				definition[key] = []byte("false")
			}
		}
	}
	return s.UpdateDefinition(ctx, id, definition)
}
