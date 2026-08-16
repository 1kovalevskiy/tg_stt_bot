package stt

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

// CheckHealth checks the STT service and returns the raw response body
// (e.g. {"status":"ok"}) for display in the admin /status command.
func (p *Provider) CheckHealth(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, p.config.GetSTTTimeout())
	defer cancel()

	url := p.config.GetSTTBaseURL() + models.STTHealthPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", providers.ErrSTTBuildRequest, err)
	}

	body, err := p.doRequest(req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}
