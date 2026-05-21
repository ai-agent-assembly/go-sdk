// Unit tests for OpControlSubscriber (AAASM-1422 PR-G / AAASM-1656).
//
// The subscriber owns a background goroutine that reads from the gRPC
// stream and dispatches signals to a per-op state machine. We exercise
// it by injecting a hand-rolled mock client + stream — no gRPC server
// stood up. Mirrors the python-sdk PR-E and node-sdk PR-F test seam
// patterns.

package assembly

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/AI-agent-assembly/go-sdk/internal/proto"
)

// fakeStream stands in for grpc.ServerStreamingClient[pb.OpControlMessage].
// Implements the methods the subscriber's readLoop calls + a push() helper
// for the test to fan messages through.
type fakeStream struct {
	mu     sync.Mutex
	cond   *sync.Cond
	queue  []*pb.OpControlMessage
	closed bool
	err    error
}

func newFakeStream() *fakeStream {
	fs := &fakeStream{}
	fs.cond = sync.NewCond(&fs.mu)
	return fs
}

func (f *fakeStream) push(msg *pb.OpControlMessage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queue = append(f.queue, msg)
	f.cond.Signal()
}

func (f *fakeStream) end() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	f.cond.Broadcast()
}

func (f *fakeStream) errorOut(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
	f.cond.Broadcast()
}

func (f *fakeStream) Recv() (*pb.OpControlMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for len(f.queue) == 0 && !f.closed && f.err == nil {
		f.cond.Wait()
	}
	if len(f.queue) > 0 {
		msg := f.queue[0]
		f.queue = f.queue[1:]
		return msg, nil
	}
	if f.err != nil {
		return nil, f.err
	}
	return nil, io.EOF
}

// The remaining ServerStreamingClient methods aren't used by the
// subscriber's readLoop — stub them out so the type-assertion compiles.
func (f *fakeStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeStream) Trailer() metadata.MD         { return nil }
func (f *fakeStream) CloseSend() error             { return nil }
func (f *fakeStream) Context() context.Context     { return context.Background() }
func (f *fakeStream) SendMsg(m any) error          { return nil }
func (f *fakeStream) RecvMsg(m any) error          { return nil }

// fakeClient implements OpControlClient.
type fakeClient struct {
	stream      *fakeStream
	lastRequest *pb.OpControlSubscribeRequest
}

func (c *fakeClient) OpControlStream(
	_ context.Context,
	in *pb.OpControlSubscribeRequest,
	_ ...grpc.CallOption,
) (grpc.ServerStreamingClient[pb.OpControlMessage], error) {
	c.lastRequest = in
	return c.stream, nil
}

func newSubscriber(t *testing.T) (*OpControlSubscriber, *fakeStream, *fakeClient) {
	t.Helper()
	stream := newFakeStream()
	client := &fakeClient{stream: stream}
	sub, err := NewOpControlSubscriber(context.Background(), client, "org", "team", "agent-7")
	if err != nil {
		t.Fatalf("NewOpControlSubscriber: %v", err)
	}
	t.Cleanup(func() {
		stream.end()
		_ = sub.Close()
	})
	return sub, stream, client
}

func msg(opID string, signal pb.OpControlSignal, sequence uint64) *pb.OpControlMessage {
	return &pb.OpControlMessage{OpId: opID, Signal: signal, Sequence: sequence}
}

func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

func TestWaitForOp_ReturnsImmediatelyForUnknownOp(t *testing.T) {
	sub, _, _ := newSubscriber(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := sub.WaitForOp(ctx, "never-seen"); err != nil {
		t.Fatalf("WaitForOp returned %v; want nil", err)
	}
}

func TestPauseBlocksUntilResume(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-1", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-1") }, time.Second)

	done := make(chan error, 1)
	go func() {
		done <- sub.WaitForOp(context.Background(), "op-1")
	}()

	select {
	case <-done:
		t.Fatal("WaitForOp returned while op was paused")
	case <-time.After(50 * time.Millisecond):
		// Good — still blocked.
	}

	stream.push(msg("op-1", pb.OpControlSignal_OP_CONTROL_SIGNAL_RESUME, 1))
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WaitForOp returned %v; want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForOp did not unblock after resume")
	}
	if sub.IsPaused("op-1") {
		t.Fatal("op-1 still paused after resume")
	}
}

