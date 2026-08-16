package stt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordedRequest captures what the fake STT server received.
type recordedRequest struct {
	mu          sync.Mutex
	method      string
	path        string
	filename    string
	fileContent string
	language    string
	languageSet bool
}

func (r *recordedRequest) record(t *testing.T, req *http.Request) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.method = req.Method
	r.path = req.URL.Path

	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Errorf("ParseMultipartForm() error: %v", err)

		return
	}

	file, header, err := req.FormFile("file")
	if err != nil {
		t.Errorf("FormFile(file) error: %v", err)

		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		t.Errorf("read multipart file: %v", err)

		return
	}

	r.filename = header.Filename
	r.fileContent = string(content)

	values, ok := req.MultipartForm.Value["language"]
	r.languageSet = ok
	if ok && len(values) > 0 {
		r.language = values[0]
	}
}

func TestTranscribe_Success_NoLanguage(t *testing.T) {
	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.record(t, r)

		if _, err := w.Write([]byte(`{"text":"hello world"}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	text, err := provider.Transcribe(context.Background(), strings.NewReader("audio-bytes"), "voice.ogg")
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}

	if text != "hello world" {
		t.Errorf("Transcribe() text = %q, want %q", text, "hello world")
	}

	if recorded.method != http.MethodPost {
		t.Errorf("request method = %q, want POST", recorded.method)
	}

	if recorded.path != "/v1/audio/transcriptions" {
		t.Errorf("request path = %q, want /v1/audio/transcriptions", recorded.path)
	}

	if recorded.filename != "voice.ogg" {
		t.Errorf("multipart filename = %q, want %q", recorded.filename, "voice.ogg")
	}

	if recorded.fileContent != "audio-bytes" {
		t.Errorf("multipart file content = %q, want %q", recorded.fileContent, "audio-bytes")
	}

	if recorded.languageSet {
		t.Errorf("language field sent as %q, want field absent for empty language", recorded.language)
	}
}

func TestTranscribe_Success_WithLanguage(t *testing.T) {
	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.record(t, r)

		if _, err := w.Write([]byte(`{"text":"privet"}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "ru", server.Client())

	text, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}

	if text != "privet" {
		t.Errorf("Transcribe() text = %q, want %q", text, "privet")
	}

	if !recorded.languageSet {
		t.Fatal("language field not sent, want it present for non-empty language")
	}

	if recorded.language != "ru" {
		t.Errorf("language field = %q, want %q", recorded.language, "ru")
	}
}

func TestTranscribe_HTTPErrorWithOpenAIEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

		if _, err := w.Write([]byte(`{"error":{"message":"unsupported audio format","type":"invalid_request_error"}}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if err == nil {
		t.Fatal("Transcribe() expected error, got nil")
	}

	if !errors.Is(err, ErrUnexpectedStatus) {
		t.Errorf("Transcribe() error = %v, want errors.Is ErrUnexpectedStatus", err)
	}

	if !strings.Contains(err.Error(), "unsupported audio format") {
		t.Errorf("Transcribe() error = %q, want it to contain envelope message", err.Error())
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Transcribe() error = %q, want it to contain status code", err.Error())
	}
}

func TestTranscribe_HTTPErrorWithoutEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err := w.Write([]byte("boom")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, ErrUnexpectedStatus) {
		t.Fatalf("Transcribe() error = %v, want errors.Is ErrUnexpectedStatus", err)
	}

	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("Transcribe() error = %q, want it to contain body snippet", err.Error())
	}
}

func TestTranscribe_ContextDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := provider.Transcribe(ctx, strings.NewReader("audio"), "voice.ogg")
	if err == nil {
		t.Fatal("Transcribe() expected error, got nil")
	}

	if !errors.Is(err, ErrRequestTimeout) {
		t.Errorf("Transcribe() error = %v, want errors.Is ErrRequestTimeout", err)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Transcribe() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
}

func TestTranscribe_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.Transcribe(ctx, strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, ErrRequestTimeout) {
		t.Fatalf("Transcribe() error = %v, want errors.Is ErrRequestTimeout", err)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Transcribe() error = %v, want it to wrap context.Canceled", err)
	}
}

func TestTranscribe_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("not-a-json")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL, "", server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("Transcribe() error = %v, want errors.Is ErrInvalidResponse", err)
	}
}

func TestTranscribe_ServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close() // shut down before the call: connection refused

	provider := NewProvider(server.URL, "", &http.Client{})

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("Transcribe() error = %v, want errors.Is ErrServiceUnavailable", err)
	}
}
