package assembly

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeHealthz_TrueOn2xx(t *testing.T) {
	t.Parallel()

	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if !probeHealthz(context.Background(), srv.URL, defaultProbeTimeout) {
		t.Fatalf("expected probe to succeed against httptest server")
	}
	if capturedPath != defaultHealthzPath {
		t.Fatalf("expected probe to hit %q, got %q", defaultHealthzPath, capturedPath)
	}
}

func TestProbeHealthz_FalseOnConnectionRefused(t *testing.T) {
	t.Parallel()

	// 127.0.0.1:1 is a port that should never be listened on in CI.
	if probeHealthz(context.Background(), "http://127.0.0.1:1", 100*time.Millisecond) {
		t.Fatalf("expected probe against unreachable host to return false")
	}
}

func TestProbeHealthz_FalseOnNon2xx(t *testing.T) {
	t.Parallel()

	for _, status := range []int{400, 404, 500, 503} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))
		if probeHealthz(context.Background(), srv.URL, defaultProbeTimeout) {
			t.Fatalf("expected probe to return false on status %d", status)
		}
		srv.Close()
	}
}

func TestWaitForHealthz_SuccessOnFirstProbe(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if !waitForHealthz(context.Background(), srv.URL, time.Second, 10*time.Millisecond) {
		t.Fatalf("expected immediate success when server is already healthy")
	}
}

func TestWaitForHealthz_SuccessAfterInitialFailures(t *testing.T) {
	t.Parallel()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if !waitForHealthz(context.Background(), srv.URL, time.Second, 10*time.Millisecond) {
		t.Fatalf("expected success after server starts returning 200")
	}
	if calls < 3 {
		t.Fatalf("expected at least 3 probes, got %d", calls)
	}
}

func TestWaitForHealthz_FalseOnTimeout(t *testing.T) {
	t.Parallel()

	if waitForHealthz(context.Background(), "http://127.0.0.1:1", 50*time.Millisecond, 10*time.Millisecond) {
		t.Fatalf("expected false when no gateway is reachable within timeout")
	}
}
