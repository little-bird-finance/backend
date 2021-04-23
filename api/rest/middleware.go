package rest

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ctxKeyTraceID string

const TraceIDKey ctxKeyTraceID = "trace-id"

var TraceIDHeader = "X-Trace-Id"

func TraceID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			traceID = uuid.NewString()
		}
		ctx = context.WithValue(ctx, TraceIDKey, traceID)

		w.Header().Set("trace-id", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

func LogHandler(logger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l := logger.
				With().
				Str("trace-id", GetTraceID(ctx)).
				Logger()

			l.Info().
				Str("method", r.Method).
				Str("url", r.URL.String()).
				Msg("request")
			ctx = l.WithContext(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
			l.Info().Msg("response")
		}
		return http.HandlerFunc(fn)
	}
}
