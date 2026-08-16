package stt

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

func TestCheckHealth_Success(t *testing.T) {
	t.Parallel()

	// The handler runs on the server's goroutine: what it records is guarded
	// by a mutex on both sides.
	var (
		mu                 sync.Mutex
		gotMethod, gotPath string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPath = r.URL.Path
		mu.Unlock()

		if _, err := w.Write([]byte("{\"status\":\"ok\"}\n")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	body, err := provider.CheckHealth(context.Background())
	if err != nil {
		t.Fatalf("CheckHealth() unexpected error: %v", err)
	}

	if body != `{"status":"ok"}` {
		t.Errorf("CheckHealth() body = %q, want %q", body, `{"status":"ok"}`)
	}

	mu.Lock()
	defer mu.Unlock()

	if gotMethod != http.MethodGet {
		t.Errorf("request method = %q, want GET", gotMethod)
	}

	if gotPath != "/health" {
		t.Errorf("request path = %q, want /health", gotPath)
	}
}

func TestCheckHealth_UnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)

		if _, err := w.Write([]byte("db down")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	_, err := provider.CheckHealth(context.Background())
	if !errors.Is(err, providers.ErrSTTUnexpectedStatus) {
		t.Fatalf("CheckHealth() error = %v, want errors.Is providers.ErrSTTUnexpectedStatus", err)
	}

	if !strings.Contains(err.Error(), "503") {
		t.Errorf("CheckHealth() error = %q, want it to contain status code", err.Error())
	}

	if !strings.Contains(err.Error(), "db down") {
		t.Errorf("CheckHealth() error = %q, want it to contain body snippet", err.Error())
	}
}

func TestCheckHealth_ConfiguredTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// No deadline on the caller's context: the timeout comes from the config.
	provider := newTestProvider(server.URL, "", 20*time.Millisecond, server.Client())

	_, err := provider.CheckHealth(context.Background())
	if !errors.Is(err, providers.ErrSTTRequestTimeout) {
		t.Fatalf("CheckHealth() error = %v, want errors.Is providers.ErrSTTRequestTimeout", err)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("CheckHealth() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
}

func TestCheckHealth_ServiceUnavailable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close() // shut down before the call: connection refused

	provider := newTestProvider(server.URL, "", testTimeout, &http.Client{})

	_, err := provider.CheckHealth(context.Background())
	if !errors.Is(err, providers.ErrSTTServiceUnavailable) {
		t.Fatalf("CheckHealth() error = %v, want errors.Is providers.ErrSTTServiceUnavailable", err)
	}
}

func TestCheckHealth_ResponseBodyReadFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A body announced but never fully delivered: the connection drops
		// while the response is being read.
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(`{"status":"partial`)); err != nil {
			t.Errorf("write response: %v", err)
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		panic(http.ErrAbortHandler)
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	_, err := provider.CheckHealth(context.Background())
	if !errors.Is(err, providers.ErrSTTReadResponse) {
		t.Fatalf("CheckHealth() error = %v, want errors.Is providers.ErrSTTReadResponse", err)
	}
}

func TestCheckHealth_ContextCanceled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.CheckHealth(ctx)
	if !errors.Is(err, providers.ErrSTTRequestCanceled) {
		t.Fatalf("CheckHealth() error = %v, want errors.Is providers.ErrSTTRequestCanceled", err)
	}
}
