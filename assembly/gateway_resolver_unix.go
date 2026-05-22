//go:build !windows

package assembly

import (
	"os/exec"
	"syscall"
)

// configureDetachedProcess marks the child process to start a new
// session so it survives the parent Go process exit — matching the
// daemon hand-off semantics of subprocess.Popen(start_new_session=True)
// in the Python SDK.
func configureDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
