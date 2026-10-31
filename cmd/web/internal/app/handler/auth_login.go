package handler

import (
	"fmt"
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
)

// AuthLogin checks JSON credentials and writes a token issued for the user.
func (h *Handler) AuthLogin(rspWriter http.ResponseWriter, req *http.Request) error {
	creds, err := decodeCredentials(rspWriter, req)
	if err != nil {
		return fmt.Errorf("decode creds %w: %w", err, httperror.BadRequest("invalid credentials"))
	}

	token, err := h.loginUsecase.Login(req.Context(), creds.Username, creds.Password)
	if err != nil {
		return serviceError("login", creds.Username, err)
	}

	return writeToken(rspWriter, http.StatusOK, token)
}
