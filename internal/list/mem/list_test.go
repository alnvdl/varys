package mem_test

import (
	"fmt"
	"maps"
	"testing"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/list"
	"github.com/alnvdl/varys/internal/list/mem"
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
	if feed.Params != expectedFeed.Params {
		t.Errorf("expected feed params %v, got %v", expectedFeed.Params, feed.Params)
	}

	checkFeedItems(t, feed, expectedFeed.SortedItems())
}

func compareFeedSummary(t *testing.T, got, want *feed.FeedSummary) {
	if got.Name != want.Name {
		t.Errorf("expected name %s, got %s", want.Name, got.Name)
	}
	if got.URL != want.URL {
		t.Errorf("expected URL %s, got %s", want.URL, got.URL)
	}
	if got.ItemCount != want.ItemCount {
		t.Errorf("expected item count %d, got %d", want.ItemCount, got.ItemCount)
	}
	if len(got.Items) != len(want.Items) {
		t.Errorf("expected %d items, got %d", len(want.Items), len(got.Items))
	}
	for i, item := range got.Items {
		if item.UID != want.Items[i].UID {
			t.Errorf("expected item UID %s, got %s", want.Items[i].UID, item.UID)
		}
		if item.FeedUID != want.Items[i].FeedUID {
			t.Errorf("expected item feed UID %s, got %s", want.Items[i].FeedUID, item.FeedUID)
		}
		if item.FeedName != want.Items[i].FeedName {
			t.Errorf("expected item feed name %s, got %s", want.Items[i].FeedName, item.FeedName)
		}
		if item.URL != want.Items[i].URL {
			t.Errorf("expected item URL %s, got %s", want.Items[i].URL, item.URL)
		}
		if item.Title != want.Items[i].Title {
			t.Errorf("expected item title %s, got %s", want.Items[i].Title, item.Title)
		}
		if item.Timestamp != want.Items[i].Timestamp {
			t.Errorf("expected item timestamp %d, got %d", want.Items[i].Timestamp, item.Timestamp)
		}
		if item.Authors != want.Items[i].Authors {
			t.Errorf("expected item authors %s, got %s", want.Items[i].Authors, item.Authors)
		}
		if item.Read != want.Items[i].Read {
			t.Errorf("expected item read %t, got %t", want.Items[i].Read, item.Read)
		}
		if item.Content != want.Items[i].Content {
			t.Errorf("expected item content %s, got %s", want.Items[i].Content, item.Content)
		}
	}
}

func TestListSummary(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		desc  string
		feeds map[string]*feed.Feed
		want  []*feed.FeedSummary
	}{{
		desc:  "no feeds",
		feeds: map[string]*feed.Feed{},
		want:  []*feed.FeedSummary{{Name: "All", ItemCount: 0}},
	}, {
		desc: "1 feed with items",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}},
				},
			},
		},
		want: []*feed.FeedSummary{
			{Name: "All", ItemCount: 1},
			{Name: "Feed 1", URL: "http://example.com/feed1", ItemCount: 1},
		},
	}, {
		desc: "3 feeds, one without items",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}},
				},
			},
			"feed2": {
				Name: "Feed 2",
				Type: "xml",
				URL:  "http://example.com/feed2",
				Items: map[string]*feed.Item{
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}},
				},
			},
			"feed3": {
				Name:  "Feed 3",
				Type:  "xml",
				URL:   "http://example.com/feed3",
				Items: map[string]*feed.Item{},
			},
		},
		want: []*feed.FeedSummary{
			{Name: "All", ItemCount: 2},
			{Name: "Feed 1", URL: "http://example.com/feed1", ItemCount: 1},
			{Name: "Feed 2", URL: "http://example.com/feed2", ItemCount: 1},
			{Name: "Feed 3", URL: "http://example.com/feed3", ItemCount: 0},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			l := mem.NewList(mem.ListParams{})
			mem.SetFeedsMap(l, test.feeds)
			got := l.Summary()
			if len(got) != len(test.want) {
				t.Fatalf("expected %d summaries, got %d", len(test.want), len(got))
			}
			for i, summary := range got {
				compareFeedSummary(t, summary, test.want[i])
			}
		})
	}
}

