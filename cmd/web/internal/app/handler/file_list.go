package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"syscall"
	"time"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
)

const (
	PathValueDirPath = "path"
)

type fileListResponse struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// FileList writes JSON list of files located in the directory taken from the request path value.
func (h *Handler) FileList(rspWriter http.ResponseWriter, r *http.Request) error {
	dirPath := r.PathValue(PathValueDirPath)

	files, err := h.fileSvc.FileList(dirPath)
	if err != nil {
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return httperror.NotFound("directory not found")
		case errors.Is(err, fs.ErrInvalid):
			return httperror.BadRequest("invalid directory path")
		case errors.Is(err, syscall.ENOTDIR):
			return httperror.BadRequest("not a directory")
		}

		return fmt.Errorf("file list %q: %w", dirPath, err)
	}

	fileList := make([]fileListResponse, 0, len(files))
	for _, f := range files {
		fileList = append(fileList, fileListResponse{
			Name:    f.Name,
			Size:    f.Size,
			ModTime: f.ModTime,
		})
	}

	rspWriter.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(rspWriter).Encode(fileList); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
