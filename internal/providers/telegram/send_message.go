package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
	"github.com/go-telegram/bot"
)

// SendMessage sends a plain text message to the chat.
func (p *Provider) SendMessage(ctx context.Context, chatID int64, text string) error {
	ctx, cancel := context.WithTimeout(ctx, p.config.GetTelegramAPITimeout())
	defer cancel()

	_, err := p.api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		return p.wrapRedacted(providers.ErrTelegramSendMessage, err)
	}

	return nil
}
