package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// transcriptionsPath is the OpenAI-compatible transcription endpoint.
const transcriptionsPath = "/v1/audio/transcriptions"

// Transcribe sends audio to the STT service and returns the recognized text.
// The multipart body is buffered in memory: callers only pass files up to
// the Telegram Bot API download limit (20 MB), so no streaming is needed.
func (p *Provider) Transcribe(ctx context.Context, audio io.Reader, filename string) (string, error) {
	body := &bytes.Buffer{}

	contentType, err := writeMultipartBody(body, audio, filename, p.language)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+transcriptionsPath, body)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrBuildRequest, err)
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
		return "", fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	return parsed.Text, nil
}

// writeMultipartBody writes the multipart form (file + optional language)
// into dst and returns the Content-Type including the generated boundary.
func writeMultipartBody(dst io.Writer, audio io.Reader, filename, language string) (string, error) {
	writer := multipart.NewWriter(dst)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrBuildRequest, err)
	}

	if _, err := io.Copy(part, audio); err != nil {
		return "", fmt.Errorf("%w: %w", ErrBuildRequest, err)
	}

	if language != "" {
		if err := writer.WriteField("language", language); err != nil {
			return "", fmt.Errorf("%w: %w", ErrBuildRequest, err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("%w: %w", ErrBuildRequest, err)
	}

	return writer.FormDataContentType(), nil
}
