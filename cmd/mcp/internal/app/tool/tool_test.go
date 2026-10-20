package tool_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/cmd/mcp/internal/app/tool"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

const (
	testMaxReadBytes = 16
)

func setupRoot(t *testing.T) string {
	t.Helper()

	rootDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "a.txt"), []byte("aaa"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "bin.dat"), []byte{0xff, 0x00, 0xfe}, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "big.txt"), make([]byte, testMaxReadBytes+1), 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(rootDir, "sub"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "sub", "c.txt"), []byte("c"), 0o600))

	return rootDir
}

// connect starts mcp server with file storage tools over rootDir and returns connected client session.
func connect(t *testing.T, rootDir string) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test-server"}, nil)
	tool.New(filesvc.New(rootDir), testMaxReadBytes).Register(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)

	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, clientSession.Close())
		require.NoError(t, serverSession.Wait())
	})

	return clientSession
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)

	return res
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()

	require.Len(t, res.Content, 1)

	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "content type %T", res.Content[0])

	return text.Text
}

func listedNames(t *testing.T, res *mcp.CallToolResult) []string {
	t.Helper()

	types := listedTypes(t, res)

	names := make([]string, 0, len(types))
	for _, entry := range types {
		names = append(names, entry.name)
	}

	return names
}

type listedEntry struct {
	name      string
	entryType string
}

// listedTypes returns names and types of list_files entries in the result order.
func listedTypes(t *testing.T, res *mcp.CallToolResult) []listedEntry {
	t.Helper()

	require.False(t, res.IsError, "tool error: %v", res.Content)

	structured, ok := res.StructuredContent.(map[string]any)
	require.True(t, ok, "structured content type %T", res.StructuredContent)

	files, ok := structured["files"].([]any)
	require.True(t, ok, "files type %T", structured["files"])

	entries := make([]listedEntry, 0, len(files))

	for _, f := range files {
		fileInfo, ok := f.(map[string]any)
		require.True(t, ok)

		name, ok := fileInfo["name"].(string)
		require.True(t, ok)

		entryType, ok := fileInfo["type"].(string)
		require.True(t, ok)

		entries = append(entries, listedEntry{name: name, entryType: entryType})
	}

	return entries
}

func TestListTools(t *testing.T) {
	t.Parallel()

	res, err := connect(t, setupRoot(t)).ListTools(t.Context(), nil)
	require.NoError(t, err)

	names := make([]string, 0, len(res.Tools))
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
	}

	assert.ElementsMatch(t, []string{"list_files", "read_file", "write_file", "delete_file", "create_directory"}, names)
}

func TestListFiles(t *testing.T) {
	t.Parallel()

	session := connect(t, setupRoot(t))

	tests := []struct {
		name string
		args map[string]any
		want []string
	}{
		{name: "root no path", args: map[string]any{}, want: []string{"sub", "a.txt", "bin.dat", "big.txt"}},
		{name: "root slash", args: map[string]any{"path": "/"}, want: []string{"sub", "a.txt", "bin.dat", "big.txt"}},
		{name: "subdir", args: map[string]any{"path": "sub"}, want: []string{"c.txt"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.ElementsMatch(t, tc.want, listedNames(t, callTool(t, session, "list_files", tc.args)))
		})
	}
}

func TestListFilesTypes(t *testing.T) {
	t.Parallel()

	entries := listedTypes(t, callTool(t, connect(t, setupRoot(t)), "list_files", map[string]any{}))
	require.NotEmpty(t, entries)

	assert.Equal(t, listedEntry{name: "sub", entryType: "directory"}, entries[0])
	assert.Contains(t, entries, listedEntry{name: "a.txt", entryType: "file"})
}

func TestListFilesEmptyDir(t *testing.T) {
	t.Parallel()

	assert.Empty(t, listedNames(t, callTool(t, connect(t, t.TempDir()), "list_files", map[string]any{})))
}

func TestToolErrors(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	session := connect(t, filepath.Join(rootDir, "sub"))

	tests := []struct {
		name    string
		tool    string
		args    map[string]any
		wantMsg string
	}{
		{name: "list not found", tool: "list_files", args: map[string]any{"path": "missing"}, wantMsg: "directory not found"},
		{name: "list escape root", tool: "list_files", args: map[string]any{"path": ".."}, wantMsg: "invalid directory path"},
		{name: "read not found", tool: "read_file", args: map[string]any{"path": "missing.txt"}, wantMsg: "file not found"},
		{name: "read escape root", tool: "read_file", args: map[string]any{"path": "../a.txt"}, wantMsg: "invalid file path"},
		{
			name:    "write invalid name",
			tool:    "write_file",
			args:    map[string]any{"name": "../x.txt", "content": "x"},
			wantMsg: "invalid file name",
		},
		{
			name:    "write invalid base64",
			tool:    "write_file",
			args:    map[string]any{"name": "x.txt", "content": "!!", "base64": true},
			wantMsg: "invalid base64 content",
		},
		{
			name:    "delete not found",
			tool:    "delete_file",
			args:    map[string]any{"path": "missing.txt"},
			wantMsg: "file not found",
		},
		{name: "delete dir", tool: "delete_file", args: map[string]any{"path": "."}, wantMsg: "not a regular file"},
		{
			name:    "create dir parent not found",
			tool:    "create_directory",
			args:    map[string]any{"path": "missing/new"},
			wantMsg: "parent directory not found",
		},
		{
			name:    "create dir exists",
			tool:    "create_directory",
			args:    map[string]any{"path": "c.txt"},
			wantMsg: "directory already exists",
		},
		{
			name:    "create dir escape root",
			tool:    "create_directory",
			args:    map[string]any{"path": "../x"},
			wantMsg: "invalid directory path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := callTool(t, session, tc.tool, tc.args)

			assert.True(t, res.IsError)
			assert.Equal(t, tc.wantMsg, resultText(t, res))
		})
	}

	assert.FileExists(t, filepath.Join(rootDir, "a.txt"))
	assert.NoFileExists(t, filepath.Join(rootDir, "x.txt"))
	assert.NoDirExists(t, filepath.Join(rootDir, "x"))
}

