package mem_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/fetch"
	"github.com/alnvdl/varys/internal/list"
	"github.com/alnvdl/varys/internal/list/mem"
	"github.com/alnvdl/varys/internal/timeutil"
)

func TestListRefresh(t *testing.T) {
	t.Parallel()
	now := timeutil.Now()

	mockFetcher := func(p fetch.FetchParams) ([]feed.RawItem, int64, error) {
		switch p.URL {
		case "http://example.com/feed1":
			return []feed.RawItem{
				{URL: "http://example.com/item1", Title: "Item 1"},
			}, now, nil
		case "http://example.com/feed2":
			return []feed.RawItem{
				{URL: "http://example.com/item2", Title: "Item 2"},
			}, now, nil
		case "http://example.com/feed3":
			return nil, 0, errors.New("oh no")
		default:
			return nil, 0, errors.New("unknown feed URL")
		}
	}

	tests := []struct {
		desc           string
		initialFeeds   map[string]*feed.Feed
		expectedFeeds  map[string]*feed.Feed
		expectedErrMsg string
	}{{
		desc:          "feed list is empty",
		initialFeeds:  map[string]*feed.Feed{},
		expectedFeeds: map[string]*feed.Feed{},
	}, {
		desc: "feed list has 1 feed",
		initialFeeds: map[string]*feed.Feed{
			"feed1": {
				Name:  "Feed 1",
				URL:   "http://example.com/feed1",
				Items: map[string]*feed.Item{},
			},
		},
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					feed.UID("http://example.com/item1"): {
						RawItem: feed.RawItem{
							URL:   "http://example.com/item1",
							Title: "Item 1",
						},
						FeedUID:   feed.UID("http://example.com/feed1"),
						Timestamp: now,
					},
				},
				LastRefreshedAt: now,
			},
		},
	}, {
		desc: "feed list has 3 feeds",
		initialFeeds: map[string]*feed.Feed{
			"feed1": {
				Name:  "Feed 1",
				URL:   "http://example.com/feed1",
				Items: map[string]*feed.Item{},
			},
			"feed2": {
				Name:  "Feed 2",
				URL:   "http://example.com/feed2",
				Items: map[string]*feed.Item{},
			},
			"feed3": {
				Name:  "Feed 3",
				URL:   "http://example.com/feed3",
				Items: map[string]*feed.Item{},
			},
		},
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					feed.UID("http://example.com/item1"): {
						RawItem: feed.RawItem{
							URL:   "http://example.com/item1",
							Title: "Item 1",
						},
						FeedUID:   feed.UID("http://example.com/feed1"),
						Timestamp: now,
					},
				},
				LastRefreshedAt: now,
			},
			"feed2": {
				Name: "Feed 2",
				URL:  "http://example.com/feed2",
				Items: map[string]*feed.Item{
					feed.UID("http://example.com/item2"): {
						RawItem: feed.RawItem{
							URL:   "http://example.com/item2",
							Title: "Item 2",
						},
						FeedUID:   feed.UID("http://example.com/feed2"),
						Timestamp: now,
					},
				},
				LastRefreshedAt: now,
			},
			"feed3": {
				Name:             "Feed 3",
				URL:              "http://example.com/feed3",
				LastRefreshError: "oh no",
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			l := mem.NewList(mem.ListParams{
				Fetcher: mockFetcher,
			})
			mem.SetFeeds(l, test.initialFeeds)
			l.Refresh(false)
			for key, expectedFeed := range test.expectedFeeds {
				actualFeed, ok := mem.Feeds(l)[key]
				if !ok {
					t.Fatalf("expected feed %s to be present", key)
				}
				checkFeed(t, *actualFeed, *expectedFeed)
			}
		})
	}
}

