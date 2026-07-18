package assembly

import (
	"context"
	"errors"
	"fmt"
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

// TestEvictionFailsClosedForPausedWaiter is the AAASM-4811 regression: when a
// still-paused op is evicted to make room under maxOpControlSlots, its blocked
// waiter must fail closed with ErrOpControlUnavailable — not wake to a silent
// allow. The evicted slot is deleted from the map, so no resume signal can ever
// reach that waiter again; yielding nil (allow) would let a paused op run just
// because the gateway pushed enough distinct opIDs to evict it.
func TestEvictionFailsClosedForPausedWaiter(t *testing.T) {
	sub, stream, _ := newSubscriber(t)

	// Pause the op we will later force out of the slot map.
	stream.push(msg("op-evict", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-evict") }, time.Second)

	// A waiter blocks on the paused op.
	done := make(chan error, 1)
	go func() {
		done <- sub.WaitForOp(context.Background(), "op-evict")
	}()

	// Confirm it is genuinely blocked (and thus registered as a waiter) before
	// eviction flushes the slot.
	select {
	case err := <-done:
		t.Fatalf("WaitForOp returned %v while op was paused; want it to block", err)
	case <-time.After(50 * time.Millisecond):
	}

	// Push maxOpControlSlots distinct fresh ops. "op-evict" holds the oldest
	// slot; the maxOpControlSlots-th brand-new op trips the cap and evicts the
	// oldest entry — op-evict — while it is still paused.
	for i := 0; i < maxOpControlSlots; i++ {
		stream.push(msg(fmt.Sprintf("filler-%d", i), pb.OpControlSignal_OP_CONTROL_SIGNAL_RESUME, uint64(i+1)))
	}

	select {
	case err := <-done:
		if !errors.Is(err, ErrOpControlUnavailable) {
			t.Fatalf("WaitForOp returned %v; want ErrOpControlUnavailable on eviction while paused (must not yield an allow)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForOp did not wake after the paused op was evicted")
	}
}

// TestEvictionExemptsTerminatedSlot is the AAASM-4832 regression (sibling of the
// AAASM-4811 paused-eviction fix): a TERMINATED op's verdict must survive
// eviction pressure. The terminated slot holds the OLDEST position, so the old
// oldest-first eviction would have dropped it once the cap tripped — after which
// a later WaitForOp for the same opID would create a fresh, non-terminated slot
// and return nil (proceed) instead of the OpTerminatedError the operator issued.
// A terminated slot must be skipped as an eviction victim, so its verdict still
// survives maxOpControlSlots later distinct opIDs.
func TestEvictionExemptsTerminatedSlot(t *testing.T) {
	sub, stream, _ := newSubscriber(t)

	// Terminate the op that holds the oldest slot.
	stream.push(msg("op-term", pb.OpControlSignal_OP_CONTROL_SIGNAL_TERMINATE, 0))
	waitFor(t, func() bool { return sub.IsTerminated("op-term") }, time.Second)

	// Push maxOpControlSlots distinct fresh ops to drive the cap past its limit.
	// op-term holds the oldest slot; the old oldest-first policy would evict it
	// and lose its terminate verdict. It must be skipped instead.
	for i := 0; i < maxOpControlSlots; i++ {
		stream.push(msg(fmt.Sprintf("filler-%d", i), pb.OpControlSignal_OP_CONTROL_SIGNAL_RESUME, uint64(i+1)))
	}

	// Messages are processed in order, so once this sentinel is paused every
	// filler before it has been dispatched and the cap has tripped.
	stream.push(msg("sentinel", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, uint64(maxOpControlSlots+1)))
	waitFor(t, func() bool { return sub.IsPaused("sentinel") }, 2*time.Second)

	err := sub.WaitForOp(context.Background(), "op-term")
	var termErr *OpTerminatedError
	if !errors.As(err, &termErr) {
		t.Fatalf("WaitForOp(op-term) = %v; want *OpTerminatedError to survive eviction (AAASM-4832)", err)
	}
}

// TestEvictionBoundsTerminatedSlotsUnderFlood is the AAASM-4843 regression: a
// gateway spamming TERMINATE for endless unique op_ids must not grow the slot map
// without bound. The AAASM-4832 fix exempts terminated slots from eviction, but if
// EVERY live slot is terminated that exemption would evict nothing and let the map
// grow forever. Under such flood pressure the oldest terminated verdict is dropped
// so the map stays hard-bounded at maxOpControlSlots — yet a RECENTLY terminated
// op's verdict must still survive (only the oldest is sacrificed).
func TestEvictionBoundsTerminatedSlotsUnderFlood(t *testing.T) {
	sub, stream, _ := newSubscriber(t)

	const flood = maxOpControlSlots + 512
	for i := 0; i < flood; i++ {
		stream.push(msg(fmt.Sprintf("term-%d", i), pb.OpControlSignal_OP_CONTROL_SIGNAL_TERMINATE, uint64(i+1)))
	}

	// Messages are processed in order, so once the newest op is observed
	// terminated, every earlier TERMINATE has been dispatched and the cap has
	// been exercised.
	newest := fmt.Sprintf("term-%d", flood-1)
	waitFor(t, func() bool { return sub.IsTerminated(newest) }, 5*time.Second)

	sub.mu.Lock()
	size := len(sub.ops)
	sub.mu.Unlock()
	if size > maxOpControlSlots {
		t.Fatalf("slot map grew to %d under a TERMINATE flood; want <= %d (must stay bounded)", size, maxOpControlSlots)
	}

	// The most recently terminated op must still carry its verdict — only the
	// OLDEST terminated slots are sacrificed to hold the cap.
	err := sub.WaitForOp(context.Background(), newest)
	var termErr *OpTerminatedError
	if !errors.As(err, &termErr) {
		t.Fatalf("WaitForOp(%s) = %v; want *OpTerminatedError to survive the flood (a recent verdict must not be dropped)", newest, err)
	}
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
