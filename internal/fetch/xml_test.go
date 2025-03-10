package fetch_test

import (
	"testing"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/fetch"
)

func TestParseXML(t *testing.T) {
	tests := []struct {
		desc string
		xml  string

		expected []feed.RawItem
		err      string
	}{{
		desc: "RSS with 0 items",
		xml: `
			<rss>
				<channel>
					<link>http://example.com</link>
				</channel>
			</rss>`,
		expected: []feed.RawItem{},
	}, {
		desc: "RSS with 1 item and relative item URLs",
		xml: `
			<rss>
				<channel>
					<link>http://example.com</link>
					<item>
						<title>Item 1</title>
						<link>content/item1</link>
						<pubDate>Sun, 02 Mar 2025 09:30:15 BRT</pubDate>
						<author>
							<name>Author 1</name>
						</author>
						<content:encoded>Content 1</content:encoded>
					</item>
				</channel>
			</rss>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/content/item1",
			Title:    "Item 1",
			Authors:  "Author 1",
			Content:  "Content 1",
			Position: 0,
		}},
	}, {
		desc: "RSS with 1 invalid item",
		xml: `
			<rss>
				<channel>
					<link>http://example.com</link>
					<item>
						<author>
							<name>Author 1</name>
						</author>
						<content:encoded>Content 1</content:encoded>
					</item>
				</channel>
			</rss>`,
		expected: []feed.RawItem{{
			Authors:  "Author 1",
			Content:  "Content 1",
			Position: 0,
		}},
	}, {
		desc: "RSS with 3 items",
		xml: `
			<rss>
				<channel>
					<link>http://example.com</link>
					<item>
						<title>Item 1</title>
						<link>http://example.com/item1</link>
						<dc:date>Mon, 02 Jan 2025 15:04:05 MST</dc:date>
						<dc:creator>Author 1</dc:creator>
						<content:encoded>Content 1</content:encoded>
					</item>
					<item>
						<title>Item 2</title>
						<link>http://example.com/item2</link>
						<dc:date>Tue, 03 Jan 2025 15:04:05 -0700</dc:date>
						<dc:creator>Author 2</dc:creator>
						<content:encoded>Content 2</content:encoded>
					</item>
					<item>
						<title>Item 3</title>
						<link>http://example.com/item3</link>
						<dc:date>Wed, 04 Jan 2025 15:04:05 +0000</dc:date>
						<dc:creator>Author 3</dc:creator>
						<dc:creator>Author 4</dc:creator>
						<content:encoded>Content 3</content:encoded>
					</item>
				</channel>
			</rss>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/item1",
			Title:    "Item 1",
			Authors:  "Author 1",
			Content:  "Content 1",
			Position: 0,
		}, {
			URL:      "http://example.com/item2",
			Title:    "Item 2",
			Authors:  "Author 2",
			Content:  "Content 2",
			Position: 1,
		}, {
			URL:      "http://example.com/item3",
			Title:    "Item 3",
			Authors:  "Author 3, Author 4",
			Content:  "Content 3",
			Position: 2,
		}},
	}, {
		desc: "Atom with 0 items",
		xml: `
			<feed>
				<link href="http://example.com"/>
			</feed>`,
		expected: []feed.RawItem{},
	}, {
		desc: "Atom with 1 item and relative item URLs",
		xml: `
					<feed>
						<link href="http://example.com"/>
						<entry>
							<title>Item 1</title>
							<link href="content/item1"/>
							<published>2025-01-02T15:04:05Z</published>
							<author>
								<name>Author 1</name>
							</author>
							<content>Content 1</content>
						</entry>
					</feed>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/content/item1",
			Title:    "Item 1",
			Authors:  "Author 1",
			Content:  "Content 1",
			Position: 0,
		}},
	}, {
		desc: "Atom with 3 items",
		xml: `
			<feed>
				<link href="http://example.com"/>
				<entry>
					<title>Item 1</title>
					<link href="http://example.com/item1"/>
					<link rel="related" href="http://example.com/item1_image"/>
					<published>2025-01-02T15:04:05Z</published>
					<author>
						<name>Author 1</name>
					</author>
					<content>Content 1</content>
				</entry>
				<entry>
					<title>Item 2</title>
					<link href="http://example.com/item2"/>
					<published>2025-01-03T15:04:05-07:00</published>
					<author>
						<name>Author 2</name>
					</author>
					<content>Content 2</content>
				</entry>
				<entry>
					<title>Item 3</title>
					<link href="http://example.com/item3"/>
					<published>2025-01-04T15:04:05+00:00</published>
					<author>
						<name>Author 3</name>
					</author>
					<content>Content 3</content>
				</entry>
			</feed>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/item1",
			Title:    "Item 1",
			Authors:  "Author 1",
			Content:  "Content 1",
			Position: 0,
		}, {
			URL:      "http://example.com/item2",
			Title:    "Item 2",
			Authors:  "Author 2",
			Content:  "Content 2",
			Position: 1,
		}, {
			URL:      "http://example.com/item3",
			Title:    "Item 3",
			Authors:  "Author 3",
			Content:  "Content 3",
			Position: 2,
		}},
	}, {
		desc: "RSS with mixed valid and invalid items",
		xml: `
			<rss>
				<channel>
					<link>http://example.com</link>
					<item>
						<title>Item 1</title>
						<link>http://example.com/item1</link>
						<pubDate>Mon, 02 Jan 2025 15:04:05 MST</pubDate>
						<dc:creator>Author 1</dc:creator>
						<dc:creator>Author 2</dc:creator>
						<content:encoded><![CDATA[
							<div>should be in output</div>
							<script>should be removed</script>
							<b>Content 1</b>]]>
						</content:encoded>
					</item>
					<entry>
						<title></title>
						<link></link>
						<pubDate></pubDate>
						<dc:creator></dc:creator>
						<content:encoded></content:encoded>
					</entry>
					<item>
						<title>Item 2</title>
						<link>http://example.com/item2</link>
						<pubDate>Mon, 02 Jan 2025 15:04:05 MST</pubDate>
						<dc:creator>Author 3</dc:creator>
						<content:encoded>Content 2</content:encoded>
					</item>
				</channel>
			</rss>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/item1",
			Title:    "Item 1",
			Authors:  "Author 1, Author 2",
			Content:  "<div>should be in output</div><b>Content 1</b>",
			Position: 0,
		}, {
			URL:      "http://example.com/item2",
			Title:    "Item 2",
			Authors:  "Author 3",
			Content:  "Content 2",
			Position: 1,
		}},
	}, {
		desc: "Atom with mixed valid and invalid items",
		xml: `
			<feed>
				<link href="http://example.com"/>
				<entry>
					<title>Item 1</title>
					<link href="http://example.com/item1"/>
					<published>2025-01-02T15:04:05Z</published>
					<author>
						<name>Author 1</name>
					</author>
					<author>
						<name>Author 2</name>
					</author>
					<content>Content 1</content>
				</entry>
				<entry>
					<title></title>
					<link></link>
					<published></published>
					<author>
						<name></name>
					</author>
					<content></content>
				</entry>
				<entry>
					<title>Item 2</title>
					<link href="http://example.com/item2"/>
					<published>2025-01-03T15:04:05-07:00</published>
					<author>
						<name>Author 3</name>
					</author>
					<content>Content 2</content>
				</entry>
			</feed>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/item1",
			Title:    "Item 1",
			Authors:  "Author 1, Author 2",
			Content:  "Content 1",
			Position: 0,
		}, {
			// Empty item.
			Position: 1,
		}, {
			URL:      "http://example.com/item2",
			Title:    "Item 2",
			Authors:  "Author 3",
			Content:  "Content 2",
			Position: 2,
		}},
	}, {
		desc: "RSS with HTML in fields",
		xml: `
			<rss>
				<channel>
					<link>http://example.com</link>
					<item>
						<title><![CDATA[<script>actually ok</script>Item 1]]></title>
						<link>http://example.com/item1<script>actually ignored</script></link>
						<dc:creator>
							<![CDATA[<script>actually ok</script>Author 1]]>
						</dc:creator>
						<content:encoded><![CDATA[<div>should be in output</div><script>should be removed</script><b>Content 1</b>]]></content:encoded>
					</item>
				</channel>
			</rss>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/item1",
			Title:    "<script>actually ok</script>Item 1",
			Authors:  "<script>actually ok</script>Author 1",
			Content:  "<div>should be in output</div><b>Content 1</b>",
			Position: 0,
		}},
	}, {
		desc: "Atom with HTML in fields",
		xml: `
			<feed>
				<link href="http://example.com"/>
				<entry>
					<title>Item 1<script>actually ignored</script></title>
					<link href="/what&lt;div&gt;should just be text&lt;/div&gt;something"/>
					<author>
						<name><![CDATA[<script>actually ok</script>Author 1]]></name>
					</author>
					<content><![CDATA[<div>should be in output</div><script>should be removed</script><b>Content 1</b>]]></content>
				</entry>
			</feed>`,
		expected: []feed.RawItem{{
			URL:      "http://example.com/what%3Cdiv%3Eshould%20just%20be%20text%3C/div%3Esomething",
			Title:    "Item 1",
			Authors:  "<script>actually ok</script>Author 1",
			Content:  "<div>should be in output</div><b>Content 1</b>",
			Position: 0,
		}},
	}, {
		desc:     "malformed XML",
		xml:      `<>`,
		expected: []feed.RawItem{},
		err:      "cannot parse XML as either RSS or Atom: XML syntax error on line 1: expected element name after <\nXML syntax error on line 1: expected element name after <",
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			data := []byte(test.xml)
			items, err := fetch.ParseXML(data, nil)
			if err != nil {
				if test.err == "" {
					t.Errorf("unexpected error: %v", err)
				} else if err.Error() != test.err {
					t.Errorf("expected error %q, got %q", test.err, err.Error())
				}
				return
			}
			if len(items) != len(test.expected) {
				t.Errorf("expected %d items, got %d", len(test.expected), len(items))
				return
			}
			for i, item := range items {
				if item != test.expected[i] {
					t.Errorf("expected item %#v, got %#v", test.expected[i], item)
				}
			}
		})
	}
}