func TestListFeedSummary(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		desc  string
		feeds map[string]*feed.Feed
		uid   string
		want  *feed.FeedSummary
	}{{
		desc:  "feed does not exist",
		feeds: map[string]*feed.Feed{},
		uid:   "nonexistent",
		want:  nil,
	}, {
		desc: "feed exists and has items",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}},
				},
			},
		},
		uid: "feed1",
		want: &feed.FeedSummary{
			UID:       "feed1",
			Name:      "Feed 1",
			URL:       "http://example.com/feed1",
			ItemCount: 1,
			Items: []*feed.ItemSummary{{
				UID:       feed.UID("http://example.com/item1"),
				FeedUID:   feed.UID("http://example.com/feed1"),
				FeedName:  "Feed 1",
				URL:       "http://example.com/item1",
				Title:     "Item 1",
				Timestamp: 0,
				Read:      false,
			}},
		},
	}, {
		desc: "feed is the all feed compiled from two other existing feeds",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {
						RawItem:   feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"},
						Timestamp: 2,
					},
				},
			},
			"feed2": {
				Name: "Feed 2",
				Type: "xml",
				URL:  "http://example.com/feed2",
				Items: map[string]*feed.Item{
					"item2": {
						RawItem:   feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"},
						Timestamp: 1,
					},
				},
			},
		},
		uid: "all",
		want: &feed.FeedSummary{
			UID:       "all",
			Name:      "All",
			ItemCount: 2,
			Items: []*feed.ItemSummary{{
				UID:       feed.UID("http://example.com/item1"),
				FeedUID:   feed.UID("http://example.com/feed1"),
				FeedName:  "Feed 1",
				URL:       "http://example.com/item1",
				Title:     "Item 1",
				Timestamp: 2,
			}, {
				UID:       feed.UID("http://example.com/item2"),
				FeedUID:   feed.UID("http://example.com/feed2"),
				FeedName:  "Feed 2",
				URL:       "http://example.com/item2",
				Title:     "Item 2",
				Timestamp: 1,
			},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			l := mem.NewList(mem.ListParams{})
			mem.SetFeedsMap(l, test.feeds)
			got := l.FeedSummary(test.uid)
			if got == nil && test.want != nil {
				t.Fatalf("expected summary, got nil")
			}
			if got != nil && test.want == nil {
				t.Fatalf("expected nil, got summary")
			}
			if got != nil && test.want != nil {
				compareFeedSummary(t, got, test.want)
			}
		})
	}
}

func TestListFeedItem(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		desc  string
		feeds map[string]*feed.Feed
		fuid  string
		iuid  string
		want  *feed.ItemSummary
	}{{
		desc:  "feed does not exist",
		feeds: map[string]*feed.Feed{},
		fuid:  "nonexistent",
		iuid:  "item1",
		want:  nil,
	}, {
		desc: "item does not exist",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}},
				},
			},
		},
		fuid: "feed1",
		iuid: "nonexistent",
		want: nil,
	}, {
		desc: "feed and item exist",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {
						RawItem: feed.RawItem{
							URL:     "http://example.com/item1",
							Title:   "Item 1",
							Content: "Content of item 1",
						},
						FeedUID:   "feed1",
						Timestamp: 1633024800,
						Read:      false,
					},
				},
			},
		},
		fuid: "feed1",
		iuid: "item1",
		want: &feed.ItemSummary{
			UID:       feed.UID("http://example.com/item1"),
			FeedUID:   feed.UID("http://example.com/feed1"),
			FeedName:  "Feed 1",
			URL:       "http://example.com/item1",
			Title:     "Item 1",
			Content:   "Content of item 1",
			Timestamp: 1633024800,
			Read:      false,
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			l := mem.NewList(mem.ListParams{})
			mem.SetFeedsMap(l, test.feeds)
			got := l.FeedItem(test.fuid, test.iuid)
			if got == nil && test.want != nil {
				t.Fatalf("expected item summary, got nil")
			}
			if got != nil && test.want == nil {
				t.Fatalf("expected nil, got item summary")
			}
			if got != nil && test.want != nil {
				if got.UID != test.want.UID ||
					got.FeedUID != test.want.FeedUID ||
					got.FeedName != test.want.FeedName ||
					got.URL != test.want.URL ||
					got.Title != test.want.Title ||
					got.Content != test.want.Content ||
					got.Timestamp != test.want.Timestamp ||
					got.Read != test.want.Read {
					t.Errorf("expected item summary %#v, got %#v", test.want, got)
				}
			}
		})
	}
}

