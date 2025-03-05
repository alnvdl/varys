package main

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/alnvdl/varys/internal/list"
	"github.com/alnvdl/varys/internal/web"
)

const (
	defaultDBPath        = "db.json"
	defaultListenAddress = ":8080"
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

func main() {
	inputFile, err := os.OpenFile(dbPath(), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open DB file: %v\n", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	feedList := list.NewSimple(list.SimpleParams{})
	feedList.Load(inputFile)
	feedList.Refresh()

	outputFile, err := os.OpenFile(dbPath(), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open DB file: %v\n", err)
		os.Exit(1)
	}

	feedList.Save(outputFile)

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
