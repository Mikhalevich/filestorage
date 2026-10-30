package router

import (
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
)

func CreateMux(hdl *handler.Handler) http.Handler {
	mux := http.NewServeMux()

	registerHandler(mux, "GET /files/{user}/{resourcePath...}", hdl.FileList)
	registerHandler(mux, "POST /files/{user}/{resourcePath...}", hdl.FileUpload)
	registerHandler(mux, "DELETE /files/{user}/{resourcePath...}", hdl.FileDelete)

	registerHandler(mux, "POST /dirs/{user}/{resourcePath...}", hdl.DirectoryCreate)

	registerHandler(mux, "GET /download/{user}/{resourcePath...}", hdl.FileDownload)

	return mux
}

func ResourcePath(req *http.Request) string {
	return req.PathValue("resourcePath")
}

func User(req *http.Request) string {
	return req.PathValue("user")
}
