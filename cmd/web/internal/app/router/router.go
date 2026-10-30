package router

import (
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
)

// CreateMux registers handlers wrapped with middlewares, the first middleware is the outermost one.
// Middlewares run after the route is matched, so they can read path values with User and ResourcePath.
func CreateMux(hdl *handler.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	var (
		mux      = http.NewServeMux()
		register = func(pattern string, h HandlerFunc) {
			registerHandler(mux, pattern, h, middlewares)
		}
	)

	register("GET /files/{user}/{resourcePath...}", hdl.FileList)
	register("POST /files/{user}/{resourcePath...}", hdl.FileUpload)
	register("DELETE /files/{user}/{resourcePath...}", hdl.FileDelete)

	register("POST /dirs/{user}/{resourcePath...}", hdl.DirectoryCreate)

	register("GET /download/{user}/{resourcePath...}", hdl.FileDownload)

	return mux
}

func ResourcePath(req *http.Request) string {
	return req.PathValue("resourcePath")
}

func User(req *http.Request) string {
	return req.PathValue("user")
}
