package adminController

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/controllers"
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// HandleCommand runs an admin command and sends the result to the chat.
// An unknown command gets a short hint. Provider failures are reported to the
// chat as well and returned wrapped, so the wiring can log them.
func (c *Controller) HandleCommand(ctx context.Context, chatID int64, command string) error {
	switch models.ParseCommandName(command) {
	case models.CommandStatus:
		return c.sendStatus(ctx, chatID)
	case models.CommandChats:
		return c.sendChats(ctx, chatID)
	default:
		return c.sendText(ctx, chatID, models.MsgUnknownCommand)
	}
}

// sendStatus checks the STT service and reports its raw health response.
func (c *Controller) sendStatus(ctx context.Context, chatID int64) error {
	health, err := c.checkHealth(ctx)
	if err != nil {
		return errors.Join(err, c.sendText(ctx, chatID, models.MsgSTTUnhealthy))
	}

	return c.sendText(ctx, chatID, models.StatusPrefix+health)
}

// checkHealth probes the STT service under its own short deadline, derived from
// the caller's context so that the reply below is still sendable when it expires.
func (c *Controller) checkHealth(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, models.HealthProbeTimeout)
	defer cancel()

	health, err := c.stt.CheckHealth(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", controllers.ErrSTTHealth, err)
	}

	return health, nil
}

// sendChats reports the configured chat whitelist.
func (c *Controller) sendChats(ctx context.Context, chatID int64) error {
	allowed := c.config.GetTelegramAllowedChats()
	if len(allowed) == 0 {
		return c.sendText(ctx, chatID, models.MsgNoChats)
	}

	lines := make([]string, 0, len(allowed)+1)
	lines = append(lines, models.ChatsHeader)

	for _, id := range allowed {
		lines = append(lines, strconv.FormatInt(id, 10))
	}

	return c.sendText(ctx, chatID, strings.Join(lines, "\n"))
}

// sendText sends text to the chat, split into Telegram-sized chunks: a health
// response and a long whitelist can both exceed the message limit, and
// Telegram rejects an oversized message instead of trimming it.
func (c *Controller) sendText(ctx context.Context, chatID int64, text string) error {
	for _, chunk := range models.SplitText(text, models.TelegramMessageLimit) {
		if err := c.telegram.SendMessage(ctx, chatID, chunk); err != nil {
			return fmt.Errorf("%w: %w", controllers.ErrSendMessage, err)
		}
	}

	return nil
}
