package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// extractVideoNote maps a video note (round video) to the pure model the
// controller accepts.
func extractVideoNote(message *tgmodels.Message) (models.IncomingAudio, bool) {
	if message.VideoNote == nil {
		return models.IncomingAudio{}, false
	}

	return models.IncomingAudio{
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		FileID:    message.VideoNote.FileID,
		// The library types VideoNote.FileSize as int, unlike Voice.FileSize.
		FileSize: int64(message.VideoNote.FileSize),
	}, true
}

// matchVideoNote reports whether the update is a video note from a served chat.
func (d *Dispatcher) matchVideoNote(update *tgmodels.Update) bool {
	return d.matchAudioMessage(update, extractVideoNote)
}

// handleVideoNote passes a video note to the chat controller.
func (d *Dispatcher) handleVideoNote(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	d.handleAudioMessage(ctx, update, extractVideoNote, d.chat.HandleVideoNote, "failed to handle video note")
}
