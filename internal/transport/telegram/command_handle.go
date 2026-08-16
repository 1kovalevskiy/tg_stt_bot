package telegram

import (
	"context"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// matchAdminCommand reports whether the update is a command in the admin's
// private chat. Commands from anyone else are ignored.
func (d *Dispatcher) matchAdminCommand(update *tgmodels.Update) bool {
	message := d.resolveAdminMessage(update)
	if message == nil {
		return false
	}

	return strings.HasPrefix(strings.TrimSpace(message.Text), models.CommandPrefix)
}

// handleAdminCommand passes a command to the admin controller.
func (d *Dispatcher) handleAdminCommand(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	if !d.matchAdminCommand(update) {
		return
	}

	chatID := update.Message.Chat.ID

	if err := d.admin.HandleCommand(ctx, chatID, update.Message.Text); err != nil {
		// Only the command word is logged: the full message text would end up
		// in the service chat through the ERROR mirror. The name is parsed the
		// same way the controller parses it, so the log and the routing can
		// never disagree about which command failed.
		logUpdateError("failed to handle admin command", err,
			"chat_id", chatID, "command", models.ParseCommandName(update.Message.Text))
	}
}
