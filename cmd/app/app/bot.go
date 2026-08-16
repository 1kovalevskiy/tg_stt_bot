package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

const (
	// botWorkers is the number of concurrent update handlers. Transcription
	// takes seconds, and the library default of one worker would make every
	// chat wait for the previous one.
	botWorkers = 4
	// commandPrefix marks a message as a bot command.
	commandPrefix = "/"
	// redactedToken replaces the bot token in error texts.
	redactedToken = "[REDACTED]"
)

type (
	// chatHandler is the consumer-side interface of the chat controller.
	chatHandler interface {
		HandleVoice(ctx context.Context, audio models.IncomingAudio) error
		HandleVideoNote(ctx context.Context, audio models.IncomingAudio) error
	}

	// adminHandler is the consumer-side interface of the admin controller.
	adminHandler interface {
		HandleCommand(ctx context.Context, chatID int64, command string) error
	}

	// dispatcher routes Telegram updates to the controllers. Updates from
	// chats the bot does not serve are silently ignored.
	dispatcher struct {
		chat    chatHandler
		admin   adminHandler
		adminID int64
		allowed []int64
	}
)

// newDispatcher creates a dispatcher for the given controllers and chat access rules.
func newDispatcher(chat chatHandler, admin adminHandler, adminID int64, allowed []int64) *dispatcher {
	return &dispatcher{
		chat:    chat,
		admin:   admin,
		adminID: adminID,
		allowed: allowed,
	}
}

// initBotClient creates the long polling client. It runs before the providers
// because the Telegram provider is built on top of this client.
func (a *App) initBotClient() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	token := a.Config.GetTelegramToken()

	client, err := bot.New(
		token,
		bot.WithWorkers(botWorkers),
		bot.WithDefaultHandler(ignoreUpdate),
		bot.WithErrorsHandler(func(err error) {
			slog.Error("telegram bot error", "err", redactToken(err.Error(), token))
		}),
	)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrCreateBot, redactToken(err.Error(), token))
	}

	a.Bot = client

	return nil
}

// initBotHandlers registers the update dispatchers. It runs after the
// controllers they dispatch to are created.
func (a *App) initBotHandlers() error {
	if a.Config == nil {
		return ErrNilConfig
	}

	if a.Bot == nil {
		return ErrNilBot
	}

	if a.Controllers.Chat == nil || a.Controllers.Admin == nil {
		return ErrNilController
	}

	d := newDispatcher(
		a.Controllers.Chat,
		a.Controllers.Admin,
		a.Config.GetTelegramAdminID(),
		a.Config.GetTelegramAllowedChats(),
	)

	a.Bot.RegisterHandlerMatchFunc(d.matchVoice, d.handleVoice)
	a.Bot.RegisterHandlerMatchFunc(d.matchVideoNote, d.handleVideoNote)
	a.Bot.RegisterHandlerMatchFunc(d.matchAdminCommand, d.handleAdminCommand)

	return nil
}

// matchVoice reports whether the update is a voice message from a served chat.
func (d *dispatcher) matchVoice(update *tgmodels.Update) bool {
	message := d.allowedMessage(update)

	return message != nil && message.Voice != nil
}

// matchVideoNote reports whether the update is a video note from a served chat.
func (d *dispatcher) matchVideoNote(update *tgmodels.Update) bool {
	message := d.allowedMessage(update)

	return message != nil && message.VideoNote != nil
}

// matchAdminCommand reports whether the update is a command in the admin's
// private chat. Commands from anyone else are ignored.
func (d *dispatcher) matchAdminCommand(update *tgmodels.Update) bool {
	if update == nil || update.Message == nil {
		return false
	}

	if update.Message.Chat.ID != d.adminID {
		return false
	}

	return strings.HasPrefix(strings.TrimSpace(update.Message.Text), commandPrefix)
}

// handleVoice passes a voice message to the chat controller.
func (d *dispatcher) handleVoice(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	message := d.allowedMessage(update)
	if message == nil || message.Voice == nil {
		return
	}

	audio := models.IncomingAudio{
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		FileID:    message.Voice.FileID,
		FileSize:  message.Voice.FileSize,
	}

	if err := d.chat.HandleVoice(ctx, audio); err != nil {
		slog.Error("failed to handle voice message",
			"err", err, "chat_id", audio.ChatID, "message_id", audio.MessageID)
	}
}

// handleVideoNote passes a video note to the chat controller.
func (d *dispatcher) handleVideoNote(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	message := d.allowedMessage(update)
	if message == nil || message.VideoNote == nil {
		return
	}

	audio := models.IncomingAudio{
		ChatID:    message.Chat.ID,
		MessageID: message.ID,
		FileID:    message.VideoNote.FileID,
		// The library types VideoNote.FileSize as int, unlike Voice.FileSize.
		FileSize: int64(message.VideoNote.FileSize),
	}

	if err := d.chat.HandleVideoNote(ctx, audio); err != nil {
		slog.Error("failed to handle video note",
			"err", err, "chat_id", audio.ChatID, "message_id", audio.MessageID)
	}
}

// handleAdminCommand passes a command to the admin controller.
func (d *dispatcher) handleAdminCommand(ctx context.Context, _ *bot.Bot, update *tgmodels.Update) {
	if !d.matchAdminCommand(update) {
		return
	}

	chatID := update.Message.Chat.ID

	if err := d.admin.HandleCommand(ctx, chatID, update.Message.Text); err != nil {
		slog.Error("failed to handle admin command",
			"err", err, "chat_id", chatID, "command", update.Message.Text)
	}
}

// allowedMessage returns the update's message if the bot serves its chat.
func (d *dispatcher) allowedMessage(update *tgmodels.Update) *tgmodels.Message {
	if update == nil || update.Message == nil {
		return nil
	}

	if !models.IsChatAllowed(update.Message.Chat.ID, d.adminID, d.allowed) {
		return nil
	}

	return update.Message
}

// ignoreUpdate drops updates no handler matched. The library default prints
// every such update, which is both noisy and leaks message content.
func ignoreUpdate(_ context.Context, _ *bot.Bot, _ *tgmodels.Update) {}

// redactToken removes the bot token from a message: library errors embed the
// request URL, which contains the token.
func redactToken(message, token string) string {
	if token == "" {
		return message
	}

	return strings.ReplaceAll(message, token, redactedToken)
}
