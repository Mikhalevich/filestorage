package handler

import (
	"fmt"
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
)

type Handler struct {
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Test(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprintln(w, "It works!!!")

	return nil
}

func (h *Handler) ErrorTest(w http.ResponseWriter, r *http.Request) error {
	return httperror.InternalServerError("test error")
}
