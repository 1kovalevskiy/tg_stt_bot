// Package stt provides a client for the parakeet speech-to-text service
// exposing an OpenAI-compatible transcription API.
package stt

import (
	"net/http"
	"strings"
)

type (
	// httpDoer is the consumer-side interface of an HTTP client.
	httpDoer interface {
		Do(req *http.Request) (*http.Response, error)
	}

	// Provider is a parakeet STT API client. It holds no mutable state
	// and is safe for concurrent use.
	Provider struct {
		baseURL  string
		language string
		client   httpDoer
	}
)

// NewProvider creates a Provider talking to the parakeet service at baseURL.
// If language is empty, it is not sent and the service detects the language
// on its own.
func NewProvider(baseURL, language string, client httpDoer) *Provider {
	return &Provider{
		baseURL:  strings.TrimRight(baseURL, "/"),
		language: language,
		client:   client,
	}
}
