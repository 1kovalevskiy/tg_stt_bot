package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// matchVoice reports whether the update is a voice message from a served chat.
func (d *Dispatcher) matchVoice(update *tgmodels.Update) bool {
	message := d.resolveAllowedMessage(update)

	return message != nil && message.Voice != nil
}

// handleVoice passes a voice message to the chat controller. The access check
// is repeated here on purpose: the library only calls a handler whose matcher
// returned true, but the whitelist is a security boundary and re-checking it
// keeps the guarantee inside the handler rather than in the registration.
func (d *Dispatcher) handleVoice(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	message := d.resolveAllowedMessage(update)
	if message == nil || message.Voice == nil {
		return
	}

	audio := models.IncomingAudio{
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		FileID:    message.Voice.FileID,
		FileSize:  message.Voice.FileSize,
	}

	if err := d.chat.HandleVoice(ctx, audio); err != nil {
		logUpdateError("failed to handle voice message", err,
			"chat_id", audio.ChatID, "message_id", audio.MessageID)
	}
}
