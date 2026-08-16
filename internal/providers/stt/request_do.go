package stt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

// doRequest sends the request, maps transport errors to layer errors and
// returns the response body for 2xx responses.
func (p *Provider) doRequest(req *http.Request) ([]byte, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, wrapTransportError(err)
	}
	defer resp.Body.Close()

	// One byte over the limit is read on purpose: a plain LimitReader would
	// hand back a truncated body, which then fails as invalid JSON and hides
	// the real problem.
	body, err := io.ReadAll(io.LimitReader(resp.Body, models.STTMaxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", providers.ErrSTTReadResponse, err)
	}

	if len(body) > models.STTMaxResponseSize {
		return nil, fmt.Errorf("%w: over %d bytes", providers.ErrSTTResponseTooLarge, models.STTMaxResponseSize)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, buildStatusError(resp.StatusCode, body)
	}

	return body, nil
}

// wrapTransportError wraps a client.Do error into a layer error. A deadline
// (the configured timeout is enforced through the request context) and a
// cancellation by the caller are told apart: the latter is what a shutdown
// looks like and must not be reported as a service failure.
func wrapTransportError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("%w: %w", providers.ErrSTTRequestCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: %w", providers.ErrSTTRequestTimeout, err)
	default:
		return fmt.Errorf("%w: %w", providers.ErrSTTServiceUnavailable, err)
	}
}

// buildStatusError builds a layer error for a non-2xx response, best-effort
// decoding the OpenAI error envelope {"error":{"message":...}}.
func buildStatusError(statusCode int, body []byte) error {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Message != "" {
		return fmt.Errorf("%w: status %d: %s", providers.ErrSTTUnexpectedStatus, statusCode, envelope.Error.Message)
	}

	if snippet := buildErrorSnippet(body); snippet != "" {
		return fmt.Errorf("%w: status %d: %s", providers.ErrSTTUnexpectedStatus, statusCode, snippet)
	}

	return fmt.Errorf("%w: status %d", providers.ErrSTTUnexpectedStatus, statusCode)
}

// buildErrorSnippet trims a raw error body down to a short printable snippet.
func buildErrorSnippet(body []byte) string {
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > models.STTMaxErrorSnippet {
		snippet = snippet[:models.STTMaxErrorSnippet]
	}

	return snippet
}
