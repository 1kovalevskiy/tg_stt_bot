package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// matchVideoNote reports whether the update is a video note from a served chat.
func (d *Dispatcher) matchVideoNote(update *tgmodels.Update) bool {
	message := d.resolveAllowedMessage(update)

	return message != nil && message.VideoNote != nil
}

// handleVideoNote passes a video note to the chat controller. The access check
// is repeated here for the same reason as in handleVoice.
func (d *Dispatcher) handleVideoNote(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	message := d.resolveAllowedMessage(update)
	if message == nil || message.VideoNote == nil {
		return
	}

	audio := models.IncomingAudio{
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		FileID:    message.VideoNote.FileID,
		// The library types VideoNote.FileSize as int, unlike Voice.FileSize.
		FileSize: int64(message.VideoNote.FileSize),
	}

	if err := d.chat.HandleVideoNote(ctx, audio); err != nil {
		logUpdateError("failed to handle video note", err,
			"chat_id", audio.ChatID, "message_id", audio.MessageID)
	}
}
