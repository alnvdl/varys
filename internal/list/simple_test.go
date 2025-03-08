package list_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"testing"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/fetch"
	"github.com/alnvdl/varys/internal/list"
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
		if item.Title != want.Items[i].Title {
			t.Errorf("expected item title %s, got %s", want.Items[i].Title, item.Title)
		}
		if item.URL != want.Items[i].URL {
			t.Errorf("expected item URL %s, got %s", want.Items[i].URL, item.URL)
		}
	}
}

func TestFeedListSave(t *testing.T) {
	feeds := map[string]*feed.Feed{
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
	}

	// Save the l to a buffer in JSON.
	l := list.NewSimple(list.SimpleParams{})
	list.SetFeeds(l, feeds)
	var buf bytes.Buffer
	err := l.Save(&buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Load the list from the buffer.
	var data list.SerializedList
	err = json.Unmarshal(buf.Bytes(), &data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(data.Feeds) != len(feeds) {
		t.Fatalf("expected %d feeds, got %d", len(feeds), len(data.Feeds))
	}
	for key, f := range feeds {
		savedFeed, ok := data.Feeds[key]
		if !ok {
			t.Fatalf("expected feed %s to be present", key)
		}
		checkFeed(t, *savedFeed, *f)
	}
}

type errorWriter struct{}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated write error")
}

func TestListSaveError(t *testing.T) {
	l := list.NewSimple(list.SimpleParams{})
	list.SetFeeds(l, make(map[string]*feed.Feed))

	err := l.Save(&errorWriter{})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	expectedErr := "cannot serialize feed list: simulated write error"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err.Error())
	}
}

func TestFeedListLoad(t *testing.T) {
	feeds := map[string]*feed.Feed{
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
	}

	// Serialize the feeds to JSON.
	var buf bytes.Buffer
	var data list.SerializedList
	data.Feeds = feeds
	err := json.NewEncoder(&buf).Encode(&data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Load the feeds from the JSON.
	loadedList := list.NewSimple(list.SimpleParams{})
	err = loadedList.Load(&buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	loadedFeeds := list.Feeds(loadedList)
	if len(loadedFeeds) != len(feeds) {
		t.Fatalf("expected %d feeds, got %d", len(feeds), len(loadedFeeds))
	}

	for key, f := range feeds {
		loadedFeed, ok := loadedFeeds[key]
		if !ok {
			t.Fatalf("expected feed %s to be present", key)
		}
		checkFeed(t, *loadedFeed, *f)
	}
}

func TestFeedListLoadError(t *testing.T) {
	corruptedJSON := `{"feeds": {"feed1":`

	list := list.NewSimple(list.SimpleParams{})
	err := list.Load(bytes.NewBufferString(corruptedJSON))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	expectedErr := "cannot deserialize feed list: unexpected EOF"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err.Error())
	}
}

