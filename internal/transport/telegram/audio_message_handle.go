package telegram

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
	tgmodels "github.com/go-telegram/bot/models"
)

type (
	// audioExtractor maps a message to models.IncomingAudio. It reports false
	// when the message carries no audio of the kind it stands for.
	audioExtractor func(message *tgmodels.Message) (models.IncomingAudio, bool)

	// audioTranscriber is the controller method an audio kind is routed to.
	audioTranscriber func(ctx context.Context, audio models.IncomingAudio) error
)

// matchAudioMessage reports whether the update carries audio the extractor
// recognizes and comes from a chat the bot serves.
func (d *Dispatcher) matchAudioMessage(update *tgmodels.Update, extract audioExtractor) bool {
	message := d.resolveAllowedMessage(update)
	if message == nil {
		return false
	}

	_, ok := extract(message)

	return ok
}

// handleAudioMessage maps an audio message and passes it to the controller,
// logging a controller failure under errMsg. The access check is repeated here
// on purpose: the library only calls a handler whose matcher returned true, but
// the whitelist is a security boundary and re-checking it keeps the guarantee
// inside the handler rather than in the registration.
func (d *Dispatcher) handleAudioMessage(
	ctx context.Context, update *tgmodels.Update,
	extract audioExtractor, transcribe audioTranscriber, errMsg string,
) {
	message := d.resolveAllowedMessage(update)
	if message == nil {
		return
	}

	audio, ok := extract(message)
	if !ok {
		return
	}

	if err := transcribe(ctx, audio); err != nil {
		logUpdateError(errMsg, err, "chat_id", audio.ChatID, "message_id", audio.MessageID)
	}
}
