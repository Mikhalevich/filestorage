package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
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
		return fileListError(dirPath, err)
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

func fileListError(dirPath string, err error) error {
	if perr, ok := perror.ParseError(err); ok {
		//nolint:exhaustive
		switch perr.Type {
		case perror.TypeNotFound:
			return httperror.NotFound(perr.Message)
		case perror.TypeInvalidArgument:
			return httperror.BadRequest(perr.Message)
		}
	}

	return fmt.Errorf("file list %q: %w", dirPath, err)
}
