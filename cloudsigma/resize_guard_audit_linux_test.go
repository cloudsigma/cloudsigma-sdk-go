//go:build linux && cloudsigma_guard_audit
// +build linux,cloudsigma_guard_audit

package cloudsigma

import (
	"syscall"
	"testing"
)

func TestResizeGuardAudit(t *testing.T) {
	for _, family := range []int{syscall.AF_UNIX, syscall.AF_INET, syscall.AF_INET6} {
		fd, err := syscall.Socket(family, syscall.SOCK_STREAM, 0)
		if fd >= 0 {
			syscall.Close(fd)
		}
		if err != syscall.EPERM {
			t.Fatalf("guard absent: %v", err)
		}
	}
	t.Log("GUARD_RUNTIME_PASS")
}
