package stt

import (
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

func TestNewProvider_KeepsConfigInsteadOfSnapshot(t *testing.T) {
	t.Parallel()

	config := &fakeConfig{baseURL: "http://stt.local:5092/", language: "ru", timeout: 42 * time.Second}
	provider := NewProvider(config, nil)

	if got := provider.config.GetSTTBaseURL(); got != "http://stt.local:5092" {
		t.Errorf("config.GetSTTBaseURL() = %q, want %q", got, "http://stt.local:5092")
	}

	if got := provider.config.GetSTTTimeout(); got != 42*time.Second {
		t.Errorf("config.GetSTTTimeout() = %v, want %v", got, 42*time.Second)
	}

	// The values are read from the config, not copied into the provider.
	config.language = "en"

	if got := provider.config.GetSTTLanguage(); got != "en" {
		t.Errorf("config.GetSTTLanguage() = %q, want %q", got, "en")
	}
}
