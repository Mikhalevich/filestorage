package handler

import (
	"fmt"
	"net/http"
)

const (
	PathValueFilePath = "path"
)

// FileDownload streams the file taken from the request path value as an attachment.
// Content type, range and conditional requests are handled by http.ServeContent.
func (h *Handler) FileDownload(rspWriter http.ResponseWriter, req *http.Request) error {
	filePath := req.PathValue(PathValueFilePath)

	info, content, err := h.fileSvc.OpenFile(filePath)
	if err != nil {
		return serviceError("open file", filePath, err)
	}

	defer content.Close()

	rspWriter.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", info.Name),
	)

	http.ServeContent(rspWriter, req, info.Name, info.ModTime, content)

	return nil
}
