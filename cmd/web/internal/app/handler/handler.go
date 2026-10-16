package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

type FileService interface {
	FileList(dirPath string) ([]file.File, error)
	OpenFile(filePath string) (file.File, io.ReadSeekCloser, error)
}

type Handler struct {
	fileSvc FileService
}

func New(fileSvc FileService) *Handler {
	return &Handler{
		fileSvc: fileSvc,
	}
}

func (h *Handler) Test(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprintln(w, "It works!!!")

	return nil
}

func (h *Handler) ErrorTest(w http.ResponseWriter, r *http.Request) error {
	return httperror.InternalServerError("test error")
}

// serviceError converts known domain errors into http errors, other errors are wrapped with operation and path.
func serviceError(operation string, path string, err error) error {
	if perr, ok := perror.ParseError(err); ok {
		//nolint:exhaustive
		switch perr.Type {
		case perror.TypeNotFound:
			return httperror.NotFound(perr.Message)
		case perror.TypeInvalidArgument:
			return httperror.BadRequest(perr.Message)
		}
	}

	return fmt.Errorf("%s %q: %w", operation, path, err)
}
