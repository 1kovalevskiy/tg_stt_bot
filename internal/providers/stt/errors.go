package stt

import "errors"

var (
	ErrBuildRequest       = errors.New("failed to build stt request")
	ErrServiceUnavailable = errors.New("stt service unavailable")
	ErrRequestTimeout     = errors.New("stt request canceled or timed out")
	ErrUnexpectedStatus   = errors.New("stt returned unexpected status")
	ErrReadResponse       = errors.New("failed to read stt response")
	ErrInvalidResponse    = errors.New("stt returned invalid response")
)
