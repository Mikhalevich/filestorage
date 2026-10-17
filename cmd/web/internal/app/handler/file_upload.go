package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
)

// FileUpload stores files from multipart/form-data request body in the directory taken from the request path value.
// Parts without file name are skipped. Files are stored one by one: if one of them fails,
// previously stored files are kept. Writes JSON list of stored files, their names may differ
// from the requested ones when those are already taken.
func (h *Handler) FileUpload(rspWriter http.ResponseWriter, req *http.Request) error {
	dirPath := req.PathValue(PathValueDirPath)

	reader, err := req.MultipartReader()
	if err != nil {
		return fmt.Errorf("%w: %w", httperror.BadRequest("multipart/form-data body expected"), err)
	}

	var stored []fileListResponse

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("%w: %w", httperror.BadRequest("invalid multipart body"), err)
		}

		if part.FileName() == "" {
			continue
		}

		info, err := h.fileSvc.SaveFile(dirPath, part.FileName(), part)
		if err != nil {
			return serviceError("save file", dirPath, err)
		}

		stored = append(stored, fileListResponse{
			Name:    info.Name,
			Size:    info.Size,
			ModTime: info.ModTime,
		})
	}

	if len(stored) == 0 {
		return httperror.BadRequest("no files in request")
	}

	rspWriter.Header().Set("Content-Type", "application/json")
	rspWriter.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(rspWriter).Encode(stored); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}