func TestTerminateReturnsOpTerminatedError(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-2", pb.OpControlSignal_OP_CONTROL_SIGNAL_TERMINATE, 0))
	waitFor(t, func() bool { return sub.IsTerminated("op-2") }, time.Second)

	err := sub.WaitForOp(context.Background(), "op-2")
	var oe *OpTerminatedError
	if !errors.As(err, &oe) {
		t.Fatalf("WaitForOp returned %v; want *OpTerminatedError", err)
	}
	if oe.OpID != "op-2" {
		t.Fatalf("OpTerminatedError.OpID = %q; want %q", oe.OpID, "op-2")
	}
}

func TestTerminateUnblocksWaiterAndReturnsError(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-3", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-3") }, time.Second)

	done := make(chan error, 1)
	go func() {
		done <- sub.WaitForOp(context.Background(), "op-3")
	}()

	stream.push(msg("op-3", pb.OpControlSignal_OP_CONTROL_SIGNAL_TERMINATE, 1))

	select {
	case err := <-done:
		var oe *OpTerminatedError
		if !errors.As(err, &oe) {
			t.Fatalf("WaitForOp returned %v; want *OpTerminatedError", err)
		}
		if oe.OpID != "op-3" {
			t.Fatalf("OpID = %q; want %q", oe.OpID, "op-3")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForOp did not unblock after terminate")
	}
}

func TestBufferedSignalObservedByFirstWaiter(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-4", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-4") }, time.Second)

	done := make(chan error, 1)
	go func() {
		done <- sub.WaitForOp(context.Background(), "op-4")
	}()

	select {
	case <-done:
		t.Fatal("WaitForOp returned while op was paused")
	case <-time.After(50 * time.Millisecond):
	}

	stream.push(msg("op-4", pb.OpControlSignal_OP_CONTROL_SIGNAL_RESUME, 1))
	if err := <-done; err != nil {
		t.Fatalf("WaitForOp returned %v; want nil", err)
	}
}

func TestSubscribeRequestCarriesCompositeAgentID(t *testing.T) {
	_, _, client := newSubscriber(t)
	if client.lastRequest == nil {
		t.Fatal("no subscribe request was captured")
	}
	got := client.lastRequest.GetAgentId()
	if got.GetOrgId() != "org" || got.GetTeamId() != "team" || got.GetAgentId() != "agent-7" {
		t.Fatalf("subscribe agent_id = %+v; want {org, team, agent-7}", got)
	}
}

func TestStreamEndWakesBlockedWaiters(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-5", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-5") }, time.Second)

	done := make(chan error, 1)
	go func() {
		done <- sub.WaitForOp(context.Background(), "op-5")
	}()

	stream.end()

	select {
	case err := <-done:
		// Close is a normal lifecycle event, not a terminate — WaitForOp
		// should return nil and let the caller observe StreamAlive().
		if err != nil {
			t.Fatalf("WaitForOp returned %v; want nil on stream end", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForOp did not wake on stream end")
	}
	waitFor(t, func() bool { return !sub.StreamAlive() }, time.Second)
}

func TestContextCancelReleasesWaiter(t *testing.T) {
	sub, stream, _ := newSubscriber(t)
	stream.push(msg("op-6", pb.OpControlSignal_OP_CONTROL_SIGNAL_PAUSE, 0))
	waitFor(t, func() bool { return sub.IsPaused("op-6") }, time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- sub.WaitForOp(ctx, "op-6")
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForOp returned %v; want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForOp did not release on ctx cancel")
	}
}
