package cloudsigma

import "errors"

// ErrDriveResizeDefinitionRedirect indicates a refused full-definition resize redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrDriveResizeDefinitionRedirect = errors.New("drive resize definition redirect refused; outcome unknown")
