package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/Mikhalevich/tgbot"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

const (
	// uploadDir is the storage directory received files are saved into.
	defaultUploadDir = ""

	// hiddenURL replaces download link in errors.
	hiddenURL = "<telegram file link>"

	// maxPhotoFileIDSuffixLen is how many trailing file id characters go into a photo file name.
	maxPhotoFileIDSuffixLen = 10
)

// Upload saves document or photo from the message into the file storage and replies with the stored file name.
// Known domain errors are sent to the chat, other errors are returned (tgbot replies with generic error).
func (h *Handler) Upload(
	ctx context.Context,
	msg tgbot.Message,
	operations tgbot.Operations,
) error {
	var (
		fileID   string
		fileName string
	)

	//nolint:exhaustive
	switch msg.Type {
	case tgbot.TypeDocument:
		fileID = msg.Document.FileID
		fileName = documentFileName(msg.Document)
	case tgbot.TypePhoto:
		photo, ok := msg.LargestPhoto()
		if !ok {
			return fmt.Errorf("photo message %d without photo sizes", msg.MessageID)
		}

		fileID = photo.FileID
		fileName = photoFileName(photo)
	default:
		return nil
	}

	link, err := operations.GetDownloadLink(ctx, fileID)
	if err != nil {
		return fmt.Errorf("get download link: %w", err)
	}

	stored, err := h.uploadFile(ctx, link, defaultUploadDir, fileName)
	if err != nil {
		if perr, ok := perror.ParseError(err); ok && perr.Type != perror.TypeUnspecified {
			operations.SendMessage(ctx, msg.ChatID, perr.Message)

			return nil
		}

		return fmt.Errorf("upload file %q: %w", fileName, err)
	}

	operations.SendMessage(
		ctx,
		msg.ChatID,
		fmt.Sprintf("saved as %s (%s)", stored.Name, formatSizeSI(stored.Size)),
	)

	return nil
}

func (h *Handler) uploadFile(
	ctx context.Context,
	link string,
	dir string,
	fileName string,
) (file.File, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return file.File{}, fmt.Errorf("create http request: %w", hideURL(err))
	}

	rsp, err := h.uploadDoer.Do(req)
	if err != nil {
		return file.File{}, fmt.Errorf("do http request: %w", hideURL(err))
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return file.File{}, fmt.Errorf("unexpected http status: %s", rsp.Status)
	}

	stored, err := h.fileSvc.SaveFile(dir, fileName, rsp.Body)
	if err != nil {
		return file.File{}, fmt.Errorf("save file: %w", err)
	}

	return stored, nil
}

// hideURL replaces the request url in err, as telegram file download link contains the bot token.
// The error chain is kept, so callers can still match the underlying cause.
func hideURL(err error) error {
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		urlErr.URL = hiddenURL
	}

	return err
}

// documentFileName strips directory components from the user provided file name
// and falls back to the file id when nothing usable is left.
func documentFileName(doc tgbot.Document) string {
	fileName := path.Base(doc.FileName)
	if fileName == "." || fileName == ".." || fileName == "/" {
		return doc.FileID
	}

	return fileName
}

func photoFileName(photo tgbot.PhotoSize) string {
	return fmt.Sprintf("%s_%s.jpg",
		time.Now().UTC().Format("20060102_150405"),
		trimLen(photo.FileID, maxPhotoFileIDSuffixLen),
	)
}

func trimLen(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}

	return name[len(name)-maxLen:]
}

func formatSizeSI(size int64) string {
	const unit = 1000
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %cB", float64(size)/float64(div), "kMGTPE"[exp])
}
