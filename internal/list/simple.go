// Package list provides implementations of feed lists that use their own
// fetching and persistence mechanisms.
package list

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"maps"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/fetch"
	"github.com/alnvdl/varys/internal/timeutil"
)

// Simple is a feed list that is kept in memory and backed by a serialized JSON
// file. It uses the "fetch" package for fetching feeds and supports a virtual
// "all" feed.
type Simple struct {
	feeds           map[string]*feed.Feed
	muFeeds         sync.Mutex
	dbFilePath      string
	refreshInterval time.Duration
	fetcher         func(p fetch.FetchParams) ([]feed.RawItem, error)
}

type serializedSimpleStore struct {
	Feeds map[string]*feed.Feed `json:"feeds"`
}

// SimpleParams is the configuration for creating a new Simple feed list.
type SimpleParams struct {
	// DBFilePath is the path to the file where the feed list is serialized to
	// and deserialized from. If empty, the feed list will not be persisted.
	DBFilePath string

	// RefreshInterval is the interval at which feeds are refreshed. If 0,
	// auto-refresh will be disabled.
	RefreshInterval time.Duration

	// Fetcher is the function used to fetch feeds. If nil, the default fetcher
	// will be used.
	Fetcher func(p fetch.FetchParams) ([]feed.RawItem, error)
}

// NewSimple creates a new Simple feed list based on the given p parameters.
func NewSimple(p SimpleParams) *Simple {
	if p.Fetcher == nil {
		p.Fetcher = fetch.Fetch
	}
	return &Simple{
		feeds:           make(map[string]*feed.Feed),
		dbFilePath:      p.DBFilePath,
		refreshInterval: p.RefreshInterval,
		fetcher:         p.Fetcher,
	}
}

// Summary returns a summary of all feeds in the list.
func (l *Simple) Summary() []*feed.FeedSummary {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	i := 1
	summaries := make([]*feed.FeedSummary, len(l.feeds)+1)
	summaries[0] = simpleStoreAllFeed(maps.Values(l.feeds), false)
	for _, feed := range l.feeds {
		summaries[i] = feed.Summary(false, nil)
		i++
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries
}

// FeedSummary returns a summary of the feed with the given UID.
func (l *Simple) FeedSummary(uid string) *feed.FeedSummary {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	if uid == "all" {
		return simpleStoreAllFeed(maps.Values(l.feeds), true)
	}

	if feed, ok := l.feeds[uid]; ok {
		return feed.Summary(true, nil)
	}

	return nil
}

// FeedItem returns a summary of the item with the given UID.
func (l *Simple) FeedItem(fuid, iuid string) *feed.ItemSummary {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	feed := l.feeds[fuid]
	if feed != nil {
		item := feed.Items[iuid]
		if item != nil {
			return item.Summary(feed, true)
		}
	}

	return nil
}

// MarkRead marks the feed or item with the given UID as read. It returns true
// if the feed or item was found and marked as read, false otherwise.
func (l *Simple) MarkRead(fuid, iuid string) bool {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	// Marking all feeds as read.
	if fuid == "all" {
		for _, feed := range l.feeds {
			feed.MarkAllRead()
		}
		return true
	}

	if f, ok := l.feeds[fuid]; ok {
		if iuid != "" {
			// Marking an item as read.
			if i, ok := f.Items[iuid]; ok {
				i.MarkRead()
				return true
			}
		} else {
			// Marking a feed as read.
			f.MarkAllRead()
			return true
		}
	}

	return false
}

// Refresh fetches all feeds in the list and then refreshes them.
func (l *Simple) Refresh() {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	wg := sync.WaitGroup{}

	for _, feed := range l.feeds {
		wg.Add(1)
		go func() {
			items, fetchErr := l.fetcher(fetch.FetchParams{
				URL:        feed.URL,
				FeedName:   feed.Name,
				FeedType:   feed.Type,
				FeedParams: feed.Params,
			})
			feed.Refresh(items, fetchErr)
			wg.Done()
		}()
	}

	wg.Wait()
}

func (l *Simple) Save(w io.Writer) error {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	enc := json.NewEncoder(w)
	if os.Getenv("DEBUG") != "" {
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
	}
	err := enc.Encode(serializedSimpleStore{Feeds: l.feeds})
	if err != nil {
		return fmt.Errorf("cannot serialize feed list: %v", err)
	}
	return nil
}

func (l *Simple) Load(r io.Reader) error {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	dec := json.NewDecoder(r)
	data := serializedSimpleStore{}
	err := dec.Decode(&data)
	if err != nil {
		return fmt.Errorf("cannot deserialize feed list: %v", err)
	}
	l.feeds = data.Feeds

	return nil
}

// LoadFeeds ensures that the feeds in the list match the given input feeds.
// It keeps existing feeds that are in the input, adds new feeds that are
// missing and discards feeds that are not in the input. So leaving inputFeeds
// empty or nil will remove all feeds.
func (l *Simple) LoadFeeds(inputFeeds []*InputFeed) {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	slog.Info("loading feeds")
	newFeeds := make(map[string]*feed.Feed)
	for _, inputFeed := range inputFeeds {
		if f, ok := l.feeds[feed.UID(inputFeed.URL)]; ok {
			// Feed is already in the list and is part of the input, keep it.
			newFeeds[f.UID()] = f
			continue
		} else {
			// Feed does not yet exist, add it to the list.
			newFeed := &feed.Feed{
				Name:   inputFeed.Name,
				URL:    inputFeed.URL,
				Type:   inputFeed.Type,
				Params: inputFeed.Params,
			}
			newFeeds[newFeed.UID()] = newFeed
		}
		// Feeds that were in the list but are not part of the input are
		// discarded.
	}

	l.feeds = newFeeds
}

// simpleStoreAllFeed returns the feed summary for the virtual feed containing
// all items from the given feeds. If withItems is true, it includes the items
// in the feed.
func simpleStoreAllFeed(feeds iter.Seq[*feed.Feed], withItems bool) *feed.FeedSummary {
	allFeed := &feed.Feed{
		Name:            "All",
		LastRefreshedAt: timeutil.Now(),
		Items:           make(map[string]*feed.Item),
	}
	itemMapper := make(map[string]*feed.Feed)
	for feed := range feeds {
		for _, item := range feed.Items {
			allFeed.Items[item.UID()] = item
			itemMapper[item.UID()] = feed
		}
	}
	allFeed.Prune(0)
	return allFeed.Summary(withItems, itemMapper)
}
