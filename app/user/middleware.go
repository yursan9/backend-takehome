package user

import (
	"app/session"
	"context"
	"net/http"
	"strings"
)

type tokenCtxKey struct{}

func TokenMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token = strings.TrimPrefix(token, "Bearer ")
		id, ok := session.Get(token)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), tokenCtxKey{}, id)
		next(w, r.WithContext(ctx))
	}
}

func IDFromContext(ctx context.Context) int {
	v, ok := ctx.Value(tokenCtxKey{}).(int)
	if !ok {
		return 0
	}
	return v
}
