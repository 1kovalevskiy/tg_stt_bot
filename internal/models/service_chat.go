package models

import "time"

// Service chat log delivery constants: queue size, delivery timeouts and the
// rate limit that keeps a repeating failure from flooding the chat.
const (
	// ServiceChatQueueSize is how many ERROR records may wait for delivery
	// before new ones are dropped instead of blocking the logging caller.
	ServiceChatQueueSize = 64
	// ServiceChatSendTimeout bounds a single delivery attempt.
	ServiceChatSendTimeout = 15 * time.Second
	// ServiceChatDrainTimeout bounds the queue drain on shutdown. It has to
	// exceed a single send timeout, otherwise Close reports a failure for a
	// delivery that is still perfectly on time.
	ServiceChatDrainTimeout = ServiceChatSendTimeout + 5*time.Second
	// ServiceChatRateWindow and ServiceChatRateBurst cap how many records may
	// be delivered per window: a repeating failure would otherwise turn into
	// one Telegram message per occurrence, flooding the chat and burning the
	// send quota shared with user replies.
	ServiceChatRateWindow = time.Minute
	ServiceChatRateBurst  = 10
	// MsgSuppressedFormat reports how many records the rate limit dropped.
	MsgSuppressedFormat = "%d error records suppressed by the service chat rate limit"
)
