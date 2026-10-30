package tool

import (
	"context"
	"fmt"

	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CreateDirectoryInput struct {
	Path string `json:"path" jsonschema:"directory path relative to the storage root"`
}

// CreateDirectory creates the input directory, its parent must exist.
func (t *Tool) CreateDirectory(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input CreateDirectoryInput,
) (*mcp.CallToolResult, any, error) {
	if err := t.fileSvc.CreateDirectory(resource.NewPublicResource(input.Path)); err != nil {
		return nil, nil, fmt.Errorf("create directory %q: %w", input.Path, err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("created %s", input.Path)}},
	}, nil, nil
}
