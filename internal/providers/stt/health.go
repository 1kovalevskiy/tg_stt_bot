package stt

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// healthPath is the parakeet health endpoint.
const healthPath = "/health"

// Health checks the STT service and returns the raw response body
// (e.g. {"status":"ok"}) for display in the admin /status command.
func (p *Provider) Health(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+healthPath, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrBuildRequest, err)
	}

	body, err := p.doRequest(req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
