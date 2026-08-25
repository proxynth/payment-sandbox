package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// BearerToken protects administrative routes with a shared secret.
func BearerToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			provided := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
			if token == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				writer.Header().Set("WWW-Authenticate", `Bearer realm="payment-sandbox-admin"`)
				WriteError(writer, http.StatusUnauthorized, "authentication_required", "a valid bearer token is required")
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}
