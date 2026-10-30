package middleware

import (
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/router"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

func Resource() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rspWriter http.ResponseWriter, req *http.Request) {
			var (
				res = resource.Resource{
					User: router.User(req),
					Path: router.ResourcePath(req),
				}

				ctx = resource.WithContext(req.Context(), res)
			)

			next.ServeHTTP(rspWriter, req.WithContext(ctx))
		})
	}
}
