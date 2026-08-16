package telegram

import "errors"

var (
	ErrGetFile          = errors.New("failed to get telegram file info")
	ErrEmptyFilePath    = errors.New("telegram returned empty file path")
	ErrBuildRequest     = errors.New("failed to build telegram download request")
	ErrDownloadFailed   = errors.New("failed to download telegram file")
	ErrUnexpectedStatus = errors.New("telegram file download returned unexpected status")
	ErrSendMessage      = errors.New("failed to send telegram message")
)
