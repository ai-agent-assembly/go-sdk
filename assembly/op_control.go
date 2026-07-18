// Package assembly: gateway → SDK op-control consumer (AAASM-1422 PR-G / AAASM-1656).
//
// OpControlSubscriber subscribes to PolicyService.OpControlStream and exposes
// a per-op_id cooperative-pause / fast-fail-terminate state machine through
// WaitForOp.
//
// State machine per op_id:
//   - OP_CONTROL_SIGNAL_PAUSE     → WaitForOp blocks until RESUME arrives.
//   - OP_CONTROL_SIGNAL_RESUME    → WaitForOp returns nil immediately.
//   - OP_CONTROL_SIGNAL_TERMINATE → WaitForOp returns *OpTerminatedError.
//
// Signals that arrive for an op_id no one is currently awaiting are buffered
// into the per-op slot so the next WaitForOp sees them.
//
// Out of scope for PR-G (deferred):
//   - Reconnection / heartbeat on stream close (caller observes StreamAlive
//     and re-instantiates if desired).
//   - Auto-wiring into the existing GatewayClient / interceptor.go hooks
//     (separate sub-task when the adapter surface is stable).

package assembly

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc"

	pb "github.com/ai-agent-assembly/go-sdk/internal/proto"
)

// maxOpControlSlots caps OpControlSubscriber.ops to prevent unbounded growth
// from a compromised gateway pushing endless unique opIDs. When the cap is
// hit, the OLDEST entry (insertion order) is evicted. See AAASM-4294.
const maxOpControlSlots = 4096

// OpControlClient is the slice of PolicyServiceClient the subscriber actually
// uses. Defined as an interface so tests can inject a mock without standing
// up a gRPC server. Mirrors PR-E's _OpControlStub Protocol and PR-F's
// OpControlClient interface.
type OpControlClient interface {
	OpControlStream(
		ctx context.Context,
		in *pb.OpControlSubscribeRequest,
		opts ...grpc.CallOption,
	) (grpc.ServerStreamingClient[pb.OpControlMessage], error)
}

// opControlState is the per-op slot used by the cooperative-pause machine.
type opControlState struct {
	paused     bool
	terminated bool
	// evicted is set when the slot is dropped from the map to make room under
	// maxOpControlSlots (see slot). A waiter woken on an evicted-but-still-paused
	// op can no longer be resumed — its resume signal would land in a fresh slot,
	// not this one — so it must fail closed exactly like a dead stream, never
	// yield an allow (AAASM-4294 / AAASM-4019).
	evicted bool
	// waiters are unbuffered channels closed when the op becomes runnable
	// (resume) or terminated (terminate). Each WaitForOp call registers a
	// fresh waiter so multiple goroutines can await the same op_id.
	waiters []chan struct{}
}

// OpControlSubscriber subscribes to OpControlStream and serves per-op
// pause/terminate signals.
//
// Construct via Connect; never directly. The zero value is not usable.
//
// Thread-safe: WaitForOp may be called from any goroutine; internal state
// is guarded by a sync.Mutex.
type OpControlSubscriber struct {
	client OpControlClient
	agent  *pb.AgentId
	conn   *grpc.ClientConn // set when Connect opened the channel; nil when constructed for tests
	cancel context.CancelFunc

	mu sync.Mutex
	// ops is the per-op_id state map. opsOrder mirrors it in insertion order
	// so the oldest entry can be evicted once the map reaches
	// maxOpControlSlots (see AAASM-4294).
	ops      map[string]*opControlState
	opsOrder []string
	alive    bool
}

// Connect opens a gRPC channel to gatewayURL, opens the OpControlStream
// subscription, and starts the background reader.
//
// gatewayURL is the "host:port" of the gateway's gRPC endpoint (no scheme;
// gRPC uses its own). The transport is secure by default: a loopback target
// (the local dev gateway) gets plaintext, any other target gets TLS using the
// system root CAs — the op-control stream carries the agent identity and
// operator pause/terminate signals, so it must not travel unencrypted to a
// remote host. To use a custom CA, mTLS, or to explicitly opt into plaintext
// for a remote host, pass your own DialOptions; supplied opts are used verbatim
// and bypass the secure-by-default decision. Mirrors python-sdk and node-sdk.
func Connect(
	ctx context.Context,
	gatewayURL string,
	orgID, teamID, agentID string,
	opts ...grpc.DialOption,
) (*OpControlSubscriber, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(resolveOpControlCredentials(gatewayURL))}
	}
	conn, err := grpc.NewClient(gatewayURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("op_control: dial %s: %w", gatewayURL, err)
	}
	client := pb.NewPolicyServiceClient(conn)
	sub, err := NewOpControlSubscriber(ctx, client, orgID, teamID, agentID)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	sub.conn = conn
	return sub, nil
}

