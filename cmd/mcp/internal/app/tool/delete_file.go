package tool

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DeleteFileInput struct {
	Path string `json:"path" jsonschema:"file path relative to the storage root"`
}

// DeleteFile removes the input file.
func (t *Tool) DeleteFile(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input DeleteFileInput,
) (*mcp.CallToolResult, any, error) {
	if err := t.fileSvc.DeleteFile(input.Path); err != nil {
		return nil, nil, fmt.Errorf("delete file %q: %w", input.Path, err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("deleted %s", input.Path)}},
	}, nil, nil
}
