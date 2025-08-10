package feed_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/timeutil"
)

func checkFeedItems(t *testing.T, feed feed.Feed, expectedItems []feed.Item) {
	if len(feed.Items) != len(expectedItems) {
		t.Errorf("expected %d items, got %d", len(expectedItems), len(feed.Items))
		return
	}
	sortedItems := feed.SortedItems()
	for i, expectedItem := range expectedItems {
		if sortedItems[i] != expectedItem {
			t.Errorf("expected item %#v, got %#v", expectedItem, sortedItems[i])
		}
	}
}

func checkFeed(t *testing.T, feed feed.Feed, expectedFeed feed.Feed) {
	if feed.Name != expectedFeed.Name {
		t.Errorf("expected feed name %v, got %v", expectedFeed.Name, feed.Name)
	}
	if feed.Type != expectedFeed.Type {
		t.Errorf("expected feed type %v, got %v", expectedFeed.Type, feed.Type)
	}
	if feed.URL != expectedFeed.URL {
		t.Errorf("expected feed URL %v, got %v", expectedFeed.URL, feed.URL)
	}
	if feed.LastRefreshedAt != expectedFeed.LastRefreshedAt {
		t.Errorf("expected last refreshed at %v, got %v", expectedFeed.LastRefreshedAt, feed.LastRefreshedAt)
	}
	if feed.LastRefreshError != expectedFeed.LastRefreshError {
		t.Errorf("expected last refresh error %v, got %v", expectedFeed.LastRefreshError, feed.LastRefreshError)
	}
	checkFeedItems(t, feed, expectedFeed.SortedItems())
}

func TestFeedPrune(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		desc             string
		initialItems     []feed.Item
		limit            int
		params           any
		expectedItems    []feed.Item
		observedRawItems int
	}{{
		desc:          "nil items, limit is 5",
		initialItems:  nil,
		limit:         5,
		expectedItems: []feed.Item{},
	}, {
		desc:          "0 items, limit is 5",
		initialItems:  []feed.Item{},
		limit:         5,
		expectedItems: []feed.Item{},
	}, {
		desc: "3 items, limit is 5",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
		},
		limit: 5,
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
		},
	}, {
		desc: "5 items, limit is 5",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
		},
		limit: 5,
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
		},
	}, {
		desc: "10 items, limit is 5",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
			{RawItem: feed.RawItem{URL: "url6"}, Timestamp: timeutil.HoursAgo(now, 5)},
			{RawItem: feed.RawItem{URL: "url7"}, Timestamp: timeutil.HoursAgo(now, 6)},
			{RawItem: feed.RawItem{URL: "url8"}, Timestamp: timeutil.HoursAgo(now, 7)},
			{RawItem: feed.RawItem{URL: "url9"}, Timestamp: timeutil.HoursAgo(now, 8)},
			{RawItem: feed.RawItem{URL: "url10"}, Timestamp: timeutil.HoursAgo(now, 9)},
		},
		limit: 5,
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
		},
	}, {
		desc: "10 items out of order, limit is 4",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: timeutil.HoursAgo(now, 5)},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 9)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 7)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url6"}, Timestamp: timeutil.HoursAgo(now, 8)},
			{RawItem: feed.RawItem{URL: "url7"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url8"}, Timestamp: timeutil.HoursAgo(now, 6)},
			{RawItem: feed.RawItem{URL: "url9"}, Timestamp: timeutil.HoursAgo(now, 4)},
			{RawItem: feed.RawItem{URL: "url10"}, Timestamp: now},
		},
		limit: 4,
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url10"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url7"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 3)},
		},
	}, {
		desc: "custom max_items param",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
			{RawItem: feed.RawItem{URL: "url6"}, Timestamp: timeutil.HoursAgo(now, 5)},
		},
		limit: 0,
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
		},
		params: map[string]any{"max_items": 3},
	}, {
		desc: "10 items out of order, limit is 0 and observedRawItems is 8",
		// This is not really a realistic test, but it exercises the fact that
		// the limit is more generous if just a few items are observed.
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: timeutil.HoursAgo(now, 5)},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 9)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 7)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url6"}, Timestamp: timeutil.HoursAgo(now, 8)},
			{RawItem: feed.RawItem{URL: "url7"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url8"}, Timestamp: timeutil.HoursAgo(now, 6)},
			{RawItem: feed.RawItem{URL: "url9"}, Timestamp: timeutil.HoursAgo(now, 4)},
			{RawItem: feed.RawItem{URL: "url10"}, Timestamp: now},
		},
		limit:            0,
		observedRawItems: 8,
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url10"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url7"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url9"}, Timestamp: timeutil.HoursAgo(now, 4)},
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: timeutil.HoursAgo(now, 5)},
			{RawItem: feed.RawItem{URL: "url8"}, Timestamp: timeutil.HoursAgo(now, 6)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 7)},
			{RawItem: feed.RawItem{URL: "url6"}, Timestamp: timeutil.HoursAgo(now, 8)},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 9)},
		},
	}, {
		desc: "5 items, limit is 0 and observedRawItems is 150",
		// This is not really a realistic test, but it exercises the fact that
		// the limit is more generous if a lot of items are observed.
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
		},
		limit:            0,
		observedRawItems: 150,
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			f := feed.Feed{
				Items:  make(map[string]*feed.Item),
				Params: test.params,
			}
			for _, item := range test.initialItems {
				f.Items[feed.UID(item.URL)] = &item
			}
			f.Prune(test.limit, test.observedRawItems)
			checkFeedItems(t, f, test.expectedItems)
		})
	}
}

