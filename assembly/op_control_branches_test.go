package assembly

import (
	"errors"
	"testing"
	"time"

	pb "github.com/ai-agent-assembly/go-sdk/internal/proto"
)

// TestReadLoop_TransportErrorMarksStreamDead covers the readLoop branch that
// handles a non-EOF, non-cancel transport error: the stream is marked dead
// and any blocked waiters are released. This is distinct from the clean
// io.EOF shutdown path already covered elsewhere.
func TestReadLoop_TransportErrorMarksStreamDead(t *testing.T) {
	sub, stream, _ := newSubscriber(t)

	// Inject a transport-level failure (not io.EOF / context.Canceled) and
	// wake the reader so readLoop takes the error branch.
	stream.mu.Lock()
	stream.err = errors.New("rpc transport reset")
	stream.cond.Broadcast()
	stream.mu.Unlock()

	waitFor(t, func() bool { return !sub.StreamAlive() }, time.Second)
}

// TestDispatch_UnspecifiedSignalIsIgnored covers the default arm of the
// dispatch switch: an UNSPECIFIED signal must not pause or terminate the op.
func TestDispatch_UnspecifiedSignalIsIgnored(t *testing.T) {
	sub, stream, _ := newSubscriber(t)

	stream.push(msg("op-unspec", pb.OpControlSignal_OP_CONTROL_SIGNAL_UNSPECIFIED, 1))

	// Give the reader a moment to process, then assert no state changed.
	waitFor(t, func() bool {
		return !sub.IsPaused("op-unspec") && !sub.IsTerminated("op-unspec")
	}, time.Second)
}
