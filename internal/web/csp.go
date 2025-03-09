package web

import "net/http"

var defaultCSPPolicy = "default-src 'self'; img-src * data:"

// addCSPPolicyHeader adds a Content-Security-Policy header to the response.
func addCSPPolicyHeader(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", defaultCSPPolicy)
		handler(w, r)
	}
}