func TestFeedSortedItems(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		desc string

		initialItems []feed.Item

		expectedItems []feed.Item
	}{{
		desc:          "no items",
		initialItems:  nil,
		expectedItems: []feed.Item{},
	}, {
		desc: "1 item",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
		},
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
		},
	}, {
		desc: "5 items already in order",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
		},
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 4)},
		},
	}, {
		desc: "5 items out-of-order",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: timeutil.HoursAgo(now, 5)},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 4)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 2)},
		},
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 1)},
			{RawItem: feed.RawItem{URL: "url5"}, Timestamp: timeutil.HoursAgo(now, 2)},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 3)},
			{RawItem: feed.RawItem{URL: "url4"}, Timestamp: timeutil.HoursAgo(now, 4)},
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: timeutil.HoursAgo(now, 5)},
		},
	}, {
		desc: "5 items with the same timestamp and different URLs",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url5", Position: 5}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url3", Position: 3}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url1", Position: 1}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url4", Position: 4}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2", Position: 2}, Timestamp: now},
		},
		expectedItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1", Position: 1}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url2", Position: 2}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url3", Position: 3}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url4", Position: 4}, Timestamp: now},
			{RawItem: feed.RawItem{URL: "url5", Position: 5}, Timestamp: now},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			f := feed.Feed{
				Items: make(map[string]*feed.Item),
			}
			for _, item := range test.initialItems {
				f.Items[feed.UID(item.URL)] = &item
			}
			sortedItems := f.SortedItems()
			if len(sortedItems) != len(test.expectedItems) {
				t.Errorf("expected %d items, got %d", len(test.expectedItems), len(sortedItems))
				return
			}
			for i, expectedItem := range test.expectedItems {
				if sortedItems[i] != expectedItem {
					t.Errorf("expected item %#v, got %#v", expectedItem, sortedItems[i])
				}
			}
		})
	}
}

