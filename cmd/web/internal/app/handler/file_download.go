package handler

import (
	"fmt"
	"net/http"

	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

// FileDownload streams the file taken from the request path value as an attachment.
// Content type, range and conditional requests are handled by http.ServeContent.
func (h *Handler) FileDownload(rspWriter http.ResponseWriter, req *http.Request) error {
	res, _ := resource.FromContext(req.Context())

	info, content, err := h.fileSvc.OpenFile(res.FullPath())
	if err != nil {
		return serviceError("open file", res.FullPath(), err)
	}

	defer content.Close()

	rspWriter.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", info.Name),
	)

	http.ServeContent(rspWriter, req, info.Name, info.ModTime, content)

	return nil
}
