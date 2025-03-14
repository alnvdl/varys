package feed

import (
	"testing"
)

func TestItemRefresh(t *testing.T) {
	tests := []struct {
		desc           string
		initialItem    Item
		rawItem        RawItem
		expectedItem   Item
		expectedResult bool
	}{{
		desc:        "raw item was empty",
		initialItem: Item{},
		rawItem:     RawItem{URL: "url1", Title: "New Title 1", Position: 2},
		expectedItem: Item{
			// Position is set because the URL was empty.
			RawItem: RawItem{URL: "url1", Title: "New Title 1", Position: 2},
		},
		expectedResult: true,
	}, {
		desc: "raw item changed",
		initialItem: Item{
			RawItem: RawItem{URL: "url1", Title: "Title 1", Position: 1, Authors: "Author 1", Content: "Content 1"},
		},
		rawItem: RawItem{URL: "url1", Title: "Updated Title 1", Position: 2, Authors: "Author 2", Content: "Content 2"},
		expectedItem: Item{
			// Position is unchanged because URL was not empty.
			RawItem: RawItem{URL: "url1", Title: "Updated Title 1", Position: 1, Authors: "Author 2", Content: "Content 2"},
		},
		expectedResult: true,
	}, {
		desc: "raw item did not change",
		initialItem: Item{
			RawItem: RawItem{URL: "url1", Title: "Title 1", Position: 1},
		},
		rawItem: RawItem{URL: "url1", Title: "Title 1", Position: 2},
		expectedItem: Item{
			RawItem: RawItem{URL: "url1", Title: "Title 1", Position: 1},
		},
		expectedResult: false,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			item := test.initialItem
			result := item.Refresh(test.rawItem)
			if result != test.expectedResult {
				t.Errorf("expected result %v, got %v", test.expectedResult, result)
			}
			if item != test.expectedItem {
				t.Errorf("expected item %#v, got %#v", test.expectedItem, item)
			}
		})
	}
}

func TestItemIsValid(t *testing.T) {
	tests := []struct {
		desc     string
		rawItem  RawItem
		expected bool
	}{{
		desc:     "valid item with URL and title",
		rawItem:  RawItem{URL: "url1", Title: "Title 1"},
		expected: true,
	}, {
		desc:     "invalid item with empty URL",
		rawItem:  RawItem{URL: "", Title: "Title 1"},
		expected: false,
	}, {
		desc:     "invalid item with empty title",
		rawItem:  RawItem{URL: "url1", Title: ""},
		expected: false,
	}, {
		desc:     "invalid item with empty URL and title",
		rawItem:  RawItem{URL: "", Title: ""},
		expected: false,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := test.rawItem.IsValid()
			if result != test.expected {
				t.Errorf("expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestItemUID(t *testing.T) {
	tests := []struct {
		desc     string
		rawItem  RawItem
		expected string
	}{{
		desc:     "valid item with URL",
		rawItem:  RawItem{URL: "url1", Title: "Title 1"},
		expected: UID("url1"),
	}, {
		desc:     "invalid item with empty URL",
		rawItem:  RawItem{URL: "", Title: "Title 1"},
		expected: "",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := test.rawItem.UID()
			if result != test.expected {
				t.Errorf("expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestItemSummary(t *testing.T) {
	tests := []struct {
		desc           string
		item           Item
		feed           Feed
		includeContent bool
		expected       ItemSummary
	}{{
		desc: "item with regular feed with content",
		item: Item{
			RawItem:   RawItem{URL: "url1", Title: "Title 1", Authors: "Author 1", Content: "Content 1"},
			FeedUID:   "feed1",
			Timestamp: 1234567890,
			Read:      true,
		},
		feed: Feed{
			Name: "Feed 1",
			URL:  "feed1",
		},
		includeContent: true,
		expected: ItemSummary{
			UID:       UID("url1"),
			FeedUID:   UID("feed1"),
			FeedName:  "Feed 1",
			URL:       "url1",
			Title:     "Title 1",
			Timestamp: 1234567890,
			Authors:   "Author 1",
			Read:      true,
			Content:   "Content 1",
		},
	}, {
		desc: "item with regular feed without content",
		item: Item{
			RawItem:   RawItem{URL: "url1", Title: "Title 1", Authors: "Author 1", Content: "Content 1"},
			FeedUID:   "feed1",
			Timestamp: 1234567890,
			Read:      true,
		},
		feed: Feed{
			Name: "Feed 1",
			URL:  "feed1",
		},
		includeContent: false,
		expected: ItemSummary{
			UID:       UID("url1"),
			FeedUID:   UID("feed1"),
			FeedName:  "Feed 1",
			URL:       "url1",
			Title:     "Title 1",
			Timestamp: 1234567890,
			Authors:   "Author 1",
			Read:      true,
		},
	}, {
		desc: "item with virtual feed named 'all' with content",
		item: Item{
			RawItem:   RawItem{URL: "url1", Title: "Title 1", Authors: "Author 1", Content: "Content 1"},
			FeedUID:   "all",
			Timestamp: 1234567890,
			Read:      true,
		},
		feed: Feed{
			Name: "all",
		},
		includeContent: true,
		expected: ItemSummary{
			UID:       UID("url1"),
			FeedUID:   "all",
			FeedName:  "all",
			URL:       "url1",
			Title:     "Title 1",
			Timestamp: 1234567890,
			Authors:   "Author 1",
			Read:      true,
			Content:   "Content 1",
		},
	}, {
		desc: "item with virtual feed named 'all' without content",
		item: Item{
			RawItem:   RawItem{URL: "url1", Title: "Title 1", Authors: "Author 1", Content: "Content 1"},
			FeedUID:   "all",
			Timestamp: 1234567890,
			Read:      true,
		},
		feed: Feed{
			Name: "All",
		},
		includeContent: false,
		expected: ItemSummary{
			UID:       UID("url1"),
			FeedUID:   "all",
			FeedName:  "All",
			URL:       "url1",
			Title:     "Title 1",
			Timestamp: 1234567890,
			Authors:   "Author 1",
			Read:      true,
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := test.item.Summary(&test.feed, test.includeContent)
			if *result != test.expected {
				t.Errorf("expected %#v, got %#v", test.expected, result)
			}
		})
	}
}
