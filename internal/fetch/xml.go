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
	ID          string   `xml:"id"`
	GUID        string   `xml:"guid"`
	Link        string   `xml:"link"`
	Title       string   `xml:"title"`
	PubDate     string   `xml:"pubDate"`
	Date        string   `xml:"date"`
	Creator     []string `xml:"creator"`
	Authors     []string `xml:"author>name"`
	Encoded     string   `xml:"encoded"`
	Description string   `xml:"description"`
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

func tryParseFeed(data []byte, v any) error {
	r := bytes.NewReader(data)
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charset.NewReaderLabel
	return dec.Decode(&v)
}

func parseXML(data []byte, _ any) ([]feed.RawItem, error) {
	var feedItems []feed.RawItem

	rss := RSS{}
	rssErr := tryParseFeed(data, &rss)
	if rssErr == nil && (len(rss.Channel.Items) > 0 || len(rss.Items) > 0) {
		baseURL := link(rss.Channel.Link, "")
		items := rss.Channel.Items
		if len(items) == 0 {
			items = rss.Items
		}
		for pos, item := range items {
			feedItems = append(feedItems, feed.RawItem{
				URL:      link(item.Link, baseURL),
				Title:    strings.TrimSpace(item.Title),
				Authors:  strings.TrimSpace(strings.Join(append(item.Authors, item.Creator...), ", ")),
				Content:  silentlySanitizeHTML(coalesce(item.Encoded, item.Description)),
				Position: pos,
			})
		}
	}

	atom := Atom{}
	atomErr := tryParseFeed(data, &atom)
	if atomErr == nil && len(atom.Entries) > 0 {
		baseURL := link(atom.Link.Href, "")
		for pos, entry := range atom.Entries {
			var url string
			for _, link := range entry.Links {
				if link.Rel == "self" || link.Rel == "" {
					url = link.Href
					break
				}
			}
			if url == "" && len(entry.Links) > 0 {
				url = entry.Links[0].Href
			}

			feedItems = append(feedItems, feed.RawItem{
				URL:      link(url, baseURL),
				Title:    strings.TrimSpace(entry.Title),
				Authors:  strings.TrimSpace(strings.Join(entry.Authors, ", ")),
				Content:  silentlySanitizeHTML(coalesce(entry.Content, entry.Summary)),
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

func link(link, base string) string {
	if link == "" {
		return ""
	}
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}

	if u.IsAbs() && (u.Scheme == "http" || u.Scheme == "https") {
		return link
	} else if base != "" {
		baseURL, err := url.Parse(base)
		if err != nil {
			return ""
		}
		return baseURL.ResolveReference(u).String()
	}

	return ""
}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