func TestAllFeed(t *testing.T) {
	t.Parallel()
	now := timeutil.Now()

	tests := []struct {
		desc            string
		feeds           map[string]*feed.Feed
		withItems       bool
		expectedSummary *feed.FeedSummary
	}{{
		desc:      "when there are no feeds",
		feeds:     map[string]*feed.Feed{},
		withItems: true,
		expectedSummary: &feed.FeedSummary{
			UID:         "all",
			Name:        "All",
			Items:       []*feed.ItemSummary{},
			ItemCount:   0,
			ReadCount:   0,
			LastUpdated: now,
		},
	}, {
		desc: "when all input feeds are without items",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name:            "Feed 1",
				Items:           map[string]*feed.Item{},
				LastRefreshedAt: now,
			},
			"feed2": {
				Name:            "Feed 2",
				Items:           map[string]*feed.Item{},
				LastRefreshedAt: now,
			},
		},
		withItems: true,
		expectedSummary: &feed.FeedSummary{
			UID:         "all",
			Name:        "All",
			Items:       []*feed.ItemSummary{},
			ItemCount:   0,
			ReadCount:   0,
			LastUpdated: now,
		},
	}, {
		desc: "when there are 2 different feeds with 5 items each",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				URL:  "url1",
				Items: map[string]*feed.Item{
					"url1": {RawItem: feed.RawItem{URL: "url1", Title: "Title 1"}, Timestamp: now, Read: false, FeedUID: feed.UID("url1")},
					"url2": {RawItem: feed.RawItem{URL: "url2", Title: "Title 2"}, Timestamp: timeutil.HoursAgo(now, 1), Read: false, FeedUID: feed.UID("url1")},
					"url3": {RawItem: feed.RawItem{URL: "url3", Title: "Title 3"}, Timestamp: timeutil.HoursAgo(now, 2), Read: false, FeedUID: feed.UID("url1")},
					"url4": {RawItem: feed.RawItem{URL: "url4", Title: "Title 4"}, Timestamp: timeutil.HoursAgo(now, 3), Read: false, FeedUID: feed.UID("url1")},
					"url5": {RawItem: feed.RawItem{URL: "url5", Title: "Title 5"}, Timestamp: timeutil.HoursAgo(now, 4), Read: false, FeedUID: feed.UID("url1")},
				},
				LastRefreshedAt: now,
			},
			"feed2": {
				Name: "Feed 2",
				URL:  "url2",
				Items: map[string]*feed.Item{
					"url6":  {RawItem: feed.RawItem{URL: "url6", Title: "Title 6"}, Timestamp: timeutil.HoursAgo(now, 5), Read: false, FeedUID: feed.UID("url2")},
					"url7":  {RawItem: feed.RawItem{URL: "url7", Title: "Title 7"}, Timestamp: timeutil.HoursAgo(now, 6), Read: false, FeedUID: feed.UID("url2")},
					"url8":  {RawItem: feed.RawItem{URL: "url8", Title: "Title 8"}, Timestamp: timeutil.HoursAgo(now, 7), Read: false, FeedUID: feed.UID("url2")},
					"url9":  {RawItem: feed.RawItem{URL: "url9", Title: "Title 9"}, Timestamp: timeutil.HoursAgo(now, 8), Read: false, FeedUID: feed.UID("url2")},
					"url10": {RawItem: feed.RawItem{URL: "url10", Title: "Title 10"}, Timestamp: timeutil.HoursAgo(now, 9), Read: false, FeedUID: feed.UID("url2")},
				},
				LastRefreshedAt: now,
			},
		},
		withItems: true,
		expectedSummary: &feed.FeedSummary{
			UID:  "all",
			Name: "All",
			Items: []*feed.ItemSummary{
				{UID: feed.UID("url1"), FeedUID: feed.UID("url1"), FeedName: "Feed 1", URL: "url1", Title: "Title 1", Timestamp: now, Read: false},
				{UID: feed.UID("url2"), FeedUID: feed.UID("url1"), FeedName: "Feed 1", URL: "url2", Title: "Title 2", Timestamp: timeutil.HoursAgo(now, 1), Read: false},
				{UID: feed.UID("url3"), FeedUID: feed.UID("url1"), FeedName: "Feed 1", URL: "url3", Title: "Title 3", Timestamp: timeutil.HoursAgo(now, 2), Read: false},
				{UID: feed.UID("url4"), FeedUID: feed.UID("url1"), FeedName: "Feed 1", URL: "url4", Title: "Title 4", Timestamp: timeutil.HoursAgo(now, 3), Read: false},
				{UID: feed.UID("url5"), FeedUID: feed.UID("url1"), FeedName: "Feed 1", URL: "url5", Title: "Title 5", Timestamp: timeutil.HoursAgo(now, 4), Read: false},
				{UID: feed.UID("url6"), FeedUID: feed.UID("url2"), FeedName: "Feed 2", URL: "url6", Title: "Title 6", Timestamp: timeutil.HoursAgo(now, 5), Read: false},
				{UID: feed.UID("url7"), FeedUID: feed.UID("url2"), FeedName: "Feed 2", URL: "url7", Title: "Title 7", Timestamp: timeutil.HoursAgo(now, 6), Read: false},
				{UID: feed.UID("url8"), FeedUID: feed.UID("url2"), FeedName: "Feed 2", URL: "url8", Title: "Title 8", Timestamp: timeutil.HoursAgo(now, 7), Read: false},
				{UID: feed.UID("url9"), FeedUID: feed.UID("url2"), FeedName: "Feed 2", URL: "url9", Title: "Title 9", Timestamp: timeutil.HoursAgo(now, 8), Read: false},
				{UID: feed.UID("url10"), FeedUID: feed.UID("url2"), FeedName: "Feed 2", URL: "url10", Title: "Title 10", Timestamp: timeutil.HoursAgo(now, 9), Read: false},
			},
			ItemCount:   10,
			ReadCount:   0,
			LastUpdated: now,
		},
	}, {
		desc: "when there are 3 different feeds each with 50 items",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				URL:  "url1",
				Items: func() map[string]*feed.Item {
					items := make(map[string]*feed.Item)
					for i := range 100 {
						url := fmt.Sprintf("url%d", i*3+1)
						items[feed.UID(url)] = &feed.Item{
							RawItem: feed.RawItem{
								URL:      url,
								Title:    fmt.Sprintf("Title %d", i*3+1),
								Position: i*3 + 1,
							},
							Timestamp: now - int64(i),
							Read:      false,
							FeedUID:   feed.UID("url1"),
						}
					}
					return items
				}(),
				LastRefreshedAt: now,
			},
			"feed2": {
				Name: "Feed 2",
				URL:  "url2",
				Items: func() map[string]*feed.Item {
					items := make(map[string]*feed.Item)
					for i := range 100 {
						url := fmt.Sprintf("url%d", i*3+2)
						items[feed.UID(url)] = &feed.Item{
							RawItem: feed.RawItem{
								URL:      url,
								Title:    fmt.Sprintf("Title %d", i*3+2),
								Position: i*3 + 2,
							},
							Timestamp: now - int64(i),
							Read:      false,
							FeedUID:   feed.UID("url2"),
						}
					}
					return items
				}(),
				LastRefreshedAt: now,
			},
			"feed3": {
				Name: "Feed 3",
				URL:  "url3",
				Items: func() map[string]*feed.Item {
					items := make(map[string]*feed.Item)
					for i := range 100 {
						url := fmt.Sprintf("url%d", i*3+3)
						items[feed.UID(url)] = &feed.Item{
							RawItem: feed.RawItem{
								URL:      url,
								Title:    fmt.Sprintf("Title %d", i*3+3),
								Position: i*3 + 3,
							},
							Timestamp: now - int64(i),
							Read:      false,
							FeedUID:   feed.UID("url3"),
						}
					}
					return items
				}(),
				LastRefreshedAt: now,
			},
		},
		withItems: true,
		expectedSummary: &feed.FeedSummary{
			UID:  "all",
			Name: "All",
			Items: func() []*feed.ItemSummary {
				var items []*feed.ItemSummary
				for i := range 300 {
					url := fmt.Sprintf("url%d", i+1)
					feedUID := feed.UID(fmt.Sprintf("url%d", (i%3)+1))
					items = append(items, &feed.ItemSummary{
						UID:       feed.UID(url),
						FeedUID:   feedUID,
						FeedName:  fmt.Sprintf("Feed %d", (i%3)+1),
						URL:       url,
						Title:     fmt.Sprintf("Title %d", i+1),
						Timestamp: now - int64(i/3),
						Read:      false,
					})
				}
				return items
			}(),
			ItemCount:   300,
			ReadCount:   0,
			LastUpdated: now,
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			summary := mem.AllFeed(maps.Values(test.feeds), test.withItems)
			if summary.UID != test.expectedSummary.UID ||
				summary.Name != test.expectedSummary.Name ||
				summary.ItemCount != test.expectedSummary.ItemCount ||
				summary.ReadCount != test.expectedSummary.ReadCount ||
				summary.LastUpdated != test.expectedSummary.LastUpdated {
				t.Errorf("expected summary %#v, got %#v", test.expectedSummary, summary)
			}

			if test.withItems {
				if len(summary.Items) != len(test.expectedSummary.Items) {
					t.Errorf("expected %d items, got %d", len(test.expectedSummary.Items), len(summary.Items))
				}
				for i, expectedItem := range test.expectedSummary.Items {
					if *summary.Items[i] != *expectedItem {
						t.Errorf("expected item %#v, got %#v", expectedItem, summary.Items[i])
					}
				}
			}
		})
	}
}

