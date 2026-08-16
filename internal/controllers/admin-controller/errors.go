package adminController

import "errors"

var (
	ErrSTTHealth   = errors.New("failed to check stt health")
	ErrSendMessage = errors.New("failed to send message to chat")
)
