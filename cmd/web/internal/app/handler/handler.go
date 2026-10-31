package handler

import (
	"context"
	"fmt"
	"io"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

type FileService interface {
	FileList(res resource.Resource) ([]file.File, error)
	OpenFile(res resource.Resource) (file.File, io.ReadSeekCloser, error)
	SaveFile(res resource.Resource, fileName string, content io.Reader) (file.File, error)
	DeleteFile(res resource.Resource) error
	CreateDirectory(res resource.Resource) error
}

type RegisterUsecase interface {
	Register(ctx context.Context, name string, password string) (string, error)
}

type LoginUsecase interface {
	Login(ctx context.Context, name string, password string) (string, error)
}

type Handler struct {
	fileSvc         FileService
	registerUsecase RegisterUsecase
	loginUsecase    LoginUsecase
}

func New(
	fileSvc FileService,
	registerUsecase RegisterUsecase,
	loginUsecase LoginUsecase,
) *Handler {
	return &Handler{
		fileSvc:         fileSvc,
		registerUsecase: registerUsecase,
		loginUsecase:    loginUsecase,
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
