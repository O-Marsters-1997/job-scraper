package client

import (
	"context"
	"net/http"
)

type Kind string

const (
	Web Kind = "web"
	IOS Kind = "ios"
)

const Header = "X-Client"

type ctxKey struct{}

// Middleware stores the request's Kind in its context; a missing or unknown header is Web.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := Web
		if Kind(r.Header.Get(Header)) == IOS {
			k = IOS
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, k)))
	})
}

func FromContext(ctx context.Context) Kind {
	if k, ok := ctx.Value(ctxKey{}).(Kind); ok {
		return k
	}
	return Web
}