// NewOpControlSubscriber wraps a pre-built PolicyServiceClient (or any type
// satisfying OpControlClient) and starts the subscription. Tests pass a
// mock client here; Connect uses this internally.
func NewOpControlSubscriber(
	ctx context.Context,
	client OpControlClient,
	orgID, teamID, agentID string,
) (*OpControlSubscriber, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	agent := &pb.AgentId{OrgId: orgID, TeamId: teamID, AgentId: agentID}
	stream, err := client.OpControlStream(streamCtx, &pb.OpControlSubscribeRequest{AgentId: agent})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("op_control: subscribe: %w", err)
	}
	sub := &OpControlSubscriber{
		client:   client,
		agent:    agent,
		cancel:   cancel,
		ops:      make(map[string]*opControlState),
		opsOrder: make([]string, 0),
		alive:    true,
	}
	go sub.readLoop(stream)
	return sub, nil
}

func (s *OpControlSubscriber) readLoop(stream grpc.ServerStreamingClient[pb.OpControlMessage]) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			// io.EOF = clean server shutdown; any other error = transport
			// failure or cancel. Either way, mark dead and wake waiters.
			s.markStreamDead()
			if !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
				// Future PR-G+1 may want to surface this; for now we let
				// the caller observe StreamAlive() and re-Connect.
				_ = err
			}
			return
		}
		s.dispatch(msg)
	}
}

func (s *OpControlSubscriber) dispatch(msg *pb.OpControlMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.slot(msg.GetOpId())
	switch msg.GetSignal() {
	case pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE:
		state.paused = true
	case pb.OpControlSignal_OP_CONTROL_SIGNAL_RESUME:
		state.paused = false
		s.flushWaiters(state)
	case pb.OpControlSignal_OP_CONTROL_SIGNAL_TERMINATE:
		state.terminated = true
		s.flushWaiters(state)
	}
	// UNSPECIFIED and any future variant: drop on the floor.
}

// slot lazily creates a state slot for opID. Caller must hold s.mu.
//
// When the map is already at maxOpControlSlots, the oldest NON-terminated entry
// is evicted to make room — a defense-in-depth measure against a compromised
// gateway pushing endless unique opIDs (AAASM-4294). A terminated slot is never
// chosen: it holds an OpTerminatedError verdict that a later WaitForOp for the
// same opID must still observe, so evicting it would let a fast-fail-terminated
// op resume into a fresh, non-terminated slot and proceed (AAASM-4832, the
// sibling of the AAASM-4811 paused-eviction fail-closed fix). Any waiters on the
// evicted slot are woken; an evicted-but-still-paused op is marked so its waiter
// fails closed with ErrOpControlUnavailable rather than waking to a silent allow
// — the slot is gone, so no resume signal can ever reach it, the same fail-closed
// contract as a dead stream (AAASM-4019).
func (s *OpControlSubscriber) slot(opID string) *opControlState {
	if state, ok := s.ops[opID]; ok {
		return state
	}
	if len(s.ops) >= maxOpControlSlots {
		s.evictOldestNonTerminated()
	}
	state := &opControlState{}
	s.ops[opID] = state
	s.opsOrder = append(s.opsOrder, opID)
	return state
}

// evictOldestNonTerminated drops the oldest slot that is NOT terminated to make
// room under maxOpControlSlots. Caller must hold s.mu.
//
// Terminated slots are skipped so a fast-fail-terminate verdict is never
// silently lost: were the oldest slot terminated and evicted, a later WaitForOp
// for the same opID would create a fresh, non-terminated slot and return nil
// (proceed) instead of the OpTerminatedError the operator issued (AAASM-4832).
// This mirrors the AAASM-4811 fix that keeps a paused op's eviction fail-closed:
// a paused slot may still be evicted here (its blocked waiter fails closed via
// the evicted flag), but a terminated slot's verdict must survive. If every live
// slot is terminated, nothing is evicted and the map is allowed to exceed the
// cap rather than drop a terminate verdict — the cap defends against unbounded
// unique opIDs, not against retaining genuine kill-switch signals.
func (s *OpControlSubscriber) evictOldestNonTerminated() {
	for i, opID := range s.opsOrder {
		state, ok := s.ops[opID]
		if !ok || state.terminated {
			continue
		}
		s.opsOrder = append(s.opsOrder[:i], s.opsOrder[i+1:]...)
		state.evicted = true
		s.flushWaiters(state)
		delete(s.ops, opID)
		return
	}
}

