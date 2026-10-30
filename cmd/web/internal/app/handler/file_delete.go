package handler

import (
	"net/http"

	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

// FileDelete removes the file taken from the request path value.
func (h *Handler) FileDelete(rspWriter http.ResponseWriter, req *http.Request) error {
	res, _ := resource.FromContext(req.Context())

	if err := h.fileSvc.DeleteFile(res.FullPath()); err != nil {
		return serviceError("delete file", res.FullPath(), err)
	}

	rspWriter.WriteHeader(http.StatusNoContent)

	return nil
}
