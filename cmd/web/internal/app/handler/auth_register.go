package handler

import (
	"fmt"
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
)

// AuthRegister creates a user from JSON credentials and writes a token issued for it.
func (h *Handler) AuthRegister(rspWriter http.ResponseWriter, req *http.Request) error {
	creds, err := decodeCredentials(rspWriter, req)
	if err != nil {
		return fmt.Errorf("decode creds %w: %w", err, httperror.BadRequest("invalid credentials"))
	}

	token, err := h.registerUsecase.Register(req.Context(), creds.Username, creds.Password)
	if err != nil {
		return serviceError("register", creds.Username, err)
	}

	return writeToken(rspWriter, http.StatusCreated, token)
}
