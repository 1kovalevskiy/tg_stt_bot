package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// SendReply sends a text message as a reply to replyToMessageID.
// AllowSendingWithoutReply is set: the original message may have been
// deleted while the transcription was in flight, and the reply should
// still reach the chat.
func (p *Provider) SendReply(ctx context.Context, chatID int64, replyToMessageID int, text string) error {
	ctx, cancel := context.WithTimeout(ctx, p.config.GetTelegramAPITimeout())
	defer cancel()

	_, err := p.api.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
		ReplyParameters: &tgmodels.ReplyParameters{
			MessageID:                replyToMessageID,
			AllowSendingWithoutReply: true,
		},
	})
	if err != nil {
		return p.wrapRedactedError(providers.ErrTelegramSendMessage, err)
	}

	return nil
}
