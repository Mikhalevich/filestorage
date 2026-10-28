package handler

import (
	"io"
	"net/http"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
)

type FileService interface {
	SaveFile(dirPath string, fileName string, content io.Reader) (file.File, error)
}

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Handler struct {
	fileSvc    FileService
	uploadDoer Doer
}

func New(
	fileSvc FileService,
	uploadDoer Doer,
) *Handler {
	return &Handler{
		fileSvc:    fileSvc,
		uploadDoer: uploadDoer,
	}
}
