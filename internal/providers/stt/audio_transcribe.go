package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

// TranscribeAudio sends audio to the STT service and returns the recognized
// text. The multipart body is buffered in memory: callers only pass files up
// to the Telegram Bot API download limit (20 MB), so no streaming is needed.
func (p *Provider) TranscribeAudio(ctx context.Context, audio io.Reader, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, p.config.GetSTTTimeout())
	defer cancel()

	body := &bytes.Buffer{}

	contentType, err := writeMultipartBody(body, audio, filename, p.config.GetSTTLanguage())
	if err != nil {
		return "", err
	}

	url := p.config.GetSTTBaseURL() + models.STTTranscriptionsPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return "", fmt.Errorf("%w: %w", providers.ErrSTTBuildRequest, err)
	}

	req.Header.Set("Content-Type", contentType)

	respBody, err := p.doRequest(req)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Text string `json:"text"`
	}

	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("%w: %w", providers.ErrSTTInvalidResponse, err)
	}

	return parsed.Text, nil
}

// writeMultipartBody writes the multipart form (file + optional language)
// into dst and returns the Content-Type including the generated boundary.
// The audio is bounded by the Telegram download limit: Telegram refuses to
// serve bigger files, so a longer stream means the caller skipped its own
// check, and the memory bound stays local to the code that does the buffering.
func writeMultipartBody(dst io.Writer, audio io.Reader, filename, language string) (string, error) {
	writer := multipart.NewWriter(dst)

	part, err := writer.CreateFormFile(models.STTFileField, filename)
	if err != nil {
		return "", fmt.Errorf("%w: %w", providers.ErrSTTBuildRequest, err)
	}

	// One byte over the limit is copied on purpose: a plain LimitReader would
	// silently send truncated audio instead of failing.
	written, err := io.Copy(part, io.LimitReader(audio, models.TelegramMaxFileSize+1))
	if err != nil {
		return "", fmt.Errorf("%w: %w", providers.ErrSTTReadAudio, err)
	}

	if written > models.TelegramMaxFileSize {
		return "", fmt.Errorf("%w: over %d bytes", providers.ErrSTTAudioTooLarge, models.TelegramMaxFileSize)
	}

	if language != "" {
		if err := writer.WriteField(models.STTLanguageField, language); err != nil {
			return "", fmt.Errorf("%w: %w", providers.ErrSTTBuildRequest, err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("%w: %w", providers.ErrSTTBuildRequest, err)
	}

	return writer.FormDataContentType(), nil
}
