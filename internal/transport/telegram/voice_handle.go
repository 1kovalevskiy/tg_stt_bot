package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// extractVoice maps a voice message to the pure model the controller accepts.
func extractVoice(message *tgmodels.Message) (models.IncomingAudio, bool) {
	if message.Voice == nil {
		return models.IncomingAudio{}, false
	}

	return models.IncomingAudio{
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		FileID:    message.Voice.FileID,
		FileSize:  message.Voice.FileSize,
	}, true
}

// matchVoice reports whether the update is a voice message from a served chat.
func (d *Dispatcher) matchVoice(update *tgmodels.Update) bool {
	return d.matchAudioMessage(update, extractVoice)
}

// handleVoice passes a voice message to the chat controller.
func (d *Dispatcher) handleVoice(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	d.handleAudioMessage(ctx, update, extractVoice, d.chat.HandleVoice, "failed to handle voice message")
}
