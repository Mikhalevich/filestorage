package handler

import (
	"net/http"
)

// FileDelete removes the file taken from the request path value.
func (h *Handler) FileDelete(rspWriter http.ResponseWriter, req *http.Request) error {
	filePath := req.PathValue(PathValueFilePath)

	if err := h.fileSvc.DeleteFile(filePath); err != nil {
		return serviceError("delete file", filePath, err)
	}

	rspWriter.WriteHeader(http.StatusNoContent)

	return nil
}
