package adminController

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/1kovalevskiy/tg_stt_bot/internal/controllers"
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

const (
	commandStatus = "/status"
	commandChats  = "/chats"

	// healthTimeout bounds the /status probe. Without it the probe inherits
	// the full transcription budget, and a hung service would keep the admin
	// waiting minutes for an answer that is supposed to be immediate.
	healthTimeout = 10 * time.Second

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
	health, err := c.health(ctx)
	if err != nil {
		return errors.Join(err, c.send(ctx, chatID, msgSTTUnhealthy))
	}

	return c.send(ctx, chatID, statusPrefix+health)
}

// health probes the STT service under its own short deadline, derived from the
// caller's context so that the reply below is still sendable when it expires.
func (c *Controller) health(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()

	health, err := c.stt.Health(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", controllers.ErrSTTHealth, err)
	}

	return health, nil
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

// send sends text to the chat, split into Telegram-sized chunks: a health
// response and a long whitelist can both exceed the message limit, and
// Telegram rejects an oversized message instead of trimming it.
func (c *Controller) send(ctx context.Context, chatID int64, text string) error {
	for _, chunk := range models.SplitText(text, models.TelegramMessageLimit) {
		if err := c.telegram.SendMessage(ctx, chatID, chunk); err != nil {
			return fmt.Errorf("%w: %w", controllers.ErrSendMessage, err)
		}
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
