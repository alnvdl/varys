package feed

import (
	"errors"
	"log/slog"
	"sort"
	"strings"
)

const (
	TypeXML   = "xml"
	TypeHTML  = "html"
	TypeImage = "img"
)

// Feed represents a feed in the application.
type Feed struct {
	// Name is the name of the feed as defined by the user.
	Name string `json:"name"`

	// Type is the type of the feed. It can be one of the types defined in this
	// file.
	Type string `json:"type"`

	// URL is the URL from which the feed is fetched. If this is empty, the
	// feed is assumed to be a virtual feed managed by the application.
	URL string `json:"url"`

	// Items is a map of items in the feed. The key is the UID of the item.
	Items map[string]*Item `json:"items"`

	// Params is an object with parameters that are specific to the feed type.
	// It is up to the feed type to define what these parameters are, and they
	// are usually used when parsing or refreshing the feed.
	Params any `json:"params"`

	// LastRefreshedAt is the time when the feed was last fetched.
	LastRefreshedAt int64 `json:"updated_at"`

	// LastRefreshError is the last error that occurred when refreshing the
	// feed.
	LastRefreshError string `json:"error"`
}

// FeedSummary is the external representation of the feed (e.g., for presenting
// to users).
type FeedSummary struct {
	// UID is the unique identifier of the feed.
	UID string `json:"uid"`

	// URL is the URL from which the feed is fetched.
	URL string `json:"url"`

	// Name is the name of the feed as defined by the user.
	Name string `json:"name"`

	// Items is a list of items in the feed. It is usually empty, unless
	// explicitly requested.
	Items []*ItemSummary `json:"items,omitempty"`

	// LastUpdated is the time when the feed was last fetched.
	LastUpdated int64 `json:"last_updated"`

	// LastError is the last error that occurred when refreshing the feed.
	LastError string `json:"last_error"`

	// ItemCount is the number of items in the feed.
	ItemCount int `json:"item_count"`

	// ReadCount is the number of items in the feed that were marked as read.
	ReadCount int `json:"read_count"`
}

func (f *Feed) UID() string {
	if f.URL == "" {
		return strings.ToLower(f.Name)
	}
	return UID(f.URL)
}

type feedParams struct {
	MaxItems int `json:"max_items"`
}

func (p *feedParams) Validate() error {
	if p.MaxItems <= 0 {
		return errors.New("max_items must be positive")
	}
	return nil
}

// Prune removes items from the feed until the number of items is less than or
// equal to n. It removes the last items as determined by Feed.SortedItems. If
// n is less than zero, or if the feeds has less than n items, it does nothing.
// If n is zero and observedRawItems is greater than zero, it takes into
// account observedRawItems, trying to reach a balance between a minimum of
// 100 items and a maximum of 200 items. If the feed's max_items param is set,
// it is always respected as long as it is greater than zero.
func (f *Feed) Prune(n int, observedRawItems int) {
	if n == 0 {
		var p feedParams
		if err := ParseParams(f.Params, &p); err == nil && p.MaxItems > 0 {
			// Respect the max_items param.
			n = p.MaxItems
		} else if observedRawItems > 0 {
			// Try to reach a balance between 100 and 200 items.
			n = observedRawItems * 2
			if n < 100 {
				n = 100
			} else if n > 200 {
				n = 200
			}
		}
	}
	if n < 0 || len(f.Items) <= n {
		return
	}
	items := f.SortedItems()
	remainingItems := items[:n]
	f.Items = make(map[string]*Item)
	for _, item := range remainingItems {
		f.Items[UID(item.URL)] = &item
	}
}

// Refresh updates the feed with information coming from raw items or an error
// coming from a fetcher, and then prunes the feed to the maximum number of
// items.
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

	f.Prune(0, len(items))
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

// MarkAllRead marks all feed items as read if their timestamp is less than or
// equal to the given before timestamp.
func (f *Feed) MarkAllRead(before int64) {
	for _, item := range f.Items {
		if item.Timestamp <= before {
			item.MarkRead()
		}
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
