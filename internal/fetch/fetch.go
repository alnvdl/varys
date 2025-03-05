// Package fetch provides functions for fetching feeds and parsing them into
// raw items to be processed by the feed package.
package fetch

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/alnvdl/varys/internal/feed"
)

// FetchParams represents the parameters needed to fetch and parse a feed.
type FetchParams struct {
	URL      string
	FeedName string
	FeedType string
}

// Fetch fetches and parses the feed identified by the given p parameters.
func Fetch(p FetchParams) ([]feed.RawItem, error) {
	log := slog.With(slog.String("feedName", p.FeedName))
	log.Info("fetching feed")

	res, err := http.Get(p.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot make request: %v", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body: %v", err)
	}

	var items []feed.RawItem
	switch p.FeedType {
	case feed.TypeXML:
		log.Info("parsing XML feed")
		items, err = parseXML(data, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot parse XML feed: %v", err)
		}
	default:
		return nil, fmt.Errorf("unsupported feed type: %s", p.FeedType)
	}

	log.Info("feed fetched", slog.Int("nFeedItems", len(items)))
	return items, nil
}