func TestMarkAllRead(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		desc string

		initialItems []feed.Item
		before       int64

		expectedRead []bool
	}{{
		desc:         "no items",
		initialItems: []feed.Item{},
		before:       now,
		expectedRead: []bool{},
	}, {
		desc: "multiple items, all marked as read",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now, Read: false},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1), Read: false},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2), Read: false},
		},
		before:       now + 1,
		expectedRead: []bool{true, true, true},
	}, {
		desc: "multiple items, some marked as read",
		initialItems: []feed.Item{
			{RawItem: feed.RawItem{URL: "url1"}, Timestamp: now, Read: false},
			{RawItem: feed.RawItem{URL: "url2"}, Timestamp: timeutil.HoursAgo(now, 1), Read: false},
			{RawItem: feed.RawItem{URL: "url3"}, Timestamp: timeutil.HoursAgo(now, 2), Read: false},
		},
		before:       timeutil.HoursAgo(now, 1),
		expectedRead: []bool{false, true, true},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			f := feed.Feed{
				Items: make(map[string]*feed.Item),
			}
			for _, item := range test.initialItems {
				f.Items[feed.UID(item.URL)] = &item
			}

			// Mark items as read before the specified timestamp.
			f.MarkAllRead(test.before)

			// Verify items are marked as expected.
			for i, item := range f.SortedItems() {
				if item.Read != test.expectedRead[i] {
					t.Errorf("expected item %v read status to be %v, got %v", item.URL, test.expectedRead[i], item.Read)
				}
			}
		})
	}
}

