package router

import (
	"net/http"
	"slices"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
)

// CreateMux registers handlers wrapped with middlewares, the first middleware is the outermost one.
// common middlewares wrap every route. resourceMiddlewares wrap only routes addressing a user's resource
// (inside common ones). Middlewares run after the route is matched, so they can read path values
// with User and ResourcePath.
func CreateMux(
	hdl *handler.Handler,
	common []Middleware,
	resourceMiddlewares []Middleware,
) http.Handler {
	var (
		mux          = http.NewServeMux()
		resourceMWs  = slices.Concat(common, resourceMiddlewares)
		registerAuth = func(pattern string, h HandlerFunc) {
			registerHandler(mux, pattern, h, common)
		}
		register = func(pattern string, h HandlerFunc) {
			registerHandler(mux, pattern, h, resourceMWs)
		}
	)

	registerAuth("POST /auth/register", hdl.AuthRegister)
	registerAuth("POST /auth/login", hdl.AuthLogin)

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
