package tool

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/trace"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

type FileService interface {
	FileList(dirPath string) ([]file.File, error)
	OpenFile(filePath string) (file.File, io.ReadSeekCloser, error)
	SaveFile(dirPath string, fileName string, content io.Reader) (file.File, error)
	DeleteFile(filePath string) error
}

type Tool struct {
	fileSvc      FileService
	maxReadBytes int64
}

func New(fileSvc FileService, maxReadBytes int64) *Tool {
	return &Tool{
		fileSvc:      fileSvc,
		maxReadBytes: maxReadBytes,
	}
}

// Register adds all file storage tools to the server.
func (t *Tool) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_files",
		Description: "List regular files in a directory of the file storage, sorted by modification time (oldest first). " +
			"Subdirectories are not listed.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: new(false),
		},
	}, handle(t.ListFiles))

	mcp.AddTool(server, &mcp.Tool{
		Name: "read_file",
		Description: "Read a file from the file storage. UTF-8 files are returned as text, " +
			"other files as a base64 encoded embedded resource. Large files are rejected.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: new(false),
		},
	}, handle(t.ReadFile))

	mcp.AddTool(server, &mcp.Tool{
		Name: "write_file",
		Description: "Save content as a new file in an existing directory of the file storage. " +
			"Existing files are never overwritten: if the name is taken, a numeric suffix is added (a.txt -> a_1.txt). " +
			"Returns the name the file was stored with.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(false),
			OpenWorldHint:   new(false),
		},
	}, handle(t.WriteFile))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a regular file from the file storage. Directories are never deleted.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: new(true),
			IdempotentHint:  true,
			OpenWorldHint:   new(false),
		},
	}, handle(t.DeleteFile))
}

type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"     jsonschema:"file size in bytes"`
	ModTime time.Time `json:"mod_time" jsonschema:"last modification time"`
}

func toFileInfo(f file.File) FileInfo {
	return FileInfo{
		Name:    f.Name,
		Size:    f.Size,
		ModTime: f.ModTime,
	}
}

// handle logs errors returned by the tool and replaces them with messages safe to show to the client:
// known domain errors keep their message, other errors become generic "internal error".
func handle[In, Out any](h mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		res, out, err := h(ctx, req, input)
		if err != nil {
			logger.FromContext(ctx).WithError(err).Error("tool error")
			trace.SpanFromContext(ctx).RecordError(err)

			var zero Out

			return nil, zero, clientError(err)
		}

		return res, out, nil
	}
}

func clientError(err error) error {
	if perr, ok := perror.ParseError(err); ok && perr.Type != perror.TypeUnspecified {
		return errors.New(perr.Message)
	}

	return errors.New("internal error")
}
