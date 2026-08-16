package telegram

import (
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	tgmodels "github.com/go-telegram/bot/models"
)

// resolveAllowedMessage returns the update's message if the bot serves its
// chat, and nil otherwise. The decision itself is a pure function in models.
func (d *Dispatcher) resolveAllowedMessage(update *tgmodels.Update) *tgmodels.Message {
	if update == nil || update.Message == nil {
		return nil
	}

	allowed := models.IsChatAllowed(
		update.Message.Chat.ID,
		d.config.GetTelegramAdminID(),
		d.config.GetTelegramAllowedChats(),
	)
	if !allowed {
		return nil
	}

	return update.Message
}
