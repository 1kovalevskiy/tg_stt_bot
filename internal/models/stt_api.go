package models

// parakeet STT API constants: endpoints, multipart field names and the limits
// applied to a single response.
const (
	// STTTranscriptionsPath is the OpenAI-compatible transcription endpoint.
	STTTranscriptionsPath = "/v1/audio/transcriptions"
	// STTHealthPath is the parakeet health endpoint.
	STTHealthPath = "/health"
	// STTFileField is the multipart field carrying the audio file.
	STTFileField = "file"
	// STTLanguageField is the multipart field carrying the recognition language.
	STTLanguageField = "language"
	// STTMaxResponseSize limits how much of a service response is read into memory.
	STTMaxResponseSize = 1 << 20 // 1 MB
	// STTMaxErrorSnippet limits how much of a non-JSON error body ends up in an error.
	STTMaxErrorSnippet = 256
)
