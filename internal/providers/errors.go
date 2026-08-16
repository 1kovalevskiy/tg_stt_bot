// Package providers holds the sentinel errors of the provider layer.
// Every provider of the layer (stt, telegram) wraps transport and API
// failures into these sentinels, so the controllers can match them with
// errors.Is. Names are prefixed with the external service the error belongs
// to: the same failure mode (building a request, an unexpected status) exists
// for both services and must stay distinguishable.
package providers

import "errors"

var (
	// parakeet STT service.

	// ErrSTTBuildRequest reports a malformed STT request or body.
	ErrSTTBuildRequest = errors.New("failed to build stt request")
	// ErrSTTReadAudio reports an unreadable audio source: the failure is on
	// the caller's stream (the Telegram download), not in the STT service.
	ErrSTTReadAudio = errors.New("failed to read audio for stt request")
	// ErrSTTAudioTooLarge reports an audio stream over the request body limit.
	ErrSTTAudioTooLarge = errors.New("audio is too large for an stt request")
	// ErrSTTServiceUnavailable reports an STT transport failure.
	ErrSTTServiceUnavailable = errors.New("stt service unavailable")
	// ErrSTTRequestTimeout reports an STT request that ran out of time.
	ErrSTTRequestTimeout = errors.New("stt request timed out")
	// ErrSTTRequestCanceled reports an STT request canceled by its caller,
	// which is what a shutdown looks like and is not a service failure.
	ErrSTTRequestCanceled = errors.New("stt request canceled")
	// ErrSTTUnexpectedStatus reports a non-2xx STT response.
	ErrSTTUnexpectedStatus = errors.New("stt returned unexpected status")
	// ErrSTTReadResponse reports an unreadable STT response body.
	ErrSTTReadResponse = errors.New("failed to read stt response")
	// ErrSTTResponseTooLarge reports an STT response body over the read limit,
	// which would otherwise be truncated and reported as invalid JSON.
	ErrSTTResponseTooLarge = errors.New("stt response is too large")
	// ErrSTTInvalidResponse reports an undecodable STT response body.
	ErrSTTInvalidResponse = errors.New("stt returned invalid response")

	// Telegram Bot API.

	// ErrTelegramGetFile reports a failed getFile call.
	ErrTelegramGetFile = errors.New("failed to get telegram file info")
	// ErrTelegramEmptyFilePath reports getFile answering without a file path
	// or without a file at all.
	ErrTelegramEmptyFilePath = errors.New("telegram returned empty file path")
	// ErrTelegramBuildRequest reports a malformed file download request.
	ErrTelegramBuildRequest = errors.New("failed to build telegram download request")
	// ErrTelegramDownloadFailed reports a file download transport failure.
	ErrTelegramDownloadFailed = errors.New("failed to download telegram file")
	// ErrTelegramUnexpectedStatus reports a non-200 file download response.
	ErrTelegramUnexpectedStatus = errors.New("telegram file download returned unexpected status")
	// ErrTelegramSendMessage reports a failed sendMessage call.
	ErrTelegramSendMessage = errors.New("failed to send telegram message")
)
