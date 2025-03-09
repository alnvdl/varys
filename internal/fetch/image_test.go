package fetch_test

import (
	"testing"
	"time"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/fetch"
)

func TestParseImage(t *testing.T) {
	tests := []struct {
		desc     string
		data     []byte
		params   map[string]any
		expected func(time.Time) []feed.RawItem
		err      string
	}{{
		desc: "error: mime_type cannot be empty",

		data: []byte{1, 2, 3},
		params: map[string]any{
			"url":   "https://example.com/image",
			"title": "Example Image",
		},
		err: "cannot parse image params: cannot validate: mime_type cannot be empty",
	}, {
		desc: "error: url cannot be empty",
		data: []byte{1, 2, 3},
		params: map[string]any{
			"mime_type": "image/png",
			"title":     "Example Image",
		},
		err: "cannot parse image params: cannot validate: url cannot be empty",
	}, {
		desc: "error: title cannot be empty",
		data: []byte{1, 2, 3},
		params: map[string]any{
			"mime_type": "image/png",
			"url":       "https://example.com/image",
		},
		err: "cannot parse image params: cannot validate: title cannot be empty",
	}, {
		desc: "success: valid image params",
		data: []byte{1, 2, 3},
		params: map[string]any{
			"mime_type": "image/png",
			"url":       "https://example.com/image",
			"title":     "Example Image",
		},
		expected: func(now time.Time) []feed.RawItem {
			date := now.Format("2006-01-02 15:04:05 UTC")
			return []feed.RawItem{{
				URL:     "https://example.com/image#039058c6f2c0cb492c533b0a4d14ef77cc0f78abccced5287d84a1a2011cfb81",
				Title:   "Example Image - " + date,
				Content: `<img src="data:image/png;base64,AQID"/>`,
			}}
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			now := time.Now()
			rawItems, err := fetch.ParseImage(test.data, test.params)
			if test.err != "" {
				if err == nil || err.Error() != test.err {
					t.Fatalf("expected error: %v, got: %v", test.err, err)
				}
				if rawItems != nil {
					t.Fatalf("expected no items, got %d", len(rawItems))
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				expectedItems := test.expected(now)
				if len(rawItems) != len(expectedItems) {
					t.Fatalf("expected %d items, got %d", len(expectedItems), len(rawItems))
				}
				for i, item := range rawItems {
					if item.URL != expectedItems[i].URL ||
						item.Title != expectedItems[i].Title ||
						item.Content != expectedItems[i].Content {
						t.Errorf("expected item %#v, got %#v", expectedItems[i], item)
					}
				}
			}
		})
	}
}
