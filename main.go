package main

import (
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alnvdl/varys/internal/list"
	"github.com/alnvdl/varys/internal/list/mem"
	"github.com/alnvdl/varys/internal/web"
)

const (
	defaultDBPath          = "db.json"
	defaultListenAddress   = ":8080"
	defaultPersistInterval = 1 * time.Minute
)

func dbPath() string {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}
	return dbPath
}

func listenAddress() string {
	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		return defaultListenAddress
	}
	return listenAddress
}

func sessionKey() []byte {
	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey != "" {
		return []byte(sessionKey)
	}

	sk := make([]byte, 32)
	rand.Read(sk)
	return sk
}

func accessToken() string {
	accessToken := os.Getenv("ACCESS_TOKEN")
	if accessToken == "" {
		panic("empty access token")
	}
	return accessToken
}

func persistInterval() time.Duration {
	pi := os.Getenv("PERSIST_INTERVAL")
	if d, err := time.ParseDuration(pi); err == nil {
		return d
	}
	return defaultPersistInterval
}

func feeds() []*list.InputFeed {
	var feeds []*list.InputFeed
	if err := json.Unmarshal([]byte(os.Getenv("FEEDS")), &feeds); err != nil {
		slog.Error("cannot parse feeds", slog.String("err", err.Error()))
	}
	return feeds
}

func main() {
	feedList := mem.NewList(mem.ListParams{
		DBFilePath:      dbPath(),
		PersistInterval: persistInterval(),
	})
	feedList.LoadFeeds(feeds())
	feedList.Refresh()

	h := web.NewHandler(&web.HandlerParams{
		FeedList:    feedList,
		AccessToken: accessToken(),
		SessionKey:  sessionKey(),
	})
	srv := &http.Server{
		Addr:    listenAddress(),
		Handler: h,
	}
	slog.Info("starting server", slog.String("address", srv.Addr))
	srv.ListenAndServe()
}
