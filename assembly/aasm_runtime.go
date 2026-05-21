// Runtime auto-detection and lifecycle management for the `aasm` sidecar
// (F115 / AAASM-1205).
//
// The InitAssembly() exported here is intentionally distinct from the
// existing gateway-based Init(ctx, opts...) — InitAssembly only ensures
// the local sidecar binary is running; register-and-connect is performed
// by Init() against the now-reachable sidecar.

package assembly

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	// BinaryName is the on-disk name of the Rust sidecar binary.
	BinaryName = "aasm"
	// DefaultPort is the localhost TCP port the sidecar listens on.
	DefaultPort = 7878
	// DefaultRuntimeHost is the localhost host the SDK probes.
	DefaultRuntimeHost = "127.0.0.1"
	// RuntimeLogFilename is the file Sidecar stdout/stderr is appended to.
	RuntimeLogFilename = ".aasm-runtime.log"
	// UserLocalBin is the conventional curl-installer destination.
	UserLocalBin = ".local/bin"
	// DockerBaseBin is the absolute path used by Docker base images.
	DockerBaseBin = "/usr/local/bin"
)

// InstallHint is the copy-paste message embedded in ErrBinaryNotFound.
const InstallHint = `agent-assembly runtime not found.
  Install with: brew install agent-assembly/tap/aasm
  Or manually:  curl -fsSL https://get.agent-assembly.io | sh
               go install github.com/AI-agent-assembly/agent-assembly/cmd/aasm@latest`

// ErrBinaryNotFound is the sentinel error returned by InitAssembly when
// no `aasm` binary is found across any of the supported install paths.
// Callers can check `errors.Is(err, assembly.ErrBinaryNotFound)`.
var ErrBinaryNotFound = errors.New(InstallHint)

// findAasmBinary locates the `aasm` binary across the 3 supported install
// paths: $PATH (Homebrew, `go install`, curl-installer) → ~/.local/bin/aasm
// (curl installer alt) → /usr/local/bin/aasm (Docker base image).
// Returns the first existing match or ErrBinaryNotFound when none exist.
func findAasmBinary() (string, error) {
	if path, err := exec.LookPath(BinaryName); err == nil {
		return path, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, UserLocalBin, BinaryName)
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	docker := filepath.Join(DockerBaseBin, BinaryName)
	if info, statErr := os.Stat(docker); statErr == nil && !info.IsDir() {
		return docker, nil
	}
	return "", ErrBinaryNotFound
}

// isRunning returns true iff a local TCP listener accepts a connect on
// DefaultRuntimeHost:port within 100 ms. Any dial error (refused,
// timeout, unreachable) is treated as no sidecar — the lifecycle
// orchestrator uses this to skip startRuntime() when the sidecar is
// already up (idempotent re-init).
func isRunning(port int) bool {
	addr := fmt.Sprintf("%s:%d", DefaultRuntimeHost, port)
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// startRuntime spawns `aasm serve --port <port>` as a detached background
// process. Stdout/stderr are appended to <cwd>/.aasm-runtime.log so the
// sidecar outlives the parent. SysProcAttr.Setsid puts the child in its
// own session so SIGHUP from the parent's controlling terminal does not
// take it down. Returns the *os.Process handle; the caller does not wait.
func startRuntime(binaryPath string, port int) (*os.Process, error) {
	logPath := filepath.Join(".", RuntimeLogFilename)
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open runtime log: %w", err)
	}
	cmd := exec.Command(binaryPath, "serve", "--port", strconv.Itoa(port))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("spawn aasm sidecar: %w", err)
	}
	return cmd.Process, nil
}
