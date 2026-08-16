package chatController

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// HandleVideoNote transcribes a video note (round video) and replies with the text.
func (c *Controller) HandleVideoNote(ctx context.Context, audio models.IncomingAudio) error {
	return c.transcribeAndReply(ctx, audio, models.VideoNoteFilename)
}
