package fetch

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/alnvdl/varys/internal/feed"
)

type RSS struct {
	Channel struct {
		Items []RSSItem `xml:"item"`
		Link  string    `xml:"link"`
	} `xml:"channel"`
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
	ID   string `xml:"id"`
	GUID string `xml:"guid"`
	Link struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Title     string   `xml:"title"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Authors   []string `xml:"author>name"`
	Content   string   `xml:"content"`
	Summary   string   `xml:"summary"`
}

func parseXML(data []byte, _ map[string]string) ([]feed.RawItem, error) {
	var feedItems []feed.RawItem

	rss := RSS{}
	atom := Atom{}

	rssErr := xml.Unmarshal(data, &rss)
	if rssErr == nil && len(rss.Channel.Items) > 0 {
		baseURL := link(rss.Channel.Link, "")
		for _, item := range rss.Channel.Items {
			feedItems = append(feedItems, feed.RawItem{
				URL:     silentlySanitizePlainText(link(item.Link, baseURL)),
				Title:   silentlySanitizePlainText(item.Title),
				Authors: silentlySanitizePlainText(strings.Join(append(item.Authors, item.Creator...), ", ")),
				Content: silentlySanitizeHTML(coalesce(item.Encoded, item.Description)),
			})
		}
	}

	atomErr := xml.Unmarshal(data, &atom)
	if atomErr == nil && len(atom.Entries) > 0 {
		baseURL := link(atom.Link.Href, "")
		for _, entry := range atom.Entries {
			feedItems = append(feedItems, feed.RawItem{
				URL:     silentlySanitizePlainText(link(entry.Link.Href, baseURL)),
				Title:   silentlySanitizePlainText(entry.Title),
				Authors: silentlySanitizePlainText(strings.Join(entry.Authors, ", ")),
				Content: silentlySanitizeHTML(coalesce(entry.Content, entry.Summary)),
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
