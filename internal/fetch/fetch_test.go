package fetch_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/fetch"
)

func TestFetch(t *testing.T) {
	tests := []struct {
		desc          string
		serverData    string
		feedURL       string
		feedType      string
		expectedItems []feed.RawItem
		expectedError string
	}{{
		desc: "a valid feed with one item",
		serverData: `
				<rss>
					<channel>
						<link>http://example.com</link>
						<item>
							<title>Item 1</title>
							<link>http://example.com/item1</link>
							<pubDate>Mon, 02 Jan 2025 15:04:05 MST</pubDate>
							<dc:creator>Author 1</dc:creator>
							<content:encoded>Content 1</content:encoded>
						</item>
					</channel>
				</rss>`,
		feedType: "xml",
		expectedItems: []feed.RawItem{
			{
				URL:     "http://example.com/item1",
				Title:   "Item 1",
				Authors: "Author 1",
				Content: "Content 1",
			},
		},
		expectedError: "",
	}, {
		desc: "invalid RSS feed",
		serverData: `
				<rss
					<channel
						<link>http://example.com</link
					</channel>
				</rss>`,
		feedType:      "xml",
		expectedItems: []feed.RawItem{},
		expectedError: "cannot parse XML feed: cannot parse XML as either RSS or Atom: XML syntax error on line 3: expected attribute name in element\nXML syntax error on line 3: expected attribute name in element",
	}, {
		desc: "invalid Atom feed",
		serverData: `
				<feed
					<link href="http://example.com"/
				</feed>`,
		feedType:      "xml",
		expectedItems: []feed.RawItem{},
		expectedError: "cannot parse XML feed: cannot parse XML as either RSS or Atom: XML syntax error on line 3: expected attribute name in element\nXML syntax error on line 3: expected attribute name in element",
	}, {
		desc: "a valid feed with more than one item and one invalid/empty item",
		serverData: `
				<feed>
					<link href="http://example.com"/>
					<entry>
						<title>Item 1</title>
						<link href="/item1"/>
						<published>2025-01-02T15:04:05Z</published>
						<author>
							<name>Author 1</name>
						</author>
						<content>Content 1</content>
					</entry>
					<entry>
						<title></title>
						<published></published>
						<content></content>
					</entry>
					<entry>
						<title>Item 2</title>
						<link href="/item2"/>
						<published>2025-01-02T15:04:05Z</published>
						<author>
							<name>Author 2</name>
						</author>
						<content>Content 2</content>
					</entry>
					<entry>
						<title>Item 3</title>
						<link href="/item3"/>
						<published>2025-01-02T15:04:05Z</published>
						<author>
							<name>Author 3</name>
						</author>
						<content>Content 3</content>
					</entry>
				</feed>`,
		feedType: "xml",
		expectedItems: []feed.RawItem{
			{
				URL:     "http://example.com/item1",
				Title:   "Item 1",
				Authors: "Author 1",
				Content: "Content 1",
			},
			{},
			{
				URL:     "http://example.com/item2",
				Title:   "Item 2",
				Authors: "Author 2",
				Content: "Content 2",
			},
			{
				URL:     "http://example.com/item3",
				Title:   "Item 3",
				Authors: "Author 3",
				Content: "Content 3",
			},
		},
		expectedError: "",
	}, {
		desc:          "HTTP error",
		serverData:    "",
		feedURL:       "http://127.0.0.1:12345/feed.xml",
		feedType:      "rss",
		expectedItems: []feed.RawItem{},
		expectedError: `cannot make request: Get "http://127.0.0.1:12345/feed.xml": dial tcp 127.0.0.1:12345: connect: connection refused`,
	}, {
		desc:          "unsupported feed type",
		serverData:    "",
		feedURL:       "http://example.com/feed.banana",
		feedType:      "banana",
		expectedItems: []feed.RawItem{},
		expectedError: "unsupported feed type: banana",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			feedURL := test.feedURL
			if feedURL == "" {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(test.serverData))
				}))
				defer server.Close()
				feedURL = server.URL
			}

			items, err := fetch.Fetch(fetch.FetchParams{
				URL:      feedURL,
				FeedName: test.desc,
				FeedType: test.feedType,
			})

			if (test.expectedError != "" && err == nil) || (err != nil && err.Error() != test.expectedError) {
				t.Errorf("expected error %v, got %v", test.expectedError, err)
			}

			if len(items) != len(test.expectedItems) {
				t.Errorf("expected %d items, got %d", len(test.expectedItems), len(items))
			}

			for i, expectedItem := range test.expectedItems {
				if items[i] != expectedItem {
					t.Errorf("expected item %#v, got %#v", expectedItem, items[i])
				}
			}
		})
	}
}
