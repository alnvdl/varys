package feed

import (
	"log/slog"
	"sort"
	"strings"
)

const maxFeedItems = 100

const (
	TypeXML   = "xml"
	TypeHTML  = "html"
	TypeImage = "img"
)

// Feed represents a feed in the application.
type Feed struct {
	Name  string           `json:"name"`
	Type  string           `json:"type"`
	URL   string           `json:"url"`
	Items map[string]*Item `json:"items"`

	Params           any    `json:"params"`
	LastRefreshedAt  int64  `json:"updated_at"`
	LastRefreshError string `json:"error"`
}

// FeedSummary is the external representation of the feed (e.g., for presenting
// to users).
type FeedSummary struct {
	UID         string         `json:"uid"`
	URL         string         `json:"url"`
	Name        string         `json:"name"`
	Items       []*ItemSummary `json:"items,omitempty"`
	LastUpdated int64          `json:"last_updated"`
	LastError   string         `json:"last_error"`
	ItemCount   int            `json:"item_count"`
	ReadCount   int            `json:"read_count"`
}

func (f *Feed) UID() string {
	if f.URL == "" {
		return strings.ToLower(f.Name)
	}
	return UID(f.URL)
}

// Prune removes items from the feed until the number of items is less than or
// equal to n. It removes the oldest items first. If n is less than or equal to
// zero, it prunes the feed to the default number of items.
func (f *Feed) Prune(n int) {
	if n <= 0 {
		n = maxFeedItems
	}
	if len(f.Items) <= n {
		return
	}
	items := f.SortedItems()
	remainingItems := items[:n]
	f.Items = make(map[string]*Item)
	for _, item := range remainingItems {
		f.Items[UID(item.URL)] = &item
	}
}

// Refresh updates the feed with information coming from raw items and the
// given fetch error, and then prunes the feed to the maximum number of items.
func (f *Feed) Refresh(items []RawItem, ts int64, fetchErr error) {
	log := slog.With(slog.String("feedName", f.Name))
	var fetchErrMsg string
	if fetchErr != nil {
		fetchErrMsg = fetchErr.Error()
	}
	log.Info("refreshing feed",
		slog.Int("nItems", len(f.Items)),
		slog.String("fetchErr", fetchErrMsg),
	)

	if fetchErr != nil {
		f.LastRefreshError = fetchErr.Error()
		return
	}
	if len(items) == 0 {
		f.LastRefreshError = "no items found in the last refresh"
		return
	}

	if f.Items == nil {
		f.Items = make(map[string]*Item)
	}

	for i, item := range items {
		if !item.IsValid() {
			log.Info("detected invalid item in feed, skipping", slog.Int("itemPos", i))
			continue
		}
		// If the item was never seen before, add it with the current
		// timestamp.
		if f.Items[item.UID()] == nil {
			f.Items[item.UID()] = &Item{
				FeedUID:   f.UID(),
				Timestamp: ts,
			}
		}
		f.Items[item.UID()].Refresh(item)
	}
	f.LastRefreshedAt = ts

	f.Prune(maxFeedItems)
	log.Info("feed refreshed", slog.Int("nFeedItems", len(f.Items)))
}

// SortedItems returns the items in the feed sorted by timestamp, position and
// then URL in descending order.
func (f *Feed) SortedItems() []Item {
	var sortedItems []Item
	for _, item := range f.Items {
		sortedItems = append(sortedItems, *item)
	}
	sort.Slice(sortedItems, func(i, j int) bool {
		if sortedItems[i].Timestamp == sortedItems[j].Timestamp &&
			sortedItems[i].Position == sortedItems[j].Position {
			return sortedItems[i].URL < sortedItems[j].URL
		}

		if sortedItems[i].Timestamp == sortedItems[j].Timestamp {
			return sortedItems[i].Position < sortedItems[j].Position
		}
		return sortedItems[i].Timestamp > sortedItems[j].Timestamp
	})
	return sortedItems
}

// MarkAllRead marks all feed items as read.
func (f *Feed) MarkAllRead() {
	for _, item := range f.Items {
		item.MarkRead()
	}
}

// Summary returns a summary of the feed. If withItems is true, it includes the
// items in the feed. itemMapper should usually be nil; it's only used in
// special cases when building virtual feeds containing items from other feeds.
func (f *Feed) Summary(withItems bool, itemMapper map[string]*Feed) *FeedSummary {
	items := f.SortedItems()
	var readCount int
	for _, item := range items {
		if item.Read {
			readCount++
		}
	}

	var itemSummaries []*ItemSummary
	if withItems {
		for _, item := range items {
			feed := f
			if itemMapper != nil {
				feed = itemMapper[item.UID()]
			}
			itemSummaries = append(itemSummaries, item.Summary(feed, false))
		}
	}

	return &FeedSummary{
		UID:         f.UID(),
		URL:         f.URL,
		Name:        f.Name,
		Items:       itemSummaries,
		LastUpdated: f.LastRefreshedAt,
		LastError:   f.LastRefreshError,
		ItemCount:   len(items),
		ReadCount:   readCount,
	}
}
