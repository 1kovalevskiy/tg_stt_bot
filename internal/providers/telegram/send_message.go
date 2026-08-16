package telegram

import (
	"context"

	"github.com/go-telegram/bot"
)

// SendMessage sends a plain text message to the chat.
func (p *Provider) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := p.api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		return p.wrapRedacted(ErrSendMessage, err)
	}

	return nil
}
