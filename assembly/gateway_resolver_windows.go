//go:build windows

package assembly

import (
	"os/exec"
	"syscall"
)

// configureDetachedProcess sets the DETACHED_PROCESS creation flag so
// the spawned aasm survives the parent Go process exit on Windows.
func configureDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008}
}
