package assembly

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"syscall"
	"time"
)

// SidecarClient is the local gRPC sidecar contract used by the SDK.
type SidecarClient interface {
	Ping(ctx context.Context) error
}

// ErrSidecarUnavailable indicates the local sidecar cannot be reached.
var ErrSidecarUnavailable = errors.New("assembly: sidecar unavailable")

const defaultStopTimeout = 5 * time.Second

// defaultHealthTimeout bounds Healthy when the caller's context carries no
// deadline of its own. It exists because the caller's context is otherwise the
// only exit condition for the poll loop: a caller following the documented
// quick-start (which uses context.Background(), a context that never cancels)
// would hang forever whenever the sidecar address is empty or unreachable. 5s
// mirrors defaultStopTimeout — a bounded, diagnosable failure beats a silent hang.
const defaultHealthTimeout = 5 * time.Second

// Sidecar manages the lifecycle of a local sidecar subprocess.
type Sidecar struct {
	binaryPath    string
	address       string
	cmd           *exec.Cmd
	stopTimeout   time.Duration
	healthTimeout time.Duration
}

// NewSidecar creates a Sidecar for the given binary and listen address.
func NewSidecar(binaryPath, address string) *Sidecar {
	return &Sidecar{
		binaryPath:    binaryPath,
		address:       address,
		stopTimeout:   defaultStopTimeout,
		healthTimeout: defaultHealthTimeout,
	}
}

// Start launches the sidecar binary as a subprocess.
func (s *Sidecar) Start(ctx context.Context) error {
	if s.cmd != nil && s.cmd.Process != nil {
		return fmt.Errorf("assembly: sidecar already started")
	}

	s.cmd = exec.CommandContext(ctx, s.binaryPath, "--listen", s.address)
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("assembly: failed to start sidecar: %w", err)
	}

	return nil
}

// Stop sends SIGTERM to the sidecar process and waits for graceful shutdown.
// If the process does not exit within the stop timeout, it sends SIGKILL.
func (s *Sidecar) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("assembly: failed to signal sidecar: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case err := <-done:
		return ignoreSignalExit(err)
	case <-time.After(s.stopTimeout):
		if killErr := s.cmd.Process.Kill(); killErr != nil {
			return fmt.Errorf("assembly: failed to kill sidecar: %w", killErr)
		}
		<-done
		return nil
	}
}

const healthPollInterval = 50 * time.Millisecond

// Healthy polls the sidecar address via TCP until it accepts connections, the
// context is cancelled, or an internal deadline elapses.
//
// It fails fast with a clear error on an empty address rather than entering the
// poll loop. When ctx carries no deadline of its own, Healthy applies
// defaultHealthTimeout so a caller passing a non-cancelling context (e.g.
// context.Background(), as the documented quick-start does) still gets a
// bounded, diagnosable failure instead of a silent hang; a caller-supplied
// deadline is honoured as-is. The timeout error names the address so the caller
// can tell the sidecar address was the cause.
func (s *Sidecar) Healthy(ctx context.Context) error {
	if s.address == "" {
		return fmt.Errorf("assembly: cannot check health of sidecar with empty address")
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.healthTimeout)
		defer cancel()
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("assembly: sidecar health check timed out for address %q: %w", s.address, ctx.Err())
		default:
			conn, err := net.DialTimeout("tcp", s.address, healthPollInterval)
			if err == nil {
				_ = conn.Close()
				return nil
			}
			time.Sleep(healthPollInterval)
		}
	}
}

// ignoreSignalExit returns nil when the process exited due to a signal,
// which is the expected outcome after sending SIGTERM or SIGKILL.
func ignoreSignalExit(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return err
}

// connectToLocalSidecar is the fallthrough boot path taken when neither a
// sidecar address nor a managed binary is configured. There is no local sidecar
// to discover here (the pure-Go build has no native transport), so rather than
// returning a bare, opaque sentinel it wraps ErrSidecarUnavailable with the
// concrete configuration a caller must supply to reach a working setup. The
// sentinel stays wrapped so errors.Is(err, ErrSidecarUnavailable) still holds.
func connectToLocalSidecar(ctx context.Context, address string) (SidecarClient, error) {
	_, _ = ctx, address
	return nil, fmt.Errorf(
		"%w: pass assembly.WithSidecarAddress(<gateway gRPC addr, e.g. 127.0.0.1:50051>) "+
			"pointing at a running gateway, or WithSidecarBinary to have the SDK manage one",
		ErrSidecarUnavailable,
	)
}
