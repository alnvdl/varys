// Package mem provides implementations of an in-memory feed list with its own
// auto-refresh and auto-persistence mechanisms.
package mem

import (
	"iter"
	"log/slog"
	"maps"
	"sort"
	"sync"
	"time"

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

	dbFilePath      string
	persistInterval time.Duration
	persistBackoff  chan bool
	persistCallback func(err error)

	wg    sync.WaitGroup
	close chan bool
}

type serializedList struct {
	Feeds map[string]*feed.Feed `json:"feeds"`
}

// ListParams is the configuration for creating a new MemList.
type ListParams struct {
	// InitialFeeds provides a way to initialize the feed list with some feeds.
	// See LoadFeeds for more information on how this is used.
	InitialFeeds []*list.InputFeed

	// DBFilePath is the path to the file in FS where the feed list is
	// serialized to and deserialized from. If empty, the feed list will not be
	// persisted and will be kept only in memory.
	DBFilePath string

	// PersistInterval is the interval at which the feed list is persisted to
	// the file. If 0, auto-persistence will be disabled.
	PersistInterval time.Duration

	// PersistCallback is an optional function to be called after each
	// persistence operation is attempted (successfully or not depending as
	// given by err).
	PersistCallback func(err error)

	// RefreshInterval is the interval at which feeds are refreshed. If 0,
	// auto-refresh will be disabled.
	RefreshInterval time.Duration

	// RefreshCallback is an optional function to be called after each
	// auto-refresh operation.
	RefreshCallback func()

	// Fetcher is the function used to fetch feeds. If nil, a default fetcher
	// will be used.
	Fetcher func(p fetch.FetchParams) ([]feed.RawItem, int64, error)
}

// NewList creates a new in-memory feed list based on the given p parameters.
// It will also initialize the persistence mechanism (if dbFilePath is given),
// load the initial feeds and start the auto-refresh mechanism. Note that it
// will not persist the feed list to the file until the first auto-persistence
// interval is reached.
func NewList(p ListParams) *List {
	if p.Fetcher == nil {
		p.Fetcher = fetch.Fetch
	}
	l := &List{
		feeds:           make(map[string]*feed.Feed),
		dbFilePath:      p.DBFilePath,
		refreshInterval: p.RefreshInterval,
		refreshCallback: p.RefreshCallback,
		fetcher:         p.Fetcher,
		persistInterval: p.PersistInterval,
		persistBackoff:  make(chan bool, 5),
		persistCallback: p.PersistCallback,
		close:           make(chan bool),
	}
	l.initPersist()
	l.LoadFeeds(p.InitialFeeds)
	l.initRefresh()
	return l
}

// Summary returns a summary of all feeds in the list.
func (l *List) Summary() []*feed.FeedSummary {
	defer l.delayPersist()
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
	defer l.delayPersist()
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
	defer l.delayPersist()
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
	defer l.delayPersist()
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

// allFeed returns the feed summary for the virtual feed containing all
// items from the given feeds. If withItems is true, it includes the items in
// the feed.
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
	allFeed.Prune(2048)
	return allFeed.Summary(withItems, itemMapper)
}
