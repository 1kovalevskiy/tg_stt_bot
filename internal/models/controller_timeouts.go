package models

import "time"

// Timeouts controllers derive on their own. They bound protocol-level steps
// rather than a whole external call, so they are constants and not settings.
const (
	// FailureReplyTimeout bounds the failure notice, which is sent with a
	// context detached from the caller's cancellation and would otherwise
	// hold up the shutdown for a full Bot API timeout.
	FailureReplyTimeout = 5 * time.Second
	// HealthProbeTimeout bounds the /status probe. Without it the probe
	// inherits the full transcription budget, and a hung service would keep
	// the admin waiting minutes for an answer that is supposed to be immediate.
	HealthProbeTimeout = 10 * time.Second
)
