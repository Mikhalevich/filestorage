package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema:"directory path relative to the storage root, empty for the root"`
}

type ListFilesOutput struct {
	Files []FileInfo `json:"files"`
}

// ListFiles returns regular files located in the input directory.
func (t *Tool) ListFiles(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input ListFilesInput,
) (*mcp.CallToolResult, ListFilesOutput, error) {
	files, err := t.fileSvc.FileList(input.Path)
	if err != nil {
		return nil, ListFilesOutput{}, fmt.Errorf("file list %q: %w", input.Path, err)
	}

	output := ListFilesOutput{
		Files: make([]FileInfo, 0, len(files)),
	}

	for _, f := range files {
		output.Files = append(output.Files, toFileInfo(f))
	}

	return nil, output, nil
}