// flushWaiters closes every pending waiter channel and clears the slice.
// Caller must hold s.mu.
func (s *OpControlSubscriber) flushWaiters(state *opControlState) {
	for _, ch := range state.waiters {
		close(ch)
	}
	state.waiters = nil
}

func (s *OpControlSubscriber) markStreamDead() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alive = false
	// Wake any blocked waiters. We don't set their state to terminated — a
	// stream close is not a terminate — but a waiter still paused when it wakes
	// fails closed via ErrOpControlUnavailable rather than resuming (AAASM-4019);
	// a not-paused waiter proceeds normally.
	for _, state := range s.ops {
		s.flushWaiters(state)
	}
}

// WaitForOp blocks until opID is runnable, or returns an error.
//
// Returns nil immediately when the op is not currently paused. When paused,
// blocks until a resume signal arrives or ctx is cancelled. Returns
// *OpTerminatedError if the op has been (or becomes) terminated. Returns
// [ErrOpControlUnavailable] if the op is paused and the control stream has died
// (or dies while waiting): the pause can no longer be lifted by the operator, so
// WaitForOp fails closed rather than yielding an allow — the tool wrapper keeps
// blocking under the enforce posture (AAASM-4019).
//
// A ctx cancel returns ctx.Err() — the caller can inspect IsPaused or
// retry. This matches the cooperative-pause expectation in the architecture
// doc (the SDK yields, it doesn't deadline-enforce).
func (s *OpControlSubscriber) WaitForOp(ctx context.Context, opID string) error {
	s.mu.Lock()
	state := s.slot(opID)
	if state.terminated {
		s.mu.Unlock()
		return &OpTerminatedError{OpID: opID}
	}
	if !state.paused {
		s.mu.Unlock()
		return nil
	}
	if !s.alive {
		// The op is paused (checked above) and the control stream is already
		// dead, so no resume signal can ever arrive. Do NOT yield to allow: a
		// paused op that resumes just because the operator's kill-switch channel
		// dropped defeats the pause. Fail closed — the caller (tool wrapper)
		// keeps blocking under enforce and can observe StreamAlive() to
		// reconnect (AAASM-4019).
		s.mu.Unlock()
		return ErrOpControlUnavailable
	}
	ch := make(chan struct{})
	state.waiters = append(state.waiters, ch)
	s.mu.Unlock()

	select {
	case <-ch:
		s.mu.Lock()
		terminated := state.terminated
		stillPaused := state.paused
		alive := s.alive
		evicted := state.evicted
		s.mu.Unlock()
		if terminated {
			return &OpTerminatedError{OpID: opID}
		}
		if stillPaused && (!alive || evicted) {
			// Woken while still paused but not by a resume — either the stream
			// died (markStreamDead) or the slot was evicted under
			// maxOpControlSlots (slot). Either way the pause can no longer be
			// lifted for this waiter, so fail closed rather than proceed
			// (AAASM-4019 / AAASM-4294).
			return ErrOpControlUnavailable
		}
		return nil
	case <-ctx.Done():
		// Best-effort drop our waiter from the slot so a future flush
		// doesn't carry a no-longer-listening channel forward.
		s.mu.Lock()
		for i, w := range state.waiters {
			if w == ch {
				state.waiters = append(state.waiters[:i], state.waiters[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		return ctx.Err()
	}
}

// IsPaused returns true iff the gateway has the op currently paused.
func (s *OpControlSubscriber) IsPaused(opID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.ops[opID]
	return ok && state.paused
}

// IsTerminated returns true iff the gateway has terminated the op.
func (s *OpControlSubscriber) IsTerminated(opID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.ops[opID]
	return ok && state.terminated
}

// StreamAlive returns false once the underlying gRPC stream has closed.
func (s *OpControlSubscriber) StreamAlive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.alive
}

// Close cancels the stream and (if Connect opened the channel) closes it.
// Always returns nil — provided so callers can `defer sub.Close()`.
func (s *OpControlSubscriber) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.markStreamDead()
	return nil
}
