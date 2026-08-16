package models

import "testing"

// TestServiceChatDrainTimeoutExceedsSendTimeout pins the relation the sink
// relies on: the drain has to outlast a single delivery attempt, otherwise
// Close reports a failure for a send that is still perfectly on time.
func TestServiceChatDrainTimeoutExceedsSendTimeout(t *testing.T) {
	t.Parallel()

	if ServiceChatDrainTimeout <= ServiceChatSendTimeout {
		t.Errorf("ServiceChatDrainTimeout = %v, want more than ServiceChatSendTimeout %v",
			ServiceChatDrainTimeout, ServiceChatSendTimeout)
	}
}

// TestServiceChatRateLimitIsPositive pins that the rate limit lets records
// through at all: a zero burst would silence the service chat completely.
func TestServiceChatRateLimitIsPositive(t *testing.T) {
	t.Parallel()

	if ServiceChatRateBurst <= 0 {
		t.Errorf("ServiceChatRateBurst = %d, want a positive burst", ServiceChatRateBurst)
	}

	if ServiceChatRateWindow <= 0 {
		t.Errorf("ServiceChatRateWindow = %v, want a positive window", ServiceChatRateWindow)
	}
}
