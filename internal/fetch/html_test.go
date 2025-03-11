package fetch

import (
	"testing"

	"github.com/alnvdl/varys/internal/feed"
)

func TestParseHTML(t *testing.T) {
	// Validating flawed to-UTF-8 conversion or invalid HTML with bad content
	// is apparently impossible with the libraries we are using, so we omit
	// these tests.
	tests := []struct {
		desc     string
		html     string
		params   any
		expected []feed.RawItem
		err      string
	}{{
		desc:   "error: corrupted params",
		html:   `<html><body><div class="other-container"></div></body></html>`,
		params: `{`,
		err:    "cannot parse HTML feed params: cannot unmarshal: json: cannot unmarshal string into Go value of type fetch.htmlParams",
	}, {
		desc: "error: container tag cannot be empty",
		html: `<html><body>
			<div class="target-container"></div>
		</body></html>`,
		params: map[string]any{
			"container_tag":    "",
			"container_attrs":  map[string]string{"class": "target-container"},
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{"https://example.com"},
		},
		err: "cannot parse HTML feed params: cannot validate: container_tag cannot be empty",
	}, {
		desc: "error: title position cannot be negative",
		html: `<html><body>
			<div class="target-container"></div>
		</body></html>`,
		params: map[string]any{
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"title_pos":        -1,
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{"https://example.com"},
		},
		err: "cannot parse HTML feed params: cannot validate: title_pos cannot be negative",
	}, {
		desc: "error: base URL cannot be empty",
		html: `<html><body>
			<div class="target-container"></div>
		</body></html>`,
		params: map[string]any{
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"base_url":         "",
			"allowed_prefixes": []string{"https://example.com"},
		},
		err: "cannot parse HTML feed params: cannot validate: base_url cannot be empty",
	}, {
		desc: "error: allowed prefixes cannot be empty",
		html: `<html><body>
			<div class="target-container"></div>
		</body></html>`,
		params: map[string]any{
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{},
		},
		err: "cannot parse HTML feed params: cannot validate: allowed_prefixes cannot be empty",
	}, {
		desc: "error: unknown encoding",
		html: `<html><body>
			<div class="other-container">
		</div></body></html>`,
		params: map[string]any{
			"encoding":         "what",
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{"https://example.com"},
		},
		err: "cannot find encoding: what",
	}, {
		desc: "success: HTML without the given container due to a mismatch on containerAttrs",
		html: `<html><body>
			<div class="other-container">
		</div></body></html>`,
		params: map[string]any{
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{"https://example.com"},
		},
		expected: nil,
	}, {
		desc: "success: HTML with the given container matching containerAttrs, but without any valid candidate items",
		html: `<html><body>
			<div class="target-container"></div>
		</body></html>`,
		params: map[string]any{
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{"https://example.com"},
		},
		expected: nil,
	}, {
		desc: "success: HTML with the given container matching containerAttrs with some candidate items in it",
		html: `<html><body>
			<div class="target-container">
				<a href="/url1">Title 1</a>
				<a href="https://example.com/url1"><span>Subtitle 1</span></a>
				<a href="/url2">Title 2<img alt="something" src="https://example.com/static/image.png" /></a>
				<a>Not really a valid item</a>
				<a href="http://example.com/url1">Disallowed prefix</a>
				<a href=":ohno">Bad URL</a>
			</div>
		</body></html>`,
		params: map[string]any{
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"title_pos":        0,
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{"https://example.com"},
		},
		expected: []feed.RawItem{{
			URL:     "https://example.com/url1",
			Title:   "Title 1",
			Content: "Title 1<br/>Subtitle 1",
		}, {
			URL:     "https://example.com/url2",
			Title:   "Title 2",
			Content: `Title 2<br/><img src="https://example.com/static/image.png"/>`,
		}},
	}, {
		desc: "success: there are no parts in the content and title_pos is beyond the number of parts",
		html: `<html><body>
			<div class="target-container">
				<a href="/url1">Title 1</a>
				<a href="/url2"></a>
			</div>
		</body></html>`,
		params: map[string]any{
			"container_tag":    "div",
			"container_attrs":  map[string]string{"class": "target-container"},
			"title_pos":        3,
			"base_url":         "https://example.com",
			"allowed_prefixes": []string{"https://example.com"},
		},
		expected: []feed.RawItem{{
			URL:     "https://example.com/url1",
			Title:   "Title 1",
			Content: "Title 1",
		}, {
			URL:     "https://example.com/url2",
			Title:   "Unknown title",
			Content: "",
		}},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			rawItems, err := parseHTML([]byte(test.html), test.params)
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
				if len(rawItems) != len(test.expected) {
					t.Fatalf("expected %d items, got %d", len(test.expected), len(rawItems))
				}
				for i, item := range rawItems {
					if item.URL != test.expected[i].URL ||
						item.Title != test.expected[i].Title ||
						item.Content != test.expected[i].Content {
						t.Errorf("expected item %#v, got %#v", test.expected[i], item)
					}
				}
			}
		})
	}
}
