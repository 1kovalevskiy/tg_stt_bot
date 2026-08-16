// Package controllers holds the sentinel errors of the controller layer.
// Every controller of the layer (chat-controller, admin-controller) wraps
// failures of the layers below into these sentinels, so the wiring can match
// them with errors.Is.
package controllers

import "errors"

var (
	// Transcription scenario (chat-controller).

	// ErrDownloadAudio reports a failed audio download from Telegram.
	ErrDownloadAudio = errors.New("failed to download audio from telegram")
	// ErrTranscribeAudio reports a failed STT transcription.
	ErrTranscribeAudio = errors.New("failed to transcribe audio")
	// ErrSendReply reports a failed reply to the original message.
	ErrSendReply = errors.New("failed to send reply to chat")

	// Service commands (admin-controller).

	// ErrSTTHealth reports a failed STT health check.
	ErrSTTHealth = errors.New("failed to check stt health")
	// ErrSendMessage reports a failed plain message send.
	ErrSendMessage = errors.New("failed to send message to chat")
)