func TestSummary(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		desc string

		feeds       map[string]*feed.Feed
		realFeed    string
		virtualFeed string
		withItems   bool
		itemMapper  bool

		expectedSummary *feed.FeedSummary
	}{{
		desc: "Feed with no items and withItems = true",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name:            "Feed 1",
				URL:             "url1",
				Type:            "xml",
				Items:           map[string]*feed.Item{},
				LastRefreshedAt: now,
			},
		},
		realFeed:  "feed1",
		withItems: true,
		expectedSummary: &feed.FeedSummary{
			UID:         feed.UID("url1"),
			Name:        "Feed 1",
			URL:         "url1",
			Items:       []*feed.ItemSummary{},
			ItemCount:   0,
			ReadCount:   0,
			LastUpdated: now,
			LastItem:    0,
		},
	}, {
		desc: "Feed with items and withItems = true",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				URL:  "url1",
				Type: "xml",
				Items: map[string]*feed.Item{
					"url1": {RawItem: feed.RawItem{URL: "url1", Title: "Title 1"}, Timestamp: now, Read: false},
					"url2": {RawItem: feed.RawItem{URL: "url2", Title: "Title 2"}, Timestamp: timeutil.HoursAgo(now, 1), Read: false},
					"url3": {RawItem: feed.RawItem{URL: "url3", Title: "Title 3"}, Timestamp: timeutil.HoursAgo(now, 2), Read: false},
				},
				LastRefreshedAt: now,
			},
		},
		realFeed:  "feed1",
		withItems: true,
		expectedSummary: &feed.FeedSummary{
			UID:  feed.UID("url1"),
			Name: "Feed 1",
			URL:  "url1",
			Items: []*feed.ItemSummary{
				{UID: "url1", FeedUID: "feed1", FeedName: "Feed 1", URL: "url1", Title: "Title 1", Timestamp: now, Read: false},
				{UID: "url2", FeedUID: "feed1", FeedName: "Feed 1", URL: "url2", Title: "Title 2", Timestamp: timeutil.HoursAgo(now, 1), Read: false},
				{UID: "url3", FeedUID: "feed1", FeedName: "Feed 1", URL: "url3", Title: "Title 3", Timestamp: timeutil.HoursAgo(now, 2), Read: false},
			},
			ItemCount:   3,
			ReadCount:   0,
			LastUpdated: now,
			LastItem:    now,
		},
	}, {
		desc: "Feed with items and withItems = false",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				URL:  "url1",
				Type: "xml",
				Items: map[string]*feed.Item{
					"url1": {RawItem: feed.RawItem{URL: "url1", Title: "Title 1"}, Timestamp: now, Read: false},
					"url2": {RawItem: feed.RawItem{URL: "url2", Title: "Title 2"}, Timestamp: timeutil.HoursAgo(now, 1), Read: true},
				},
				LastRefreshedAt: now,
			},
		},
		realFeed:  "feed1",
		withItems: false,
		expectedSummary: &feed.FeedSummary{
			UID:         feed.UID("url1"),
			Name:        "Feed 1",
			URL:         "url1",
			ItemCount:   2,
			ReadCount:   1,
			LastUpdated: now,
			LastItem:    now,
		},
	}, {
		desc: "A virtual feed named 'virtual' with items and a valid itemMapper pointing at two other feeds",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Items: map[string]*feed.Item{
					"url1": {RawItem: feed.RawItem{URL: "url1", Title: "Title 1"}, Timestamp: now, Read: true, FeedUID: "feed1"},
				},
				LastRefreshedAt: now,
			},
			"feed2": {
				Name: "Feed 2",
				Items: map[string]*feed.Item{
					"url2": {RawItem: feed.RawItem{URL: "url2", Title: "Title 2"}, Timestamp: timeutil.HoursAgo(now, 1), Read: false, FeedUID: "feed2"},
				},
				LastRefreshedAt: now,
			},
		},
		virtualFeed: "Virtual",
		withItems:   true,
		itemMapper:  true,
		expectedSummary: &feed.FeedSummary{
			UID:  "virtual",
			Name: "Virtual",
			Items: []*feed.ItemSummary{
				{UID: "url1", FeedUID: "feed1", FeedName: "Feed 1", URL: "url1", Title: "Title 1", Timestamp: now, Read: false},
				{UID: "url2", FeedUID: "feed2", FeedName: "Feed 2", URL: "url2", Title: "Title 2", Timestamp: timeutil.HoursAgo(now, 1), Read: false},
			},
			ItemCount:   2,
			ReadCount:   1,
			LastUpdated: now,
			LastItem:    now,
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var f *feed.Feed
			var itemMapper map[string]*feed.Feed
			if test.virtualFeed != "" {
				f = &feed.Feed{
					Name:            test.virtualFeed,
					Items:           make(map[string]*feed.Item),
					LastRefreshedAt: now,
				}
				itemMapper = make(map[string]*feed.Feed)
				for _, feed := range test.feeds {
					for _, item := range feed.Items {
						f.Items[item.UID()] = item
						itemMapper[item.UID()] = feed
					}
				}
			} else {
				f = test.feeds[test.realFeed]
			}

			summary := f.Summary(test.withItems, itemMapper)
			if summary.UID != test.expectedSummary.UID ||
				summary.Name != test.expectedSummary.Name ||
				summary.ItemCount != test.expectedSummary.ItemCount ||
				summary.ReadCount != test.expectedSummary.ReadCount ||
				summary.LastUpdated != test.expectedSummary.LastUpdated ||
				summary.LastItem != test.expectedSummary.LastItem {
				t.Errorf("expected summary %#v, got %#v", test.expectedSummary, summary)
			}

			if test.withItems {
				checkFeedItems(t, *f, f.SortedItems())
			}
		})
	}
}

