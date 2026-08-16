package telegram

import (
	"context"
	"io"
)

// cancelReadCloser ties a context to the body it wraps. DownloadFile returns
// a stream the caller reads after the call returns, so the download timeout
// must stay armed until the caller is done: the context is released on Close,
// not when DownloadFile returns.
type cancelReadCloser struct {
	body   io.ReadCloser
	cancel context.CancelFunc
}

// newCancelReadCloser wraps body so that Close also calls cancel.
func newCancelReadCloser(body io.ReadCloser, cancel context.CancelFunc) *cancelReadCloser {
	return &cancelReadCloser{
		body:   body,
		cancel: cancel,
	}
}

// Read reads from the underlying body.
func (r *cancelReadCloser) Read(p []byte) (int, error) {
	return r.body.Read(p)
}

// Close closes the underlying body and releases the context. Repeated calls
// are safe: a context.CancelFunc is idempotent.
func (r *cancelReadCloser) Close() error {
	defer r.cancel()

	return r.body.Close()
}
