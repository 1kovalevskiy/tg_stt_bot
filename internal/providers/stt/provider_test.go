package stt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testTimeout is generous enough not to interfere with tests that drive the
// deadline through the caller's context.
const testTimeout = 5 * time.Second

// fakeConfig is a hand-written fake of the configProvider interface.
type fakeConfig struct {
	baseURL  string
	language string
	timeout  time.Duration
}

func (c fakeConfig) GetSTTBaseURL() string {
	return strings.TrimRight(c.baseURL, "/")
}

func (c fakeConfig) GetSTTLanguage() string {
	return c.language
}

func (c fakeConfig) GetSTTTimeout() time.Duration {
	return c.timeout
}

// newTestProvider builds a Provider over a fake config.
func newTestProvider(baseURL, language string, timeout time.Duration, client httpDoer) *Provider {
	return NewProvider(fakeConfig{baseURL: baseURL, language: language, timeout: timeout}, client)
}

// TestNewProvider_ReadsConfigOnEveryCall pins the rule that the constructor
// keeps the config instead of snapshotting its values: a setting changed after
// the provider was built must show up in the next request it sends.
func TestNewProvider_ReadsConfigOnEveryCall(t *testing.T) {
	t.Parallel()

	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.record(t, r)

		if _, err := w.Write([]byte(`{"text":"ok"}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	config := &fakeConfig{baseURL: server.URL, language: "ru", timeout: testTimeout}
	provider := NewProvider(config, server.Client())

	if _, err := provider.TranscribeAudio(context.Background(), strings.NewReader("audio"), "voice.ogg"); err != nil {
		t.Fatalf("first TranscribeAudio() unexpected error: %v", err)
	}

	if got := recorded.snapshot(); got.language != "ru" {
		t.Fatalf("first request language = %q, want %q", got.language, "ru")
	}

	config.language = "en"

	if _, err := provider.TranscribeAudio(context.Background(), strings.NewReader("audio"), "voice.ogg"); err != nil {
		t.Fatalf("second TranscribeAudio() unexpected error: %v", err)
	}

	if got := recorded.snapshot(); got.language != "en" {
		t.Errorf("second request language = %q, want %q: the config was snapshot into the provider",
			got.language, "en")
	}
}
