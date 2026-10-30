package handler

import (
	"io"
	"net/http"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

type FileService interface {
	SaveFile(res resource.Resource, fileName string, content io.Reader) (file.File, error)
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
