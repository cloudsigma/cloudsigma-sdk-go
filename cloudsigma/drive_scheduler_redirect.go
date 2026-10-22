package cloudsigma

import "errors"

// ErrDriveSetSchedulerRedirect indicates a refused scheduler-binding redirect.
// The initial POST outcome is unknown; callers must not blindly retry.
var ErrDriveSetSchedulerRedirect = errors.New("drive set scheduler redirect refused; outcome unknown")
