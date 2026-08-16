// Package stt provides a client for the parakeet speech-to-text service
// exposing an OpenAI-compatible transcription API.
package stt

import (
	"net/http"
	"time"
)

type (
	// httpDoer is the consumer-side interface of an HTTP client.
	httpDoer interface {
		Do(req *http.Request) (*http.Response, error)
	}

	// configProvider is the consumer-side interface of the application config.
	// The base URL is expected to come normalized (no trailing slash), and the
	// timeout bounds a single call to the service.
	configProvider interface {
		GetSTTBaseURL() string
		GetSTTLanguage() string
		GetSTTTimeout() time.Duration
	}

	// Provider is a parakeet STT API client. It holds no mutable state
	// and is safe for concurrent use.
	Provider struct {
		config configProvider
		client httpDoer
	}
)

// NewProvider creates a Provider talking to the parakeet service configured by
// config. The settings are read from the config on every call, never snapshot
// into the provider. If the configured language is empty, it is not sent and
// the service detects the language on its own.
func NewProvider(config configProvider, client httpDoer) *Provider {
	return &Provider{
		config: config,
		client: client,
	}
}