func TestListSummary(t *testing.T) {
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
			l := list.NewSimple(list.SimpleParams{})
			list.SetFeeds(l, test.feeds)
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
			Items: []*feed.ItemSummary{
				{UID: "item1", FeedUID: "feed1", FeedName: "Feed 1", URL: "http://example.com/item1", Title: "Item 1"},
			},
		},
	}, {
		desc: "feed is the all feed compiled from two other existing feeds",
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
		},
		uid: "all",
		want: &feed.FeedSummary{
			UID:       "all",
			Name:      "All",
			ItemCount: 2,
			Items: []*feed.ItemSummary{
				{UID: "item1", FeedUID: "feed1", FeedName: "Feed 1", URL: "http://example.com/item1", Title: "Item 1"},
				{UID: "item2", FeedUID: "feed2", FeedName: "Feed 2", URL: "http://example.com/item2", Title: "Item 2"},
			},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			l := list.NewSimple(list.SimpleParams{})
			list.SetFeeds(l, test.feeds)
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
			l := list.NewSimple(list.SimpleParams{})
			list.SetFeeds(l, test.feeds)
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
	now := timeutil.Now()

	tests := []struct {
		desc            string
		feeds           map[string]*feed.Feed
		withItems       bool
		expectedSummary *feed.FeedSummary
	}{{
		desc:      "When the input feeds has no feeds",
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
		desc: "When all the input feeds are empty without items",
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
		desc: "When there are 10 items from 2 different feeds",
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
		desc: "When there are more than 100 items from 3 different feeds",
		feeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				URL:  "url1",
				Items: func() map[string]*feed.Item {
					items := make(map[string]*feed.Item)
					for i := range 100 {
						url := fmt.Sprintf("url%d", i*3+1)
						items[feed.UID(url)] = &feed.Item{
							RawItem:   feed.RawItem{URL: url, Title: fmt.Sprintf("Title %d", i*3+1)},
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
							RawItem:   feed.RawItem{URL: url, Title: fmt.Sprintf("Title %d", i*3+2)},
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
							RawItem:   feed.RawItem{URL: url, Title: fmt.Sprintf("Title %d", i*3+3)},
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
			summary := list.SimpleAllFeed(maps.Values(test.feeds), test.withItems)
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

func TestSimpleMarkRead(t *testing.T) {
	tests := []struct {
		desc           string
		feeds          map[string]*feed.Feed
		fuid           string
		iuid           string
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
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: false},
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: false},
					"item3": {RawItem: feed.RawItem{URL: "http://example.com/item3", Title: "Item 3"}, Read: false},
				},
			},
		},
		fuid:           "feed1",
		iuid:           "",
		expectedResult: true,
		expectedFeeds: map[string]*feed.Feed{
			"feed1": {
				Name: "Feed 1",
				Type: "xml",
				URL:  "http://example.com/feed1",
				Items: map[string]*feed.Item{
					"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}, Read: true},
					"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}, Read: true},
					"item3": {RawItem: feed.RawItem{URL: "http://example.com/item3", Title: "Item 3"}, Read: true},
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
			l := list.NewSimple(list.SimpleParams{})
			list.SetFeeds(l, test.feeds)
			result := l.MarkRead(test.fuid, test.iuid)
			if result != test.expectedResult {
				t.Errorf("expected result %v, got %v", test.expectedResult, result)
			}
			for key, expectedFeed := range test.expectedFeeds {
				actualFeed, ok := list.Feeds(l)[key]
				if !ok {
					t.Fatalf("expected feed %s to be present", key)
				}
				checkFeed(t, *actualFeed, *expectedFeed)
			}
		})
	}
}

func TestSimpleRefresh(t *testing.T) {
	now := timeutil.Now()

	mockFetcher := func(p fetch.FetchParams) ([]feed.RawItem, error) {
		switch p.URL {
		case "http://example.com/feed1":
			return []feed.RawItem{
				{URL: "http://example.com/item1", Title: "Item 1"},
			}, nil
		case "http://example.com/feed2":
			return []feed.RawItem{
				{URL: "http://example.com/item2", Title: "Item 2"},
			}, nil
		case "http://example.com/feed3":
			return nil, errors.New("oh no")
		default:
			return nil, errors.New("unknown feed URL")
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
						Timestamp: now},
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
						Timestamp: now},
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
						Timestamp: now},
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
			l := list.NewSimple(list.SimpleParams{
				Fetcher: mockFetcher,
			})
			list.SetFeeds(l, test.initialFeeds)
			l.Refresh()
			for key, expectedFeed := range test.expectedFeeds {
				actualFeed, ok := list.Feeds(l)[key]
				if !ok {
					t.Fatalf("expected feed %s to be present", key)
				}
				checkFeed(t, *actualFeed, *expectedFeed)
			}
		})
	}
}

func TestSimpleLoadFeeds(t *testing.T) {
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
			l := list.NewSimple(list.SimpleParams{})
			list.SetFeeds(l, test.initialFeeds)
			l.LoadFeeds(test.inputFeeds)
			actualFeeds := list.Feeds(l)
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
