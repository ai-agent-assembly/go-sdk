//go:build !windows

package assembly

import (
	"os/exec"
	"testing"
)

// TestConfigureDetachedProcess_SetsSid covers the unix detach helper: it sets
// SysProcAttr.Setsid so the spawned gateway starts its own session and
// survives the parent Go process exit.
func TestConfigureDetachedProcess_SetsSid(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("/bin/true")
	configureDetachedProcess(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be set")
	}
	if !cmd.SysProcAttr.Setsid {
		t.Fatal("expected Setsid to be true so the child detaches into its own session")
	}
}
