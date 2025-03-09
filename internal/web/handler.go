package web

import (
	"embed"
	"encoding/json"
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
	MarkRead(fuid, iuid string, before int64) bool
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
		h.HandleFunc(e.method+" "+e.path, logpanics(log(addCSPPolicyHeader(handler))))
	}

	return h
}

func jsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		slog.Error("cannot encode response", slog.String("err", err.Error()))
		writeErrorResponse(w, http.StatusInternalServerError, "cannot encode response")
		return
	}
}

func (s *handler) login(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	var data struct {
		Token string `json:"token"`
	}
	err := dec.Decode(&data)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "cannot decode request")
		return
	}

	if data.Token == s.p.AccessToken {
		http.SetCookie(w, s.sessionCookie())
		w.WriteHeader(http.StatusOK)
		return
	} else {
		writeErrorResponse(w, http.StatusUnauthorized, "unauthorized")
	}
}

func (s *handler) feedList(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, s.p.FeedList.Summary())
}

func (s *handler) feed(w http.ResponseWriter, r *http.Request) {
	fuid := r.PathValue("fuid")
	feed := s.p.FeedList.FeedSummary(fuid)
	if feed == nil {
		writeErrorResponse(w, http.StatusNotFound, "feed not found")
		return
	}

	jsonResponse(w, feed)
}

func (s *handler) item(w http.ResponseWriter, r *http.Request) {
	fuid := r.PathValue("fuid")
	iuid := r.PathValue("iuid")
	item := s.p.FeedList.FeedItem(fuid, iuid)
	if item == nil {
		writeErrorResponse(w, http.StatusNotFound, "item not found")
		return
	}

	jsonResponse(w, item)
}

func (s *handler) read(w http.ResponseWriter, r *http.Request) {
	fuid := r.PathValue("fuid")
	iuid := r.PathValue("iuid")

	var data struct {
		Before int64 `json:"before"`
	}

	if iuid == "" && r.Body != nil {
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&data)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "cannot decode request")
			return
		}
	}

	done := s.p.FeedList.MarkRead(fuid, iuid, data.Before)
	if !done {
		writeErrorResponse(w, http.StatusNotFound, "item or feed not found")
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
		writeErrorResponse(w, http.StatusInternalServerError, "cannot read version file")
	}
	version := strings.TrimSpace(string(bVersion))

	jsonResponse(w, statusResponse{
		Status:  "ok",
		Version: version,
	})
}
