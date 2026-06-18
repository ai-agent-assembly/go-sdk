package assembly

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGatewayClient_CheckNilContextGetsDefaultTimeout(t *testing.T) {
	t.Parallel()

	var sawDeadline bool
	client := NewGatewayClient(gatewayTransportStub{check: func(ctx context.Context, _ CheckRequest) (Decision, error) {
		_, sawDeadline = ctx.Deadline()
		return Decision{}, nil
	}})

	//nolint:staticcheck // explicitly testing the nil-ctx guard
	if _, err := client.Check(nil, CheckRequest{ToolName: "calc"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sawDeadline {
		t.Fatal("expected Check to apply a default deadline when ctx is nil")
	}
}

func TestGatewayClient_CheckNilTransportReturnsNotInitialized(t *testing.T) {
	t.Parallel()

	client := NewGatewayClient(nil)
	_, err := client.Check(context.Background(), CheckRequest{ToolName: "calc"})
	if !errors.Is(err, ErrRuntimeNotInitialized) {
		t.Fatalf("expected ErrRuntimeNotInitialized, got %v", err)
	}
}

func TestGatewayClient_CheckHonoursCallerDeadline(t *testing.T) {
	t.Parallel()

	want := time.Now().Add(123 * time.Millisecond)
	var got time.Time
	client := NewGatewayClient(gatewayTransportStub{check: func(ctx context.Context, _ CheckRequest) (Decision, error) {
		dl, _ := ctx.Deadline()
		got = dl
		return Decision{}, nil
	}})

	ctx, cancel := context.WithDeadline(context.Background(), want)
	defer cancel()
	if _, err := client.Check(ctx, CheckRequest{ToolName: "calc"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("expected caller deadline %v to be preserved, got %v", want, got)
	}
}

func TestNewGatewayClient_IgnoresNilOption(t *testing.T) {
	t.Parallel()

	// A nil option in the variadic list must be skipped, not panic.
	client := NewGatewayClient(gatewayTransportStub{}, nil, WithTimeout(time.Second))
	if client.config.timeout != time.Second {
		t.Fatalf("expected timeout option to apply past a nil option, got %v", client.config.timeout)
	}
}
