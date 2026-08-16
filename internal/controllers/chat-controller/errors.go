package chatController

import "errors"

var (
	ErrDownloadAudio   = errors.New("failed to download audio from telegram")
	ErrTranscribeAudio = errors.New("failed to transcribe audio")
	ErrSendReply       = errors.New("failed to send reply to chat")
)
