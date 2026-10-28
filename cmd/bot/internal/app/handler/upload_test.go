package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mikhalevich/tgbot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/cmd/bot/internal/app/handler"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

const (
	testChatID = 42
)

// fakeOperations serves file content by file id from test http server and records sent messages.
type fakeOperations struct {
	serverURL string
	messages  []string
}

func (f *fakeOperations) SendMessage(_ context.Context, chatID int64, msg string) {
	if chatID == testChatID {
		f.messages = append(f.messages, msg)
	}
}

func (f *fakeOperations) DeleteMessage(context.Context, int64, int) {
}

func (f *fakeOperations) GetDownloadLink(_ context.Context, fileID string) (string, error) {
	if fileID == "missing" {
		return "", errors.New("file not found")
	}

	return f.serverURL + "/" + fileID, nil
}

// setup returns handler over a temp storage root, fake operations and the root path.
func setup(t *testing.T) (*handler.Handler, *fakeOperations, string) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/broken" {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		//nolint:gosec
		_, _ = w.Write([]byte("content of " + r.URL.Path[1:]))
	}))
	t.Cleanup(srv.Close)

	rootDir := t.TempDir()

	return handler.New(filesvc.New(rootDir), srv.Client()),
		&fakeOperations{serverURL: srv.URL},
		rootDir
}

func documentMessage(fileID, fileName string) tgbot.Message {
	return tgbot.Message{
		Type:   tgbot.TypeDocument,
		ChatID: testChatID,
		Document: tgbot.Document{
			FileID:   fileID,
			FileName: fileName,
			FileSize: 10,
		},
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(content)
}

func TestUploadDocument(t *testing.T) {
	t.Parallel()

	hndlr, ops, rootDir := setup(t)

	require.NoError(t, hndlr.Upload(t.Context(), documentMessage("doc1", "a.txt"), ops))

	assert.Equal(t, "content of doc1", readFile(t, filepath.Join(rootDir, "a.txt")))
	assert.Equal(t, []string{"saved as a.txt (15 bytes)"}, ops.messages)
}

func TestUploadDocumentDuplicateName(t *testing.T) {
	t.Parallel()

	hndlr, ops, rootDir := setup(t)

	require.NoError(t, hndlr.Upload(t.Context(), documentMessage("doc1", "a.txt"), ops))
	require.NoError(t, hndlr.Upload(t.Context(), documentMessage("doc2", "a.txt"), ops))

	assert.Equal(t, "content of doc1", readFile(t, filepath.Join(rootDir, "a.txt")))
	assert.Equal(t, "content of doc2", readFile(t, filepath.Join(rootDir, "a_1.txt")))
	assert.Equal(t, []string{"saved as a.txt (15 bytes)", "saved as a_1.txt (15 bytes)"}, ops.messages)
}

func TestUploadDocumentSanitizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{name: "parent dir", fileName: "../x.txt", expected: "x.txt"},
		{name: "nested path", fileName: "/etc/passwd", expected: "passwd"},
		{name: "parent dir only", fileName: "..", expected: "doc1"},
		{name: "empty", fileName: "", expected: "doc1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hndlr, ops, rootDir := setup(t)

			require.NoError(t, hndlr.Upload(t.Context(), documentMessage("doc1", tc.fileName), ops))

			assert.Equal(t, "content of doc1", readFile(t, filepath.Join(rootDir, tc.expected)))
		})
	}
}

func TestUploadPhoto(t *testing.T) {
	t.Parallel()

	hndlr, ops, rootDir := setup(t)

	require.NoError(t, hndlr.Upload(t.Context(), tgbot.Message{
		Type:   tgbot.TypePhoto,
		ChatID: testChatID,
		Photo: []tgbot.PhotoSize{
			{FileID: "small", Width: 10, Height: 10},
			{FileID: "large", Width: 100, Height: 100},
			{FileID: "medium", Width: 50, Height: 50},
		},
	}, ops))

	assert.Equal(t, "content of large", readFile(t, filepath.Join(rootDir, "large.jpg")))
	assert.Equal(t, []string{"saved as large.jpg (16 bytes)"}, ops.messages)
}

func TestUploadTextIgnored(t *testing.T) {
	t.Parallel()

	hndlr, ops, _ := setup(t)

	require.NoError(t, hndlr.Upload(t.Context(), tgbot.Message{
		Type:   tgbot.TypeText,
		ChatID: testChatID,
		Text:   "hello",
	}, ops))

	assert.Empty(t, ops.messages)
}

func TestUploadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fileID string
	}{
		{name: "download link", fileID: "missing"},
		{name: "http status", fileID: "broken"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hndlr, ops, rootDir := setup(t)

			require.Error(t, hndlr.Upload(t.Context(), documentMessage(tc.fileID, "a.txt"), ops))

			entries, err := os.ReadDir(rootDir)
			require.NoError(t, err)
			assert.Empty(t, entries)
			assert.Empty(t, ops.messages)
		})
	}
}

func TestUploadErrorHidesLink(t *testing.T) {
	t.Parallel()

	const secret = "123:SECRET-token"

	tests := []struct {
		name      string
		serverURL string
	}{
		{name: "invalid link", serverURL: "http://bad host/file/bot" + secret},
		{name: "connection refused", serverURL: "http://127.0.0.1:1/file/bot" + secret},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hndlr, ops, _ := setup(t)
			ops.serverURL = tc.serverURL

			err := hndlr.Upload(t.Context(), documentMessage("doc1", "a.txt"), ops)
			require.Error(t, err)

			assert.NotContains(t, err.Error(), secret)
			assert.Contains(t, err.Error(), "<telegram file link>")

			_, isURLErr := errors.AsType[*url.Error](err)
			assert.True(t, isURLErr, "error chain is kept")
		})
	}
}
