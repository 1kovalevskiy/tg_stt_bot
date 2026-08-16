package adminController

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	commandStatus = "/status"
	commandChats  = "/chats"

	statusPrefix      = "parakeet: "
	chatsHeader       = "Разрешённые чаты:"
	msgSTTUnhealthy   = "parakeet недоступен."
	msgNoChats        = "Список разрешённых чатов пуст."
	msgUnknownCommand = "Неизвестная команда. Доступны: /status — проверка parakeet, /chats — список разрешённых чатов."
)

// HandleCommand runs an admin command and sends the result to the chat.
// An unknown command gets a short hint. Provider failures are reported to the
// chat as well and returned wrapped, so the wiring can log them.
func (c *Controller) HandleCommand(ctx context.Context, chatID int64, command string) error {
	switch normalizeCommand(command) {
	case commandStatus:
		return c.status(ctx, chatID)
	case commandChats:
		return c.chats(ctx, chatID)
	default:
		return c.send(ctx, chatID, msgUnknownCommand)
	}
}

// status checks the STT service and reports its raw health response.
func (c *Controller) status(ctx context.Context, chatID int64) error {
	health, err := c.stt.Health(ctx)
	if err != nil {
		return errors.Join(fmt.Errorf("%w: %w", ErrSTTHealth, err), c.send(ctx, chatID, msgSTTUnhealthy))
	}

	return c.send(ctx, chatID, statusPrefix+health)
}

// chats reports the configured chat whitelist.
func (c *Controller) chats(ctx context.Context, chatID int64) error {
	allowed := c.config.GetTelegramAllowedChats()
	if len(allowed) == 0 {
		return c.send(ctx, chatID, msgNoChats)
	}

	lines := make([]string, 0, len(allowed)+1)
	lines = append(lines, chatsHeader)

	for _, id := range allowed {
		lines = append(lines, strconv.FormatInt(id, 10))
	}

	return c.send(ctx, chatID, strings.Join(lines, "\n"))
}

// send sends text to the chat.
func (c *Controller) send(ctx context.Context, chatID int64, text string) error {
	if err := c.telegram.SendMessage(ctx, chatID, text); err != nil {
		return fmt.Errorf("%w: %w", ErrSendMessage, err)
	}

	return nil
}

// normalizeCommand extracts the command name from the message text: arguments
// are dropped, the "@botname" suffix Telegram adds in groups is stripped and
// the name is lowercased.
func normalizeCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}

	name, _, _ := strings.Cut(fields[0], "@")

	return strings.ToLower(name)
}
