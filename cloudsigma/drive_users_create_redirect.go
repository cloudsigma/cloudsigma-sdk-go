package cloudsigma

import "errors"

// ErrDriveUsersCreateRedirect indicates a refused RDS drive user-count create redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrDriveUsersCreateRedirect = errors.New("drive users create redirect refused; outcome unknown")
