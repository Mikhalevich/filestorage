package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/Mikhalevich/tgbot"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

const (
	// MaxFileSize is the maximum size of a file the bot api allows to download.
	MaxFileSize = 20 * 1024 * 1024

	// uploadDir is the storage directory received files are saved into.
	uploadDir = ""
)

type FileService interface {
	SaveFile(dirPath string, fileName string, content io.Reader) (file.File, error)
}

type Handler struct {
	fileSvc    FileService
	httpClient *http.Client
}

func New(fileSvc FileService, httpClient *http.Client) *Handler {
	return &Handler{
		fileSvc:    fileSvc,
		httpClient: httpClient,
	}
}

// Upload saves document or photo from the message into the file storage and replies with the stored file name.
// Known domain errors are sent to the chat, other errors are returned (tgbot replies with generic error).
func (h *Handler) Upload(ctx context.Context, msg tgbot.Message, operations tgbot.Operations) error {
	var fileID, fileName string

	//nolint:exhaustive
	switch msg.Type {
	case tgbot.TypeDocument:
		if msg.Document.FileSize > MaxFileSize {
			operations.SendMessage(ctx, msg.ChatID, fmt.Sprintf("file is too large, max size is %d bytes", MaxFileSize))

			return nil
		}

		fileID = msg.Document.FileID
		fileName = documentFileName(msg.Document)
	case tgbot.TypePhoto:
		photo, ok := msg.LargestPhoto()
		if !ok {
			return fmt.Errorf("photo message %d without photo sizes", msg.MessageID)
		}

		fileID = photo.FileID
		fileName = photo.FileID + ".jpg"
	default:
		operations.SendMessage(ctx, msg.ChatID, "send a document or photo to upload it")

		return nil
	}

	stored, err := h.uploadFile(ctx, operations, fileID, fileName)
	if err != nil {
		if perr, ok := perror.ParseError(err); ok && perr.Type != perror.TypeUnspecified {
			operations.SendMessage(ctx, msg.ChatID, perr.Message)

			return nil
		}

		return fmt.Errorf("upload file %q: %w", fileName, err)
	}

	operations.SendMessage(ctx, msg.ChatID, fmt.Sprintf("saved as %s (%d bytes)", stored.Name, stored.Size))

	return nil
}

func (h *Handler) uploadFile(
	ctx context.Context,
	operations tgbot.Operations,
	fileID string,
	fileName string,
) (file.File, error) {
	link, err := operations.GetDownloadLink(ctx, fileID)
	if err != nil {
		return file.File{}, fmt.Errorf("get download link: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return file.File{}, fmt.Errorf("create http request: %w", err)
	}

	rsp, err := h.httpClient.Do(req)
	if err != nil {
		return file.File{}, fmt.Errorf("do http request: %w", err)
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return file.File{}, fmt.Errorf("unexpected http status: %s", rsp.Status)
	}

	stored, err := h.fileSvc.SaveFile(uploadDir, fileName, rsp.Body)
	if err != nil {
		return file.File{}, fmt.Errorf("save file: %w", err)
	}

	return stored, nil
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
