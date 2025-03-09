package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

const sessionCookieName = "session"

// sessionCookie creates a new session cookie with a random session ID and a
// signature.
func (s *handler) sessionCookie() *http.Cookie {
	bSessionID := make([]byte, 32)
	rand.Read(bSessionID)
	sessionID := base64.RawURLEncoding.EncodeToString(bSessionID)

	h := hmac.New(sha256.New, s.p.SessionKey)
	if _, err := h.Write([]byte(sessionID)); err != nil {
		return nil
	}
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID + ":" + base64.RawURLEncoding.EncodeToString(h.Sum(nil)),
		HttpOnly: true,
		Secure:   true,
	}
}

// isAuthenticated checks if the request is authenticated by checking the
// session cookie.
func (s *handler) isAuthenticated(r *http.Request) bool {
	if r == nil {
		return false
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}

	parts := strings.Split(cookie.Value, ":")
	if len(parts) != 2 {
		return false
	}
	sessionID, cookieSigB64 := parts[0], parts[1]

	cookieSig, err := base64.RawURLEncoding.DecodeString(cookieSigB64)
	if err != nil {
		return false
	}

	expectedSig := hmac.New(sha256.New, s.p.SessionKey)
	expectedSig.Write([]byte(sessionID))

	return hmac.Equal(expectedSig.Sum(nil), cookieSig)
}

// requireAuthentication wraps an HTTP handler and requires that the request is
// authenticated.
func (s *handler) requireAuthentication(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthenticated(r) {
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		handler(w, r)
	}
}
