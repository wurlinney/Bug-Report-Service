package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

const headerRequestID = "X-Request-Id"

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := r.Header.Get(headerRequestID)
			if rid == "" {
				rid = newRequestID()
			}

			w.Header().Set(headerRequestID, rid)
			ctx := context.WithValue(r.Context(), RequestIDKey, rid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func GetRequestID(ctx context.Context) string {
	v, _ := ctx.Value(RequestIDKey).(string)
	return v
}