func TestListMarkRead(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc           string
		feeds          map[string]*feed.Feed
		fuid           string
		iuid           string
		before         int64
		expectedResult bool
		expectedFeeds  map[string]*feed.Feed
	}{{
		desc: "feed does not exist",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false},
				},
			},
		},
		fuid:           "nonexistent",
		iuid:           "item1",
		before:         timeutil.Now(),
		expectedResult: false,
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false},
				},
			},
		},
	}, {
		desc: "feed exists, but item does not exist",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false},
				},
			},
		},
		fuid:           "feed1",
		iuid:           "nonexistent",
		before:         timeutil.Now(),
		expectedResult: false,
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false},
				},
			},
		},
	}, {
		desc: "feed exists and item is empty",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false, Timestamp: 3},
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: false, Timestamp: 2},
					"item3": {RawItem: feed.RawItem{URL: "http://example.com/item3", Title: "Item 3"}, Read: false, Timestamp: 1},
				},
			},
		},
		fuid:           "feed1",
		iuid:           "",
		before:         4,
		expectedResult: true,
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: true, Timestamp: 3},
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: true, Timestamp: 2},
					"item3": {RawItem: feed.RawItem{URL: "http://example.com/item3", Title: "Item 3"}, Read: true, Timestamp: 1},
				},
			},
		},
	}, {
		desc: "feed exists and item is empty, some items marked as read",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false, Timestamp: 3},
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: false, Timestamp: 2},
					"item3": {RawItem: feed.RawItem{URL: "http://example.com/item3", Title: "Item 3"}, Read: false, Timestamp: 1},
				},
			},
		},
		fuid:           "feed1",
		iuid:           "",
		before:         2,
		expectedResult: true,
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false, Timestamp: 3},
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: true, Timestamp: 2},
					"item3": {RawItem: feed.RawItem{URL: "http://example.com/item3", Title: "Item 3"}, Read: true, Timestamp: 1},
				},
			},
		},
	}, {
		desc: "feed and item exist",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false},
				},
			},
		},
		fuid:           "feed1",
		iuid:           "item1",
		before:         timeutil.Now(),
		expectedResult: true,
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: true},
				},
			},
		},
	}, {
		desc: "the all feed is being marked as read",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false},
				},
			},
			"feed2": {
				Name: "Feed 2",
				Type: "xml",
				URL:  "http://example.com/feed2",
				Items: map[string]*feed.Item{
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: false},
				},
			},
		},
		fuid:           "all",
		iuid:           "",
		before:         timeutil.Now(),
		expectedResult: true,
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: true},
				},
			},
			"feed2": {
				Name: "Feed 2",
				Type: "xml",
				URL:  "http://example.com/feed2",
				Items: map[string]*feed.Item{
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: true},
				},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			l := mem.NewList(mem.ListParams{})
			mem.SetFeedsMap(l, test.feeds)
			result := l.MarkRead(test.fuid, test.iuid, test.before)
			if result != test.expectedResult {
				t.Errorf("expected result %v, got %v", test.expectedResult, result)
			}
			for key, expectedFeed := range test.expectedFeeds {
				actualFeed, ok := mem.FeedsMap(l)[key]
				if !ok {
					t.Fatalf("expected feed %s to be present", key)
				}
				checkFeed(t, *actualFeed, *expectedFeed)
			}
		})
	}
}

