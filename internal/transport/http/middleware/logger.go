package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func Logger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()
			defer func() {
				requestID, _ := GetRequestID(r.Context())
				logger.Info().
					Str("request_id", requestID).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Int("status", ww.Status()).
					Dur("latency", time.Since(start)).
					Int("bytes", ww.BytesWritten()).
					Msg("request handled")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
