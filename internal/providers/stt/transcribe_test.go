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

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
)

// requestData is what the fake STT server saw in a single request.
type requestData struct {
	method      string
	path        string
	filename    string
	fileContent string
	language    string
	languageSet bool
}

// recordedRequest captures what the fake STT server received. The handler runs
// on the server's goroutine, so the data is guarded by the mutex on both the
// writing and the reading side.
type recordedRequest struct {
	mu   sync.Mutex
	data requestData
}

// snapshot returns a copy of the recorded data, safe to read from the test.
func (r *recordedRequest) snapshot() requestData {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.data
}

func (r *recordedRequest) record(t *testing.T, req *http.Request) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.data.method = req.Method
	r.data.path = req.URL.Path

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

	r.data.filename = header.Filename
	r.data.fileContent = string(content)

	values, ok := req.MultipartForm.Value["language"]
	r.data.languageSet = ok
	if ok && len(values) > 0 {
		r.data.language = values[0]
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

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	text, err := provider.Transcribe(context.Background(), strings.NewReader("audio-bytes"), "voice.ogg")
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}

	if text != "hello world" {
		t.Errorf("Transcribe() text = %q, want %q", text, "hello world")
	}

	got := recorded.snapshot()

	if got.method != http.MethodPost {
		t.Errorf("request method = %q, want POST", got.method)
	}

	if got.path != "/v1/audio/transcriptions" {
		t.Errorf("request path = %q, want /v1/audio/transcriptions", got.path)
	}

	if got.filename != "voice.ogg" {
		t.Errorf("multipart filename = %q, want %q", got.filename, "voice.ogg")
	}

	if got.fileContent != "audio-bytes" {
		t.Errorf("multipart file content = %q, want %q", got.fileContent, "audio-bytes")
	}

	if got.languageSet {
		t.Errorf("language field sent as %q, want field absent for empty language", got.language)
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

	provider := newTestProvider(server.URL, "ru", testTimeout, server.Client())

	text, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}

	if text != "privet" {
		t.Errorf("Transcribe() text = %q, want %q", text, "privet")
	}

	got := recorded.snapshot()

	if !got.languageSet {
		t.Fatal("language field not sent, want it present for non-empty language")
	}

	if got.language != "ru" {
		t.Errorf("language field = %q, want %q", got.language, "ru")
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

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if err == nil {
		t.Fatal("Transcribe() expected error, got nil")
	}

	if !errors.Is(err, providers.ErrSTTUnexpectedStatus) {
		t.Errorf("Transcribe() error = %v, want errors.Is providers.ErrSTTUnexpectedStatus", err)
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

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, providers.ErrSTTUnexpectedStatus) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTUnexpectedStatus", err)
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

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := provider.Transcribe(ctx, strings.NewReader("audio"), "voice.ogg")
	if err == nil {
		t.Fatal("Transcribe() expected error, got nil")
	}

	if !errors.Is(err, providers.ErrSTTRequestTimeout) {
		t.Errorf("Transcribe() error = %v, want errors.Is providers.ErrSTTRequestTimeout", err)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Transcribe() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
}

func TestTranscribe_ConfiguredTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// No deadline on the caller's context: the timeout comes from the config.
	provider := newTestProvider(server.URL, "", 20*time.Millisecond, server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, providers.ErrSTTRequestTimeout) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTRequestTimeout", err)
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

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.Transcribe(ctx, strings.NewReader("audio"), "voice.ogg")
	// A cancellation is the caller shutting down, not the service failing:
	// it must not be reported as a timeout.
	if !errors.Is(err, providers.ErrSTTRequestCanceled) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTRequestCanceled", err)
	}

	if errors.Is(err, providers.ErrSTTRequestTimeout) {
		t.Errorf("Transcribe() error = %v, want a cancellation told apart from a timeout", err)
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

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, providers.ErrSTTInvalidResponse) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTInvalidResponse", err)
	}
}

func TestTranscribe_ServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close() // shut down before the call: connection refused

	provider := newTestProvider(server.URL, "", testTimeout, &http.Client{})

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, providers.ErrSTTServiceUnavailable) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTServiceUnavailable", err)
	}
}

// failingReader fails mid-stream, the way a dropped Telegram download does.
type failingReader struct {
	head string
	err  error
	done bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, r.err
	}

	r.done = true

	return copy(p, r.head), nil
}

