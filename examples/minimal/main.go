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