func TestToolInternalError(t *testing.T) {
	t.Parallel()

	res := callTool(t, connect(t, filepath.Join(t.TempDir(), "missing-root")), "list_files", map[string]any{})

	assert.True(t, res.IsError)
	assert.Equal(t, "internal error", resultText(t, res))
}

func TestReadFileText(t *testing.T) {
	t.Parallel()

	session := connect(t, setupRoot(t))

	for path, want := range map[string]string{"a.txt": "aaa", "/sub/c.txt": "c"} {
		res := callTool(t, session, "read_file", map[string]any{"path": path})

		require.False(t, res.IsError, "path %q: %v", path, res.Content)
		assert.Equal(t, want, resultText(t, res), "path %q", path)
	}
}

func TestReadFileBinary(t *testing.T) {
	t.Parallel()

	res := callTool(t, connect(t, setupRoot(t)), "read_file", map[string]any{"path": "bin.dat"})

	require.False(t, res.IsError, "%v", res.Content)
	require.Len(t, res.Content, 1)

	resource, ok := res.Content[0].(*mcp.EmbeddedResource)
	require.True(t, ok, "content type %T", res.Content[0])

	assert.Equal(t, "filestorage:///bin.dat", resource.Resource.URI)
	assert.Equal(t, "application/octet-stream", resource.Resource.MIMEType)
	assert.Equal(t, []byte{0xff, 0x00, 0xfe}, resource.Resource.Blob)
}

func TestReadFileTooLarge(t *testing.T) {
	t.Parallel()

	res := callTool(t, connect(t, setupRoot(t)), "read_file", map[string]any{"path": "big.txt"})

	assert.True(t, res.IsError)
	assert.Equal(t, "file is larger than 16 bytes", resultText(t, res))
}

func TestWriteFile(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	session := connect(t, rootDir)

	tests := []struct {
		name     string
		args     map[string]any
		wantName string
		wantPath string
		want     []byte
	}{
		{
			name:     "new file",
			args:     map[string]any{"name": "new.txt", "content": "new"},
			wantName: "new.txt",
			wantPath: "new.txt",
			want:     []byte("new"),
		},
		{
			name:     "rename taken",
			args:     map[string]any{"name": "a.txt", "content": "renamed"},
			wantName: "a_1.txt",
			wantPath: "a_1.txt",
			want:     []byte("renamed"),
		},
		{
			name:     "subdir",
			args:     map[string]any{"dir": "sub", "name": "d.txt", "content": ""},
			wantName: "d.txt",
			wantPath: filepath.Join("sub", "d.txt"),
			want:     []byte{},
		},
		{
			name: "base64",
			args: map[string]any{
				"name":    "bin2.dat",
				"content": base64.StdEncoding.EncodeToString([]byte{0x01, 0xff}),
				"base64":  true,
			},
			wantName: "bin2.dat",
			wantPath: "bin2.dat",
			want:     []byte{0x01, 0xff},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := callTool(t, session, "write_file", tc.args)
			require.False(t, res.IsError, "%v", res.Content)

			structured, ok := res.StructuredContent.(map[string]any)
			require.True(t, ok, "structured content type %T", res.StructuredContent)
			assert.Equal(t, tc.wantName, structured["name"])
			assert.InDelta(t, len(tc.want), structured["size"], 0)
			assert.Equal(t, "file", structured["type"])

			data, err := os.ReadFile(filepath.Join(rootDir, tc.wantPath))
			require.NoError(t, err)
			assert.Equal(t, tc.want, data)
		})
	}

	data, err := os.ReadFile(filepath.Join(rootDir, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "aaa", string(data))
}

func TestDeleteFile(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	session := connect(t, rootDir)

	res := callTool(t, session, "delete_file", map[string]any{"path": "sub/c.txt"})
	require.False(t, res.IsError, "%v", res.Content)
	assert.Equal(t, "deleted sub/c.txt", resultText(t, res))

	assert.NoFileExists(t, filepath.Join(rootDir, "sub", "c.txt"))
	assert.DirExists(t, filepath.Join(rootDir, "sub"))
	assert.Empty(t, listedNames(t, callTool(t, session, "list_files", map[string]any{"path": "sub"})))
}

func TestCreateDirectory(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	session := connect(t, rootDir)

	res := callTool(t, session, "create_directory", map[string]any{"path": "sub/new"})
	require.False(t, res.IsError, "%v", res.Content)
	assert.Equal(t, "created sub/new", resultText(t, res))
	assert.DirExists(t, filepath.Join(rootDir, "sub", "new"))

	res = callTool(t, session, "write_file", map[string]any{"dir": "sub/new", "name": "d.txt", "content": "d"})
	require.False(t, res.IsError, "%v", res.Content)

	listed := callTool(t, session, "list_files", map[string]any{"path": "sub/new"})
	assert.Equal(t, []string{"d.txt"}, listedNames(t, listed))
}
