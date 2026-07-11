// Package main shows a minimal go-sdk bootstrap example.
package main

import (
	"context"
	"log"
	"time"

	"github.com/ai-agent-assembly/go-sdk/assembly"
)

func main() {
	a, err := assembly.Init(context.Background(),
		assembly.WithGatewayURL("https://your-gateway.com"),
		assembly.WithAPIKey("xxx"),
		// Point at a running gateway's gRPC address. This is required to reach
		// Init's registration path: without it (or WithSidecarBinary) Init
		// returns assembly.ErrSidecarUnavailable. Full gateway registration (the
		// possession-proof handshake) additionally requires building with the
		// native cgo binding (`-tags aa_ffi_go`, CGO_ENABLED=1); the default
		// pure-Go build has no native transport and does not self-register.
		assembly.WithSidecarAddress("127.0.0.1:50051"),
		// WARNING: WithFailClosed(false) enables fail-open mode.
		// Use only for testing/debugging. Production should use fail-closed (default).
		assembly.WithFailClosed(false),
		assembly.WithTimeout(500*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("init assembly runtime: %v", err)
	}
	defer func() {
		if err := a.Close(); err != nil {
			log.Printf("close assembly runtime: %v", err)
		}
	}()
}
