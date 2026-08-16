// Package chatController implements the transcription scenario for incoming
// audio: download the file from Telegram, send it to the STT service and
// reply to the original message with the recognized text.
package chatController

import (
	"context"
	"io"
)

type (
	// telegramProvider is the consumer-side interface of the Telegram provider.
	telegramProvider interface {
		DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error)
		SendReply(ctx context.Context, chatID int64, replyToMessageID int, text string) error
	}

	// sttProvider is the consumer-side interface of the STT provider.
	sttProvider interface {
		TranscribeAudio(ctx context.Context, audio io.Reader, filename string) (string, error)
	}

	// Controller handles voice messages and video notes. It holds no mutable
	// state and is safe for concurrent use as long as its providers are.
	Controller struct {
		telegram telegramProvider
		stt      sttProvider
	}
)

// NewController creates a Controller on top of the Telegram and STT providers.
func NewController(telegram telegramProvider, stt sttProvider) *Controller {
	return &Controller{
		telegram: telegram,
		stt:      stt,
	}
}