func TestTranscribe_AudioReadFailureIsNotAnSTTFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("request sent, want the failure detected before any request")
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	readErr := errors.New("connection reset by peer")

	_, err := provider.Transcribe(context.Background(), &failingReader{head: "audio", err: readErr}, "voice.ogg")
	if !errors.Is(err, providers.ErrSTTReadAudio) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTReadAudio", err)
	}

	if !errors.Is(err, readErr) {
		t.Errorf("Transcribe() error = %v, want it to wrap the reader failure", err)
	}

	// The audio source failed, so the STT service must not be blamed for it.
	if errors.Is(err, providers.ErrSTTBuildRequest) {
		t.Errorf("Transcribe() error = %v, want the audio failure told apart from a request failure", err)
	}
}

func TestTranscribe_AudioOverTheSizeLimitIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("request sent, want oversized audio rejected before any request")
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	// One byte over the limit: a plain LimitReader would silently send a
	// truncated file instead of failing.
	audio := strings.NewReader(strings.Repeat("a", maxAudioSize+1))

	_, err := provider.Transcribe(context.Background(), audio, "voice.ogg")
	if !errors.Is(err, providers.ErrSTTAudioTooLarge) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTAudioTooLarge", err)
	}
}

func TestTranscribe_AudioExactlyAtTheSizeLimitIsSent(t *testing.T) {
	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.record(t, r)

		if _, err := w.Write([]byte(`{"text":"ok"}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	audio := strings.Repeat("a", maxAudioSize)

	text, err := provider.Transcribe(context.Background(), strings.NewReader(audio), "voice.ogg")
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}

	if text != "ok" {
		t.Errorf("Transcribe() text = %q, want %q", text, "ok")
	}

	if got := len(recorded.snapshot().fileContent); got != maxAudioSize {
		t.Errorf("server received %d bytes, want the whole %d", got, maxAudioSize)
	}
}

func TestTranscribe_RedirectStatusIsNotATranscript(t *testing.T) {
	// A 3xx body is not a transcript: without the upper 2xx bound it would be
	// parsed and returned as recognized text.
	for _, status := range []int{http.StatusFound, http.StatusNotModified} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))

		provider := newTestProvider(server.URL, "", testTimeout, &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		})

		_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
		if !errors.Is(err, providers.ErrSTTUnexpectedStatus) {
			t.Errorf("Transcribe() error for status %d = %v, want errors.Is providers.ErrSTTUnexpectedStatus",
				status, err)
		}

		server.Close()
	}
}

func TestTranscribe_OversizedResponseIsRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A misrouted base_url answering with a huge page must not come back
		// as "invalid response": the size is the actual problem.
		if _, err := w.Write([]byte(`{"text":"` + strings.Repeat("a", maxResponseSize) + `"}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, providers.ErrSTTResponseTooLarge) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTResponseTooLarge", err)
	}

	if errors.Is(err, providers.ErrSTTInvalidResponse) {
		t.Errorf("Transcribe() error = %v, want the size failure told apart from a decode failure", err)
	}
}

func TestTranscribe_ResponseAtTheReadLimitIsAccepted(t *testing.T) {
	text := strings.Repeat("a", maxResponseSize-len(`{"text":""}`))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(`{"text":"` + text + `"}`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	got, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if err != nil {
		t.Fatalf("Transcribe() unexpected error: %v", err)
	}

	if got != text {
		t.Errorf("Transcribe() returned %d characters, want %d", len(got), len(text))
	}
}

func TestTranscribe_ErrorBodySnippetIsTruncated(t *testing.T) {
	body := strings.Repeat("b", maxErrorSnippet*4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	provider := newTestProvider(server.URL, "", testTimeout, server.Client())

	_, err := provider.Transcribe(context.Background(), strings.NewReader("audio"), "voice.ogg")
	if !errors.Is(err, providers.ErrSTTUnexpectedStatus) {
		t.Fatalf("Transcribe() error = %v, want errors.Is providers.ErrSTTUnexpectedStatus", err)
	}

	// The whole body would otherwise land in slog.Error and from there in the
	// service chat.
	if strings.Count(err.Error(), "b") > maxErrorSnippet {
		t.Errorf("Transcribe() error carries %d body bytes, want at most %d",
			strings.Count(err.Error(), "b"), maxErrorSnippet)
	}
}
