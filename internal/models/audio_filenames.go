package models

// File names reported to the STT service for each kind of incoming audio.
// parakeet detects the container by content, so the extension is
// informational only.
const (
	// VoiceFilename is the file name used for voice messages.
	VoiceFilename = "voice.oga"
	// VideoNoteFilename is the file name used for video notes (round videos).
	VideoNoteFilename = "video_note.mp4"
)
