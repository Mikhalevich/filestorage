package middleware

import (
	"net/http"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/router"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

// Resource puts the resource addressed by the matched route into the request context
// and adds its user to the request logger. It must wrap handlers inside the mux,
// path values are empty before the route is matched.
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

			ctx = logger.WithLogger(ctx, logger.FromContext(ctx).WithField("user", res.User))

			next.ServeHTTP(rspWriter, req.WithContext(ctx))
		})
	}
}
