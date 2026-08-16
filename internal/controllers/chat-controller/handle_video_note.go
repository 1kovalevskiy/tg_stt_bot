package chatController

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// videoNoteFilename is the file name reported to the STT service for video
// notes. parakeet detects the container by content, so the extension is
// informational only.
const videoNoteFilename = "video_note.mp4"

// HandleVideoNote transcribes a video note (round video) and replies with the text.
func (c *Controller) HandleVideoNote(ctx context.Context, audio models.IncomingAudio) error {
	return c.transcribeAndReply(ctx, audio, videoNoteFilename)
}
