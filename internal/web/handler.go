package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alnvdl/varys/internal/feed"
)

//go:embed static/*
var staticFiles embed.FS

// FeedLister is the interface that the API server uses to interact with a list
// of feeds.
type FeedLister interface {
	Summary() []*feed.FeedSummary
	FeedSummary(uid string) *feed.FeedSummary
	FeedItem(fuid, iuid string) *feed.ItemSummary
	MarkRead(fuid, iuid string) bool
	Refresh()
}

// HandlerParams contains the parameters for creating a new API server.
type HandlerParams struct {
	FeedList    FeedLister
	AccessToken string
	SessionKey  []byte
}

type handler struct {
	*http.ServeMux
	p *HandlerParams
}

// NewHandler creates a new HTTP handler for the entire SPA of the feed reader,
// including static files and the API.
func NewHandler(p *HandlerParams) *handler {
	h := &handler{
		ServeMux: http.NewServeMux(),
		p:        p,
	}
	h.HandleFunc("POST /login", h.login)
	h.HandleFunc("GET /api/feeds", h.requireAuthentication(h.feedList))
	h.HandleFunc("GET /api/feeds/{fuid}", h.requireAuthentication(h.feed))
	h.HandleFunc("POST /api/feeds/{fuid}/read", h.requireAuthentication(h.read))
	h.HandleFunc("GET /api/feeds/{fuid}/items/{iuid}", h.requireAuthentication(h.item))
	h.HandleFunc("POST /api/feeds/{fuid}/items/{iuid}/read", h.requireAuthentication(h.read))
	h.Handle("GET /static/", http.FileServer(http.FS(staticFiles)))
	h.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticFiles, "/static/index.html")
	})
	return h
}

func (s *handler) sessionCookie() *http.Cookie {
	bSessionID := make([]byte, 32)
	rand.Read(bSessionID)
	sessionID := base64.RawURLEncoding.EncodeToString(bSessionID)

	h := hmac.New(sha256.New, s.p.SessionKey)
	if _, err := h.Write([]byte(sessionID)); err != nil {
		return nil
	}
	return &http.Cookie{
		Name:     "session",
		Value:    sessionID + ":" + base64.RawURLEncoding.EncodeToString(h.Sum(nil)),
		HttpOnly: true,
		Secure:   true,
	}
}

func (s *handler) isAuthenticated(r *http.Request) bool {
	if r == nil {
		return false
	}
	cookie, err := r.Cookie("session")
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

type errorResponse struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

func (s *handler) writeErrorResponse(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(errorResponse{
		Code:    fmt.Sprintf("%d", code),
		Name:    http.StatusText(code),
		Message: message,
	})
	if err != nil {
		slog.Error("cannot send error response", slog.String("err", err.Error()))
	}
}

func (s *handler) login(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var data struct {
		Token string `json:"token"`
	}
	err := dec.Decode(&data)
	if err != nil {
		s.writeErrorResponse(w, http.StatusUnauthorized, "cannot decode request")
		return
	}

	if data.Token == s.p.AccessToken {
		http.SetCookie(w, s.sessionCookie())
		w.WriteHeader(http.StatusOK)
		return
	} else {
		s.writeErrorResponse(w, http.StatusUnauthorized, "unauthorized")
	}
}

func (s *handler) requireAuthentication(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthenticated(r) {
			s.writeErrorResponse(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		handler(w, r)
	}
}

func (s *handler) feedList(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(s.p.FeedList.Summary())
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "cannot encode response")
		return
	}
}

func (s *handler) feed(w http.ResponseWriter, r *http.Request) {
	fuid := r.PathValue("fuid")
	feed := s.p.FeedList.FeedSummary(fuid)
	if feed == nil {
		s.writeErrorResponse(w, http.StatusNotFound, "feed not found")
		return
	}

	err := json.NewEncoder(w).Encode(feed)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "cannot encode response")
		return
	}
}

func (s *handler) item(w http.ResponseWriter, r *http.Request) {
	fuid := r.PathValue("fuid")
	iuid := r.PathValue("iuid")
	item := s.p.FeedList.FeedItem(fuid, iuid)
	if item == nil {
		s.writeErrorResponse(w, http.StatusNotFound, "item not found")
		return
	}

	err := json.NewEncoder(w).Encode(item)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "cannot encode response")
		return
	}
}

func (s *handler) read(w http.ResponseWriter, r *http.Request) {
	fuid := r.PathValue("fuid")
	iuid := r.PathValue("iuid")
	done := s.p.FeedList.MarkRead(fuid, iuid)
	if !done {
		s.writeErrorResponse(w, http.StatusNotFound, "item or feed not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}
