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

// resolveAdminMessage returns the update's message if it comes from the admin's
// private chat, and nil otherwise. Every matcher goes through one of the two
// resolvers, so the nil guard and the config reading live in a single place.
func (d *Dispatcher) resolveAdminMessage(update *tgmodels.Update) *tgmodels.Message {
	if update == nil || update.Message == nil {
		return nil
	}

	if update.Message.Chat.ID != d.config.GetTelegramAdminID() {
		return nil
	}

	return update.Message
}
