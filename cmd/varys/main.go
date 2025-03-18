// Package main provides the entrypoint for the varys feed reader server.
package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alnvdl/varys/internal/list"
	"github.com/alnvdl/varys/internal/list/mem"
	"github.com/alnvdl/varys/internal/web"
)

const (
	defaultDBPath              = "db.json"
	defaultPort                = "8080"
	defaultPersistInterval     = 1 * time.Minute
	defaultRefreshInterval     = 5 * time.Minute
	defaultHealthCheckInterval = 3 * time.Minute
)

func dbPath() string {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}
	return dbPath
}

func port() string {
	port := os.Getenv("PORT")
	if port == "" {
		return defaultPort
	}
	return port
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

func refreshInterval() time.Duration {
	ri := os.Getenv("REFRESH_INTERVAL")
	if d, err := time.ParseDuration(ri); err == nil {
		return d
	}
	return defaultRefreshInterval
}

func healthCheckInterval() time.Duration {
	hci := os.Getenv("HEALTH_CHECK_INTERVAL")
	if d, err := time.ParseDuration(hci); err == nil {
		return d
	}
	return defaultHealthCheckInterval
}

func feeds() []*list.InputFeed {
	var feeds []*list.InputFeed
	if err := json.Unmarshal([]byte(os.Getenv("FEEDS")), &feeds); err != nil {
		slog.Error("cannot parse feeds", slog.String("err", err.Error()))
	}
	return feeds
}

// serverHealthCheck periodically checks the server health by making a request
// to the /status endpoint.
func serverHealthCheck(interval time.Duration, port string, close chan bool) {
	for {
		select {
		case <-time.After(interval):
			res, err := http.Get(fmt.Sprintf("http://localhost:%s/status", port))
			if err != nil {
				slog.Error("error making health check request",
					slog.String("err", err.Error()))
				continue
			}
			if res.StatusCode == http.StatusOK {
				slog.Info("server is healthy")
			} else {
				slog.Error("server is not healthy",
					slog.Int("status_code", res.StatusCode))
			}
		case <-close:
			slog.Info("stopping keep-alive mechanism")
			return
		}
	}
}

func main() {
	feedList := mem.NewList(mem.ListParams{
		InitialFeeds:    feeds(),
		DBFilePath:      dbPath(),
		PersistInterval: persistInterval(),
		RefreshInterval: refreshInterval(),
	})

	handler := web.NewHandler(&web.HandlerParams{
		FeedList:    feedList,
		AccessToken: accessToken(),
		SessionKey:  sessionKey(),
	})

	server := &http.Server{
		Addr:    ":" + port(),
		Handler: handler,
	}

	healthCheck := make(chan bool)
	go serverHealthCheck(healthCheckInterval(), port(), healthCheck)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals

		close(healthCheck)
		feedList.Close()
		slog.Info("shutting down server")
		server.Shutdown(context.Background())
	}()

	slog.Info("starting server", slog.String("address", server.Addr))
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			slog.Info("server shut down")
		} else {
			slog.Error("unexpected error on listen and serve", slog.String("error", err.Error()))
		}
	}
}
