package fetch

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/alnvdl/varys/internal/feed"
	"golang.org/x/net/html/charset"
)

type RSS struct {
	Channel struct {
		Items []RSSItem `xml:"item"`
		Link  string    `xml:"link"`
	} `xml:"channel"`
	Items []RSSItem `xml:"item"`
}

type RSSItem struct {
	ID           string   `xml:"id"`
	GUID         string   `xml:"guid"`
	Link         string   `xml:"link"`
	Title        string   `xml:"title"`
	PubDate      string   `xml:"pubDate"`
	Date         string   `xml:"date"`
	Creator      []string `xml:"creator"`
	Authors      []string `xml:"author>name"`
	Encoded      string   `xml:"encoded"`
	Descriptions []string `xml:"description"`
}

type Atom struct {
	Entries []AtomEntry `xml:"entry"`
	Link    struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
}

type AtomEntry struct {
	ID    string `xml:"id"`
	GUID  string `xml:"guid"`
	Links []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
	Title     string   `xml:"title"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Authors   []string `xml:"author>name"`
	Content   string   `xml:"content"`
	Summary   string   `xml:"summary"`
}

type xmlParams struct {
	InferParagraphs bool `json:"infer_paragraphs"`
}

func (p *xmlParams) Validate() error {
	return nil
}

func tryParseFeed(data []byte, v any) error {
	r := bytes.NewReader(data)
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charset.NewReaderLabel
	return dec.Decode(&v)
}

func parseXML(data []byte, params any) ([]feed.RawItem, error) {
	var p xmlParams
	if err := feed.ParseParams(params, &p); err != nil {
		return nil, fmt.Errorf("cannot parse XML feed params: %v", err)
	}

	var feedItems []feed.RawItem

	rss := RSS{}
	rssErr := tryParseFeed(data, &rss)
	if rssErr == nil && (len(rss.Channel.Items) > 0 || len(rss.Items) > 0) {
		baseURL := absoluteURL(rss.Channel.Link)
		items := rss.Channel.Items
		if len(items) == 0 {
			items = rss.Items
		}
		for pos, item := range items {
			resolvedItemURL := resolveURL(item.Link, baseURL, nil)
			content := coalesce(item.Encoded, strings.Join(item.Descriptions, "\n"))
			if p.InferParagraphs {
				content = inferParagraphs(content)
			}
			feedItems = append(feedItems, feed.RawItem{
				URL:      urlToString(resolvedItemURL),
				Title:    strings.TrimSpace(item.Title),
				Authors:  strings.TrimSpace(strings.Join(append(item.Authors, item.Creator...), ", ")),
				Content:  silentlySanitizeHTML(content, resolvedItemURL),
				Position: pos,
			})
		}
	}

	atom := Atom{}
	atomErr := tryParseFeed(data, &atom)
	if atomErr == nil && len(atom.Entries) > 0 {
		baseURL := absoluteURL(atom.Link.Href)
		for pos, entry := range atom.Entries {
			// Atom may have multiple links for entries, prefer the most
			// logical one (with rel="self" or no rel attribute), or pick the
			// first one if none of the above are found.
			var itemURL string
			for _, link := range entry.Links {
				if link.Rel == "self" || link.Rel == "" {
					itemURL = link.Href
					break
				}
			}
			if itemURL == "" && len(entry.Links) > 0 {
				itemURL = entry.Links[0].Href
			}
			resolvedItemURL := resolveURL(itemURL, baseURL, nil)
			content := coalesce(entry.Content, entry.Summary)
			if p.InferParagraphs {
				content = inferParagraphs(content)
			}
			feedItems = append(feedItems, feed.RawItem{
				URL:      urlToString(resolvedItemURL),
				Title:    strings.TrimSpace(entry.Title),
				Authors:  strings.TrimSpace(strings.Join(entry.Authors, ", ")),
				Content:  silentlySanitizeHTML(content, resolvedItemURL),
				Position: pos,
			})
		}
	}

	err := errors.Join(rssErr, atomErr)
	if err != nil {
		err = fmt.Errorf("cannot parse XML as either RSS or Atom: %v", errors.Join(rssErr, atomErr))
	}
	return feedItems, err
}

// absoluteURL returns the URL for input if it is a valid absolute URL,
// otherwise it returns nil.
func absoluteURL(input string) *url.URL {
	u, err := url.Parse(input)
	if err != nil || !u.IsAbs() {
		return nil
	}
	return u

}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func inferParagraphs(input string) string {
	lines := strings.Split(input, "\n")
	paragraphs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			paragraphs = append(paragraphs, "<p>"+line+"</p>")
		}
	}
	return strings.Join(paragraphs, "")
}
