package stt

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

// healthPath is the parakeet health endpoint.
const healthPath = "/health"

// Health checks the STT service and returns the raw response body
// (e.g. {"status":"ok"}) for display in the admin /status command.
func (p *Provider) Health(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, p.config.GetSTTTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config.GetSTTBaseURL()+healthPath, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", providers.ErrSTTBuildRequest, err)
	}

	body, err := p.doRequest(req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
