package assembly

import (
	"context"
	"strings"
	"testing"
)

// TestConnect_ErrorIsWrapped drives Connect's error-return path with an
// invalid gRPC target so no real network I/O occurs. grpc.NewClient is lazy,
// so the failure surfaces from the subscribe call; either way Connect must
// wrap it under the op_control prefix and not leak a live subscriber.
func TestConnect_ErrorIsWrapped(t *testing.T) {
	t.Parallel()

	sub, err := Connect(context.Background(), "://bad-target", "org", "team", "agent")
	if err == nil {
		t.Fatal("expected Connect to fail for an invalid gRPC target")
	}
	if sub != nil {
		t.Fatalf("expected nil subscriber on error, got %+v", sub)
	}
	if !strings.HasPrefix(err.Error(), "op_control:") {
		t.Fatalf("expected op_control error wrap, got %v", err)
	}
}
