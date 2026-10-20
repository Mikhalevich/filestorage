package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
)

const (
	PathValueDirPath = "path"
)

type fileListResponse struct {
	Name    string    `json:"name"`
	Type    string    `json:"type"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

func toFileListResponse(entry file.File) fileListResponse {
	return fileListResponse{
		Name:    entry.Name,
		Type:    entry.Type.String(),
		Size:    entry.Size,
		ModTime: entry.ModTime,
	}
}

// FileList writes JSON list of files and subdirectories located in the directory taken from the request path value.
func (h *Handler) FileList(rspWriter http.ResponseWriter, r *http.Request) error {
	dirPath := r.PathValue(PathValueDirPath)

	files, err := h.fileSvc.FileList(dirPath)
	if err != nil {
		return serviceError("file list", dirPath, err)
	}

	fileList := make([]fileListResponse, 0, len(files))
	for _, f := range files {
		fileList = append(fileList, toFileListResponse(f))
	}

	rspWriter.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(rspWriter).Encode(fileList); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
