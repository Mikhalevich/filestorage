package handler

import (
	"net/http"

	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

// DirectoryCreate creates the directory taken from the request path value.
func (h *Handler) DirectoryCreate(rspWriter http.ResponseWriter, req *http.Request) error {
	res, _ := resource.FromContext(req.Context())

	if err := h.fileSvc.CreateDirectory(res); err != nil {
		return serviceError("create directory", res.FullPath(), err)
	}

	rspWriter.WriteHeader(http.StatusCreated)

	return nil
}
