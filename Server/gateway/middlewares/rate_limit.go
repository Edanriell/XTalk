package middlewares

import "net/http"

// TODO
// Do we need this ?
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO
		// Implement rate limiting logic here
		next.ServeHTTP(w, r)
	})
}
