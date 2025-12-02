package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// Recoverer returns a middleware that recovers from panics.
func Recoverer(log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					log.Error("panic recovered",
						"error", rvr,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
					)

					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