func TestAutoRefresh(t *testing.T) {
	t.Parallel()

	// Initial mock fetcher data.
	now := timeutil.Now()
	t1 := now     // First and second refresh.
	t2 := now + 1 // Third refresh.

	mockResponses := map[string][]feed.RawItem{
		"http://example.com/feed1": {
			{URL: "http://example.com/item1", Title: "Item 1"},
		},
		"http://example.com/feed2": {
			{URL: "http://example.com/item2", Title: "Item 2"},
		},
	}

	mockFetcher := func(p fetch.FetchParams) ([]feed.RawItem, int64, error) {
		if items, ok := mockResponses[p.URL]; ok {
			return items, now, nil
		}
		return nil, 0, errors.New("unknown feed URL")
	}

	refreshNotify := make(chan bool, 1)
	l := mem.NewList(mem.ListParams{
		InitialFeeds: []*list.InputFeed{{
			Name: "Feed 1",
			URL:  "http://example.com/feed1",
			Type: "xml",
		}, {
			Name: "Feed 2",
			URL:  "http://example.com/feed2",
			Type: "xml",
		}},
		RefreshInterval: 1 * time.Second,
		Fetcher:         mockFetcher,
		RefreshCallback: func() {
			refreshNotify <- true
		},
	})
	select {
	case <-time.After(2 * time.Second):
		t.Fatalf("expected refresh to be triggered")
	case <-refreshNotify:
		// The refresh callback was triggered.
	}

	// Check initial feed state.
	expectedFeeds := map[string]*feed.Feed{
		feed.UID("http://example.com/feed1"): {
			Name:            "Feed 1",
			URL:             "http://example.com/feed1",
			Type:            "xml",
			LastRefreshedAt: t1,
			Items: map[string]*feed.Item{
				feed.UID("http://example.com/item1"): {
					RawItem: feed.RawItem{
						URL:   "http://example.com/item1",
						Title: "Item 1",
					},
					FeedUID:   feed.UID("http://example.com/feed1"),
					Timestamp: t1,
				},
			},
		},
		feed.UID("http://example.com/feed2"): {
			Name:            "Feed 2",
			URL:             "http://example.com/feed2",
			Type:            "xml",
			LastRefreshedAt: t1,
			Items: map[string]*feed.Item{
				feed.UID("http://example.com/item2"): {
					RawItem: feed.RawItem{
						URL:   "http://example.com/item2",
						Title: "Item 2",
					},
					FeedUID:   feed.UID("http://example.com/feed2"),
					Timestamp: t1,
				},
			},
		},
	}
	for key, expectedFeed := range expectedFeeds {
		actualFeed, ok := mem.Feeds(l)[key]
		if !ok {
			t.Fatalf("expected feed %s to be present", key)
		}
		checkFeed(t, *actualFeed, *expectedFeed)
	}

	// Modify the mock responses to return updated items.
	mockResponses["http://example.com/feed1"] = []feed.RawItem{
		{URL: "http://example.com/item1", Title: "Item 1"},
		{URL: "http://example.com/item3", Title: "Item 3"},
	}
	mockResponses["http://example.com/feed2"] = []feed.RawItem{
		{URL: "http://example.com/item2", Title: "Item 2"},
		{URL: "http://example.com/item4", Title: "Item 4"},
	}
	now++

	// Wait for the refresh callback.
	select {
	case <-time.After(2 * time.Second):
		t.Fatalf("expected refresh to be triggered")
	case <-refreshNotify:
		// The refresh callback was triggered.
	}

	// Check updated feeds.
	expectedFeeds = map[string]*feed.Feed{
		feed.UID("http://example.com/feed1"): {
			Name:            "Feed 1",
			URL:             "http://example.com/feed1",
			Type:            "xml",
			LastRefreshedAt: t2,
			Items: map[string]*feed.Item{
				feed.UID("http://example.com/item1"): {
					RawItem: feed.RawItem{
						URL:   "http://example.com/item1",
						Title: "Item 1",
					},
					FeedUID:   feed.UID("http://example.com/feed1"),
					Timestamp: t1,
				},
				feed.UID("http://example.com/item3"): {
					RawItem: feed.RawItem{
						URL:   "http://example.com/item3",
						Title: "Item 3",
					},
					FeedUID:   feed.UID("http://example.com/feed1"),
					Timestamp: t2,
				},
			},
		},
		feed.UID("http://example.com/feed2"): {
			Name:            "Feed 2",
			URL:             "http://example.com/feed2",
			Type:            "xml",
			LastRefreshedAt: t2,
			Items: map[string]*feed.Item{
				feed.UID("http://example.com/item2"): {
					RawItem: feed.RawItem{
						URL:   "http://example.com/item2",
						Title: "Item 2",
					},
					FeedUID:   feed.UID("http://example.com/feed2"),
					Timestamp: t1,
				},
				feed.UID("http://example.com/item4"): {
					RawItem: feed.RawItem{
						URL:   "http://example.com/item4",
						Title: "Item 4",
					},
					FeedUID:   feed.UID("http://example.com/feed2"),
					Timestamp: t2,
				},
			},
		},
	}
	for key, expectedFeed := range expectedFeeds {
		actualFeed, ok := mem.Feeds(l)[key]
		if !ok {
			t.Fatalf("expected feed %s to be present", key)
		}
		checkFeed(t, *actualFeed, *expectedFeed)
	}

	l.Close()
}
