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
	if update == nil || update.Message == nil {
		return false
	}

	if update.Message.Chat.ID != d.config.GetTelegramAdminID() {
		return false
	}

	return strings.HasPrefix(strings.TrimSpace(update.Message.Text), models.CommandPrefix)
}

// handleAdminCommand passes a command to the admin controller.
func (d *Dispatcher) handleAdminCommand(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	if !d.matchAdminCommand(update) {
		return
	}

	chatID := update.Message.Chat.ID

	if err := d.admin.HandleCommand(ctx, chatID, update.Message.Text); err != nil {
		// Only the command word is logged: the full message text would end up
		// in the service chat through the ERROR mirror.
		logUpdateError("failed to handle admin command", err,
			"chat_id", chatID, "command", parseCommandName(update.Message.Text))
	}
}

// parseCommandName returns the command word of a message without its arguments
// and without the "@botname" suffix Telegram adds in groups.
func parseCommandName(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	name, _, _ := strings.Cut(fields[0], "@")

	return name
}
