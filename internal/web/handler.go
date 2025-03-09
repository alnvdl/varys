package web

import (
	"embed"
	"encoding/json"
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

	endpoints := []struct {
		method  string
		path    string
		handler http.HandlerFunc
		authn   bool
	}{{method: "POST",
		path:    "/login",
		handler: h.login,
		authn:   false,
	}, {
		method:  "GET",
		path:    "/api/feeds",
		handler: h.feedList,
		authn:   true,
	}, {
		method:  "GET",
		path:    "/api/feeds/{fuid}",
		handler: h.feed,
		authn:   true,
	}, {
		method:  "POST",
		path:    "/api/feeds/{fuid}/read",
		handler: h.read,
		authn:   true,
	}, {
		method:  "GET",
		path:    "/api/feeds/{fuid}/items/{iuid}",
		handler: h.item,
		authn:   true,
	}, {
		method:  "POST",
		path:    "/api/feeds/{fuid}/items/{iuid}/read",
		handler: h.read,
		authn:   true,
	}, {
		method:  "GET",
		path:    "/status",
		handler: h.status,
		authn:   false,
	}, {
		method:  "GET",
		path:    "/static/",
		handler: http.FileServer(http.FS(staticFiles)).ServeHTTP,
		authn:   false,
	}, {
		method: "GET",
		path:   "/",
		handler: func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, staticFiles, "static/index.html")
		},
		authn: false,
	}}

	for _, e := range endpoints {
		handler := e.handler
		if e.authn {
			handler = h.requireAuthentication(handler)
		}
		h.HandleFunc(e.method+" "+e.path, h.recover(h.log(handler)))
	}

	return h
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

type statusResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (s *handler) status(w http.ResponseWriter, r *http.Request) {
	bVersion, err := staticFiles.ReadFile("static/version")
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "cannot read version file")
	}
	version := strings.TrimSpace(string(bVersion))

	err = json.NewEncoder(w).Encode(statusResponse{
		Status:  "ok",
		Version: version,
	})
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "cannot encode status response")
	}
}