func TestListLoadFeeds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc          string
		initialFeeds  map[string]*feed.Feed
		inputFeeds    []*list.InputFeed
		expectedFeeds map[string]*feed.Feed
	}{{
		desc:          "list has 0 feeds, and the inputFeeds are empty",
		initialFeeds:  map[string]*feed.Feed{},
		inputFeeds:    []*list.InputFeed{},
		expectedFeeds: map[string]*feed.Feed{},
	}, {
		desc:         "list has 0 feeds, and inputFields has 2 feeds",
		initialFeeds: map[string]*feed.Feed{},
		inputFeeds: []*list.InputFeed{
			{Name: "Feed 1", URL: "http://example.com/feed1", Type: "xml"},
			{Name: "Feed 2", URL: "http://example.com/feed2", Type: "xml"},
		},
		expectedFeeds: map[string]*feed.Feed{
			feed.UID("http://example.com/feed1"): {Name: "Feed 1", URL: "http://example.com/feed1", Type: "xml"},
			feed.UID("http://example.com/feed2"): {Name: "Feed 2", URL: "http://example.com/feed2", Type: "xml"},
		},
	}, {
		desc: "list has 2 feeds, and inputFields adds a new feed, updates an existing one and omits one existing feed",
		initialFeeds: map[string]*feed.Feed{
			feed.UID("http://example.com/feed1"): {Name: "Feed 1", URL: "http://example.com/feed1", Type: "xml"},
			feed.UID("http://example.com/feed2"): {Name: "Feed 2", URL: "http://example.com/feed2", Type: "xml"},
		},
		inputFeeds: []*list.InputFeed{
			{Name: "Feed 1 - Updated", URL: "http://example.com/feed1", Type: "html", Params: "abc"},
			{Name: "Feed 3", URL: "http://example.com/feed3", Type: "xml"},
		},
		expectedFeeds: map[string]*feed.Feed{
			feed.UID("http://example.com/feed1"): {Name: "Feed 1 - Updated", URL: "http://example.com/feed1", Type: "html", Params: "abc"},
			feed.UID("http://example.com/feed3"): {Name: "Feed 3", URL: "http://example.com/feed3", Type: "xml"},
		},
	}, {
		desc: "list has 2 feeds, and inputFields has no feeds",
		initialFeeds: map[string]*feed.Feed{
			feed.UID("http://example.com/feed1"): {Name: "Feed 1", URL: "http://example.com/feed1", Type: "xml"},
			feed.UID("http://example.com/feed2"): {Name: "Feed 2", URL: "http://example.com/feed2", Type: "xml"},
		},
		inputFeeds:    []*list.InputFeed{},
		expectedFeeds: map[string]*feed.Feed{},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			l := mem.NewList(mem.ListParams{})
			mem.SetFeedsMap(l, test.initialFeeds)
			l.LoadFeeds(test.inputFeeds)
			actualFeeds := mem.FeedsMap(l)
			if len(actualFeeds) != len(test.expectedFeeds) {
				t.Fatalf("expected %d feeds, got %d", len(test.expectedFeeds), len(actualFeeds))
			}
			for key, expectedFeed := range test.expectedFeeds {
				actualFeed, ok := actualFeeds[key]
				if !ok {
					t.Fatalf("expected feed %s to be present", key)
				}
				checkFeed(t, *actualFeed, *expectedFeed)
			}
		})
	}
}
