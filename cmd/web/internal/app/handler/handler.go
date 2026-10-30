package handler

import (
	"fmt"
	"io"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

type FileService interface {
	FileList(dirPath string) ([]file.File, error)
	OpenFile(filePath string) (file.File, io.ReadSeekCloser, error)
	SaveFile(dirPath string, fileName string, content io.Reader) (file.File, error)
	DeleteFile(filePath string) error
	CreateDirectory(dirPath string) error
}

type Handler struct {
	fileSvc FileService
}

func New(fileSvc FileService) *Handler {
	return &Handler{
		fileSvc: fileSvc,
	}
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
		case perror.TypeAlreadyExists:
			return httperror.Conflict(perr.Message)
		case perror.TypeUnauthorized:
			return httperror.Unauthorized(perr.Message)
		}
	}

	return fmt.Errorf("%s %q: %w", operation, path, err)
}