func TestFeedRefresh(t *testing.T) {
	now := timeutil.Now()

	tests := []struct {
		desc string

		initialFeed feed.Feed
		items       []feed.RawItem
		fetchErr    error

		expectedFeed   feed.Feed
		expectedErrMsg string
	}{{
		desc: "the feed had no items and items is nil and fetchErr is not nil",
		initialFeed: feed.Feed{
			Name:  "Feed 1",
			URL:   "url1",
			Items: map[string]*feed.Item{},
		},
		items:    nil,
		fetchErr: fmt.Errorf("fetch error"),
		expectedFeed: feed.Feed{
			Name:             "Feed 1",
			URL:              "url1",
			Items:            map[string]*feed.Item{},
			LastRefreshError: "fetch error",
		},
	}, {
		desc: "the feed had no items and items is nil and fetchErr is nil",
		initialFeed: feed.Feed{
			Name:  "Feed 1",
			URL:   "url1",
			Items: map[string]*feed.Item{},
		},
		items:    nil,
		fetchErr: nil,
		expectedFeed: feed.Feed{
			Name:             "Feed 1",
			URL:              "url1",
			Items:            map[string]*feed.Item{},
			LastRefreshError: "no items found in the last refresh",
		},
	}, {
		desc: "the feed had no items and items contain one valid item",
		initialFeed: feed.Feed{
			Name:  "Feed 1",
			URL:   "url1",
			Items: map[string]*feed.Item{},
		},
		items: []feed.RawItem{
			{URL: "url1", Title: "Title 1"},
		},
		fetchErr: nil,
		expectedFeed: feed.Feed{
			Name: "Feed 1",
			URL:  "url1",
			Items: map[string]*feed.Item{
				feed.UID("url1"): {RawItem: feed.RawItem{URL: "url1", Title: "Title 1"}, FeedUID: feed.UID("url1"), Timestamp: now},
			},
			LastRefreshedAt: now,
		},
	}, {
		desc: "the feed had items already and items has invalid and valid items",
		initialFeed: feed.Feed{
			Name: "Feed 1",
			URL:  "url1",
			Items: map[string]*feed.Item{
				feed.UID("url1"): {RawItem: feed.RawItem{URL: "url1", Title: "Title 1", Position: 0}, FeedUID: feed.UID("url1"), Timestamp: now},
			},
		},
		items: []feed.RawItem{
			{URL: "url1", Title: "Updated Title 1", Position: 0},
			{URL: "url2", Title: "Title 2", Position: 1},
			{URL: "", Title: "Invalid Item", Position: 2},
		},
		fetchErr: nil,
		expectedFeed: feed.Feed{
			Name: "Feed 1",
			URL:  "url1",
			Items: map[string]*feed.Item{
				feed.UID("url1"): {RawItem: feed.RawItem{URL: "url1", Title: "Updated Title 1", Position: 0}, FeedUID: feed.UID("url1"), Timestamp: now},
				feed.UID("url2"): {RawItem: feed.RawItem{URL: "url2", Title: "Title 2", Position: 1}, FeedUID: feed.UID("url1"), Timestamp: now},
			},
			LastRefreshedAt: now,
		},
	}, {
		desc: "successful refresh clears previous error",
		initialFeed: feed.Feed{
			Name:             "Feed 1",
			URL:              "url1",
			Items:            map[string]*feed.Item{},
			LastRefreshError: "previous error",
		},
		items: []feed.RawItem{
			{URL: "url1", Title: "Title 1"},
		},
		fetchErr: nil,
		expectedFeed: feed.Feed{
			Name: "Feed 1",
			URL:  "url1",
			Items: map[string]*feed.Item{
				feed.UID("url1"): {RawItem: feed.RawItem{URL: "url1", Title: "Title 1"}, FeedUID: feed.UID("url1"), Timestamp: now},
			},
			LastRefreshedAt:  now,
			LastRefreshError: "",
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			f := test.initialFeed
			f.Refresh(test.items, now, test.fetchErr)
			checkFeed(t, f, test.expectedFeed)
		})
	}
}

func TestFeedUID(t *testing.T) {
	tests := []struct {
		desc string

		feed feed.Feed

		expected string
	}{{
		desc: "feed with URL",
		feed: feed.Feed{
			Name: "Feed 1",
			URL:  "http://example.com/feed",
		},
		expected: feed.UID("http://example.com/feed"),
	}, {
		desc: "feed without URL",
		feed: feed.Feed{
			Name: "Feed 1",
			URL:  "",
		},
		expected: "feed 1",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := test.feed.UID()
			if result != test.expected {
				t.Errorf("expected %v, got %v", test.expected, result)
			}
		})
	}
}
