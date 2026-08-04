// Package mem provides an implementation of an in-memory feed list with its
// own auto-refresh and auto-save mechanisms.
package mem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"maps"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/alnvdl/autosave"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/fetch"
	"github.com/alnvdl/varys/internal/list"
	"github.com/alnvdl/varys/internal/timeutil"
)

// List is a feed list that is kept in memory and optionally backed by a
// serialized JSON file. It uses the "fetch" package for fetching feeds and
// supports a virtual "all" feed.
type List struct {
	feeds   map[string]*feed.Feed
	muFeeds sync.Mutex

	refreshInterval time.Duration
	refreshCallback func()
	fetcher         func(p fetch.FetchParams) ([]feed.RawItem, int64, error)
	wg              sync.WaitGroup
	close           chan bool

	autoSaver *autosave.AutoSaver
}

type serializedList struct {
	Feeds map[string]*feed.Feed `json:"feeds"`
}

// ListParams is the configuration for creating a new MemList.
type ListParams struct {
	// InitialFeeds provides a way to initialize the feed list with some feeds.
	// See LoadFeeds for more information on how this is used.
	InitialFeeds []*list.InputFeed

	// RefreshInterval is the interval at which feeds are refreshed. If 0,
	// auto-refresh will be disabled.
	RefreshInterval time.Duration

	// RefreshCallback is an optional function to be called after each
	// auto-refresh operation.
	RefreshCallback func()

	// Fetcher is the function used to fetch feeds. If nil, a default fetcher
	// will be used.
	Fetcher func(p fetch.FetchParams) ([]feed.RawItem, int64, error)

	// AutoSaveParams is the configuration for auto-save. If FilePath is empty,
	// auto-save will be disabled and the list will be entirely in-memory only.
	// The LoaderSave field will be set to the created List, so any value set
	// by the caller will be ignored.
	AutoSaveParams autosave.Params
}

// NewList creates a new in-memory feed list based on the given p parameters.
// It will initialize the auto-save mechanism if configured to do so, then load
// the initial feeds and start the auto-refresh mechanism. Note that it will
// not save the feed list to the file until the first auto-save interval is
// reached.
func NewList(p ListParams) (*List, error) {
	if p.Fetcher == nil {
		p.Fetcher = fetch.Fetch
	}
	l := &List{
		feeds:           make(map[string]*feed.Feed),
		refreshInterval: p.RefreshInterval,
		refreshCallback: p.RefreshCallback,
		fetcher:         p.Fetcher,
		close:           make(chan bool),
	}

	if p.AutoSaveParams.FilePath != "" {
		p.AutoSaveParams.LoaderSaver = l

		var err error
		l.autoSaver, err = autosave.New(p.AutoSaveParams)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize auto-saver: %v", err)
		}
	}

	l.LoadFeeds(p.InitialFeeds)
	l.initRefresh()
	return l, nil
}

// delayAutoSave calls Delay on the autoSaver if it is not nil.
func (l *List) delayAutoSave() {
	if l.autoSaver != nil {
		l.autoSaver.Delay()
	}
}

// Summary returns a summary of all feeds in the list.
func (l *List) Summary() []*feed.FeedSummary {
	defer l.delayAutoSave()
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	i := 1
	summaries := make([]*feed.FeedSummary, len(l.feeds)+1)
	summaries[0] = allFeed(maps.Values(l.feeds), false)
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
func (l *List) FeedSummary(uid string) *feed.FeedSummary {
	defer l.delayAutoSave()
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	if uid == "all" {
		return allFeed(maps.Values(l.feeds), true)
	}

	if feed, ok := l.feeds[uid]; ok {
		return feed.Summary(true, nil)
	}

	return nil
}

// FeedItem returns a summary of the item with the given UID.
func (l *List) FeedItem(fuid, iuid string) *feed.ItemSummary {
	defer l.delayAutoSave()
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

// MarkRead marks the feed or item with the given UID as read. If iuid is
// empty, only items whose timestamp is less than or equal to before are marked
// read. If fuid is "all", all feeds are marked as read, also respecting the
// before timestamp. If the feed or item is found, true is returned, otherwise
// false.
func (l *List) MarkRead(fuid, iuid string, before int64) bool {
	defer l.delayAutoSave()
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	// Marking all feeds as read.
	if fuid == "all" {
		for _, feed := range l.feeds {
			feed.MarkAllRead(before)
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
			f.MarkAllRead(before)
			return true
		}
	}

	return false
}

// LoadFeeds ensures that the feeds in the list match the given input feeds.
// It keeps existing feeds that are in the input, adds new feeds that are
// missing and discards feeds that are not in the input. So leaving inputFeeds
// empty or nil will remove all feeds.
func (l *List) LoadFeeds(inputFeeds []*list.InputFeed) {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	slog.Info("loading feeds",
		slog.Int("currentFeedCount", len(l.feeds)),
		slog.Int("inputFeedCount", len(inputFeeds)),
	)

	var kept, added, discarded int
	newFeeds := make(map[string]*feed.Feed)
	for _, inputFeed := range inputFeeds {
		if f, ok := l.feeds[feed.UID(inputFeed.URL)]; ok {
			// Feed is already in the list and is part of the input, keep it,
			// updating some fields.
			f.Name = inputFeed.Name
			f.Type = inputFeed.Type
			f.Params = inputFeed.Params
			newFeeds[f.UID()] = f
			kept++
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
			added++
		}
		// Feeds that were in the list but are not part of the input are
		// discarded.
	}
	for feed := range l.feeds {
		if _, ok := newFeeds[feed]; !ok {
			discarded++
		}
	}

	slog.Info("finished loading feeds",
		slog.Int("kept", kept),
		slog.Int("added", added),
		slog.Int("discarded", discarded),
		slog.Int("feedCount", len(newFeeds)),
	)
	l.feeds = newFeeds
}

// Load deserializes the feed list from the given reader.
func (l *List) Load(r io.Reader) error {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	dec := json.NewDecoder(r)
	data := serializedList{}
	err := dec.Decode(&data)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		// Ignoring a corrupted file is intentional: we prefer to lose all
		// items and repopulate on refresh than prevent the application from
		// starting.
		slog.Error("error loading data caused by EOF, ignoring",
			slog.String("err", err.Error()))
		return nil
	} else if err != nil {
		return fmt.Errorf("cannot deserialize feed list: %w", err)
	}
	l.feeds = data.Feeds

	return nil
}

// Save serializes the feed list to the given writer.
func (l *List) Save(w io.Writer) error {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	enc := json.NewEncoder(w)
	if os.Getenv("DEBUG") != "" {
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
	}
	err := enc.Encode(serializedList{Feeds: l.feeds})
	if err != nil {
		return fmt.Errorf("cannot serialize feed list: %w", err)
	}
	return nil
}

// Close stops the auto-refresh and auto-save mechanisms and waits for them to
// finish.
func (l *List) Close() {
	close(l.close)
	l.wg.Wait()
	if l.autoSaver != nil {
		l.autoSaver.Close()
	}
}

// allFeed returns the feed summary for the virtual feed containing all items
// from the given feeds. If withItems is true, it includes the items in the
// feed.
func allFeed(feeds iter.Seq[*feed.Feed], withItems bool) *feed.FeedSummary {
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
	allFeed.Prune(500, 0)
	return allFeed.Summary(withItems, itemMapper)
}
