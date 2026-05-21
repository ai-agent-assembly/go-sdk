// Runtime auto-detection and lifecycle management for the `aasm` sidecar
// (F115 / AAASM-1205).
//
// The InitAssembly() exported here is intentionally distinct from the
// existing gateway-based Init(ctx, opts...) — InitAssembly only ensures
// the local sidecar binary is running; register-and-connect is performed
// by Init() against the now-reachable sidecar.

package assembly

import "errors"

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
