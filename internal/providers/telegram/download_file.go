package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/go-telegram/bot"
)

// DownloadFile resolves fileID via getFile and downloads the file content.
// The caller must Close the returned ReadCloser.
func (p *Provider) DownloadFile(ctx context.Context, fileID string) (io.ReadCloser, error) {
	file, err := p.api.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, p.wrapRedacted(ErrGetFile, err)
	}

	if file.FilePath == "" {
		return nil, ErrEmptyFilePath
	}

	url := fileBaseURL + "/file/bot" + p.token + "/" + file.FilePath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, p.wrapRedacted(ErrBuildRequest, err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, p.wrapRedacted(ErrDownloadFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		return nil, fmt.Errorf("%w: status %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	return resp.Body, nil
}
