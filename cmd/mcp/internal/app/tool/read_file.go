package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

const (
	resourceURIPrefix = "filestorage:///"
)

type ReadFileInput struct {
	Path string `json:"path" jsonschema:"file path relative to the storage root"`
}

// ReadFile returns content of the input file: valid UTF-8 as text, anything else as a blob resource.
// Files larger than the configured limit are rejected.
func (t *Tool) ReadFile(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input ReadFileInput,
) (*mcp.CallToolResult, any, error) {
	info, content, err := t.fileSvc.OpenFile(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("open file %q: %w", input.Path, err)
	}

	defer content.Close()

	if info.Size > t.maxReadBytes {
		return nil, nil, fmt.Errorf("check size %q: %w", input.Path, t.tooLargeError())
	}

	// the file may grow after open, so never read more than the limit.
	data, err := io.ReadAll(io.LimitReader(content, t.maxReadBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read file %q: %w", input.Path, err)
	}

	if int64(len(data)) > t.maxReadBytes {
		return nil, nil, fmt.Errorf("check read size %q: %w", input.Path, t.tooLargeError())
	}

	if utf8.Valid(data) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.EmbeddedResource{
			Resource: &mcp.ResourceContents{
				URI:      resourceURIPrefix + strings.TrimPrefix(input.Path, "/"),
				MIMEType: http.DetectContentType(data),
				Blob:     data,
			},
		}},
	}, nil, nil
}

func (t *Tool) tooLargeError() error {
	return perror.InvalidArgument(fmt.Sprintf("file is larger than %d bytes", t.maxReadBytes))
}
