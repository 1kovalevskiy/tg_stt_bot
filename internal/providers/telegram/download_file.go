package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/1kovalevskiy/tg_stt_bot/internal/providers"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// DownloadFile resolves fileID via getFile and downloads the file content.
// The caller must Close the returned ReadCloser.
//
// The download timeout has to cover the caller's read of the body, so its
// cancel is deliberately not deferred: on success it is owned by the returned
// ReadCloser and runs on Close, and every error path below calls it before
// returning.
func (p *Provider) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	file, err := p.getFile(ctx, fileID)
	if err != nil {
		return nil, err
	}

	// A nil file with a nil error is not something the Bot API promises, but
	// the library types it as possible and dereferencing it here would take
	// the whole process down: the update workers have no recover().
	if file == nil || file.FilePath == "" {
		return nil, providers.ErrTelegramEmptyFilePath
	}

	url := fileBaseURL + "/file/bot" + p.config.GetTelegramToken() + "/" + file.FilePath

	ctx, cancel := context.WithTimeout(ctx, p.config.GetTelegramDownloadTimeout())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel()

		return nil, p.wrapRedacted(providers.ErrTelegramBuildRequest, err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		cancel()

		return nil, p.wrapRedacted(providers.ErrTelegramDownloadFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		cancel()

		return nil, fmt.Errorf("%w: status %d", providers.ErrTelegramUnexpectedStatus, resp.StatusCode)
	}

	return newCancelReadCloser(resp.Body, cancel), nil
}

// getFile resolves the file metadata, bounded by the Bot API timeout: the
// response is fully read before this returns, so the context is released here.
func (p *Provider) getFile(ctx context.Context, fileID string) (*tgmodels.File, error) {
	ctx, cancel := context.WithTimeout(ctx, p.config.GetTelegramAPITimeout())
	defer cancel()

	file, err := p.api.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, p.wrapRedacted(providers.ErrTelegramGetFile, err)
	}

	return file, nil
}
