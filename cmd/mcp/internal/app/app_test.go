package app_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/Mikhalevich/filestorage/cmd/mcp/internal/app"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

type recordingTracer struct {
	mu    sync.Mutex
	names []string
}

func (r *recordingTracer) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	r.mu.Lock()
	r.names = append(r.names, spanName)
	r.mu.Unlock()

	//nolint:spancheck
	return noop.NewTracerProvider().Tracer("").Start(ctx, spanName, opts...)
}

func (r *recordingTracer) spanNames() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.names)
}

func TestStreamableHTTP(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "a.txt"), []byte("aaa"), 0o600))

	tracer := &recordingTracer{}

	srv := httptest.NewServer(app.NewHTTPHandler(app.NewServer(filesvc.New(rootDir), 1024, tracer)))
	t.Cleanup(srv.Close)

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil).Connect(
		t.Context(),
		&mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp", DisableStandaloneSSE: true},
		nil,
	)
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, session.Close()) })

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "read_file",
		Arguments: map[string]any{"path": "a.txt"},
	})
	require.NoError(t, err)
	require.False(t, res.IsError, "%v", res.Content)
	require.Len(t, res.Content, 1)

	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "content type %T", res.Content[0])
	assert.Equal(t, "aaa", text.Text)

	assert.Contains(t, tracer.spanNames(), "tools/call read_file")
}
