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
