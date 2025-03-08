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
	URL        string
	FeedName   string
	FeedType   string
	FeedParams any
}

// parser is a function that parses feed data, optionally using the given
// params, and returns raw items.
type parser func(data []byte, params any) ([]feed.RawItem, error)

var parsers = map[string]parser{
	feed.TypeXML:  parseXML,
	feed.TypeHTML: parseHTML,
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
	if parser, ok := parsers[p.FeedType]; ok {
		log.Info("parsing feed", slog.String("feedType", p.FeedType))
		items, err = parser(data, p.FeedParams)
		if err != nil {
			return nil, fmt.Errorf("cannot parse feed: %v", err)
		}
	} else {
		return nil, fmt.Errorf("unsupported feed type: %s", p.FeedType)
	}

	log.Info("feed fetched and parsed", slog.Int("nFeedItems", len(items)))
	return items, nil
}
