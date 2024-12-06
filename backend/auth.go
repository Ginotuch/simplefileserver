package backend

import (
	"net/http"
	"strings"
)

func BasicAuthMiddleware(username, password string, next http.Handler) http.Handler {
	if username == "" || password == "" {
		// No auth required if not configured
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != username || p != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="simplefileserver"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Whether a path should be protected by auth
func isProtectedPath(p string) bool {
	// Previously certain endpoints were protected. We can preserve that logic:
	// The original code protected /download/, /gettemplink/, /walk/, /favicon.ico
	// Let's do the same:
	protectedPrefixes := []string{"/download/", "/gettemplink/", "/walk/", "/favicon.ico"}
	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}
