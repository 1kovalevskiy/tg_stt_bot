package stt

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

func TestHealth_Success(t *testing.T) {
	var gotMethod, gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path

		if _, err := w.Write([]byte("{\"status\":\"ok\"}\n")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	body, err := provider.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() unexpected error: %v", err)
	}

	if body != `{"status":"ok"}` {
		t.Errorf("Health() body = %q, want %q", body, `{"status":"ok"}`)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("request method = %q, want GET", gotMethod)
	}

	if gotPath != "/health" {
		t.Errorf("request path = %q, want /health", gotPath)
	}
}

func TestHealth_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)

		if _, err := w.Write([]byte("db down")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	_, err := provider.Health(context.Background())
	if !errors.Is(err, providers.ErrSTTUnexpectedStatus) {
		t.Fatalf("Health() error = %v, want errors.Is providers.ErrSTTUnexpectedStatus", err)
	}

	if !strings.Contains(err.Error(), "503") {
		t.Errorf("Health() error = %q, want it to contain status code", err.Error())
	}

	if !strings.Contains(err.Error(), "db down") {
		t.Errorf("Health() error = %q, want it to contain body snippet", err.Error())
	}
}

func TestHealth_ServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close() // shut down before the call: connection refused

	provider := NewProvider(server.URL, "", &http.Client{})

	_, err := provider.Health(context.Background())
	if !errors.Is(err, providers.ErrSTTServiceUnavailable) {
		t.Fatalf("Health() error = %v, want errors.Is providers.ErrSTTServiceUnavailable", err)
	}
}
