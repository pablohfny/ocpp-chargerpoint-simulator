package middleware

import (
	"crypto/subtle"
	"net/http"
)

// BasicAuth protects a handler with HTTP basic auth. When user is empty the
// middleware is a no-op, which keeps local development friction free.
func BasicAuth(realm, user, pass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if user == "" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			givenUser, givenPass, ok := r.BasicAuth()
			if !ok || !credentialsMatch(givenUser, givenPass, user, pass) {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`", charset="UTF-8"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// credentialsMatch compares credentials in constant time.
func credentialsMatch(givenUser, givenPass, wantUser, wantPass string) bool {
	userMatch := subtle.ConstantTimeCompare([]byte(givenUser), []byte(wantUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(givenPass), []byte(wantPass)) == 1
	return userMatch && passMatch
}
