package tool

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

type WriteFileInput struct {
	Dir     string `json:"dir,omitempty"    jsonschema:"existing directory relative to the storage root, empty for root"`
	Name    string `json:"name"             jsonschema:"file name, a single path element"`
	Content string `json:"content"          jsonschema:"file content: plain text, or base64 when base64 is true"`
	Base64  bool   `json:"base64,omitempty" jsonschema:"content is base64 encoded (standard encoding with padding)"`
}

// WriteFile stores input content as a new file, adding a numeric suffix to the name if it is taken.
func (t *Tool) WriteFile(
	_ context.Context,
	_ *mcp.CallToolRequest,
	input WriteFileInput,
) (*mcp.CallToolResult, FileInfo, error) {
	var content io.Reader = strings.NewReader(input.Content)

	if input.Base64 {
		data, err := base64.StdEncoding.DecodeString(input.Content)
		if err != nil {
			return nil, FileInfo{}, fmt.Errorf("decode content: %w: %w", perror.InvalidArgument("invalid base64 content"), err)
		}

		content = bytes.NewReader(data)
	}

	info, err := t.fileSvc.SaveFile(resource.NewPublicResource(input.Dir), input.Name, content)
	if err != nil {
		return nil, FileInfo{}, fmt.Errorf("save file %q in %q: %w", input.Name, input.Dir, err)
	}

	return nil, toFileInfo(info), nil
}
