package handler

import (
	"fmt"
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
)

type FileService interface {
	FileList(dirPath string) ([]file.File, error)
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
