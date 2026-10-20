package handler

import (
	"net/http"
)

// DirectoryCreate creates the directory taken from the request path value.
func (h *Handler) DirectoryCreate(rspWriter http.ResponseWriter, req *http.Request) error {
	dirPath := req.PathValue(PathValueDirPath)

	if err := h.fileSvc.CreateDirectory(dirPath); err != nil {
		return serviceError("create directory", dirPath, err)
	}

	rspWriter.WriteHeader(http.StatusCreated)

	return nil
}
