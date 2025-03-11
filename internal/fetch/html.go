package fetch

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/alnvdl/varys/internal/feed"
	"golang.org/x/net/html"
)

// htmlParams defines the parameters for parseHTML.
type htmlParams struct {
	Encoding        string            `json:"encoding"`
	ContainerTag    string            `json:"container_tag"`
	ContainerAttrs  map[string]string `json:"container_attrs"`
	TitlePos        int               `json:"title_pos"`
	BaseURL         string            `json:"base_url"`
	AllowedPrefixes []string          `json:"allowed_prefixes"`
}

func (p *htmlParams) validate() error {
	if p.ContainerTag == "" {
		return fmt.Errorf("container_tag cannot be empty")
	}
	if p.TitlePos < 0 {
		return fmt.Errorf("title_pos cannot be negative")
	}
	if p.BaseURL == "" {
		return fmt.Errorf("base_url cannot be empty")
	}
	if _, err := url.Parse(p.BaseURL); err != nil {
		return fmt.Errorf("cannot parse base_url: %v", err)
	}
	if len(p.AllowedPrefixes) == 0 {
		return fmt.Errorf("allowed_prefixes cannot be empty")
	}
	return nil
}

// candidateItem is a candidate feed item extracted from HTML content.
type candidateItem struct {
	url string

	// parts are relevant segments from inside the candidate item. They are
	// usually extracted from text nodes and img tags, and have to be
	// interpreted by the caller (e.g., to determine the title).
	parts []string

	position int
}

func (c *candidateItem) merge(other *candidateItem) {
	c.parts = append(c.parts, other.parts...)
}

// extractCandidateItem extracts a feed item from an HTML node. The node is
// expected to be an anchor tag, or a nil item will be returned. The function
// walks the node and its children to extract the candidate item. Nil will be
// returned if resolveURL returns an empty URL.
func extractCandidateItem(anchor *html.Node, baseURL *url.URL, allowedPrefixes []string) *candidateItem {
	if anchor.Type != html.ElementNode || anchor.Data != "a" {
		return nil
	}

	var url string
	for _, attr := range anchor.Attr {
		if attr.Key == "href" {
			url = urlToString(resolveURL(attr.Val, baseURL, allowedPrefixes))
		}
	}
	if url == "" {
		return nil
	}

	ci := &candidateItem{url: url}
	var extractContent func(*html.Node)
	extractContent = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "img" {
			for _, attr := range n.Attr {
				if attr.Key == "src" || attr.Key == "data-src" {
					imgSrc := resolveURL(attr.Val, baseURL, nil)
					imgNode := &html.Node{
						Type: html.ElementNode,
						Data: "img",
						Attr: []html.Attribute{{Key: "src", Val: imgSrc.String()}},
					}
					var buf bytes.Buffer
					html.Render(&buf, imgNode)
					ci.parts = append(ci.parts, buf.String())
					break
				}
			}
		}
		// We checked the allowed tags to prevent useless content (e.g., a
		// "style" node) from being picked up.
		if n.Type == html.TextNode && n.Parent != nil && defaultAllowedTags[n.Parent.Data] {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				ci.parts = append(ci.parts, text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractContent(c)
		}
	}
	extractContent(anchor)

	return ci
}

// parseHTML parses an HTML page and extracts feed items based on the given
// params.
func parseHTML(data []byte, params any) ([]feed.RawItem, error) {
	var p htmlParams
	if err := parseParams(params, &p); err != nil {
		return nil, fmt.Errorf("cannot parse HTML feed params: %v", err)
	}
	// The parseParams call should have validated the base URL already.
	baseURL, _ := url.Parse(p.BaseURL)

	if p.Encoding != "" {
		if encodingMap[p.Encoding] == nil {
			return nil, fmt.Errorf("cannot find encoding: %s", p.Encoding)
		}
		var err error
		data, err = encodingMap[p.Encoding].NewDecoder().Bytes(data)
		if err != nil {
			return nil, fmt.Errorf("cannot decode HTML as %s: %v", p.Encoding, err)
		}
	}

	cisByURL := make(map[string]*candidateItem)
	doc, err := html.ParseWithOptions(bytes.NewReader(data), html.ParseOptionEnableScripting(false))
	if err != nil {
		return nil, fmt.Errorf("cannot parse HTML: %v", err)
	}

	var containers []*html.Node
	var findContainers func(*html.Node)
	findContainers = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == p.ContainerTag && matchAttrs(n, p.ContainerAttrs) {
			containers = append(containers, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findContainers(c)
		}
	}
	findContainers(doc)

	var position int
	for _, container := range containers {
		var findCandidateItems func(*html.Node)
		findCandidateItems = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "a" {
				ci := extractCandidateItem(n, baseURL, p.AllowedPrefixes)
				if ci != nil {
					if cisByURL[ci.url] == nil {
						ci.position = position
						position++
						cisByURL[ci.url] = ci
					} else {
						cisByURL[ci.url].merge(ci)
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				findCandidateItems(c)
			}
		}
		findCandidateItems(container)
	}

	cis := make([]*candidateItem, 0, len(cisByURL))
	for _, ci := range cisByURL {
		cis = append(cis, ci)
	}
	sort.Slice(cis, func(i, j int) bool {
		return cis[i].position < cis[j].position
	})

	var rawItems []feed.RawItem
	for _, ci := range cis {
		title := "Unknown title"
		if p.TitlePos < len(ci.parts) {
			title = ci.parts[p.TitlePos]
		} else if len(ci.parts) > 0 {
			title = ci.parts[0]
		}
		rawItems = append(rawItems, feed.RawItem{
			URL:      ci.url,
			Title:    title,
			Content:  silentlySanitizeHTML(strings.Join(ci.parts, "<br/>"), nil),
			Position: ci.position,
		})
	}

	return rawItems, nil
}

// matchAttrs returns true if the given node n has all the attributes specified
// in attrs with the same values.
func matchAttrs(n *html.Node, attrs map[string]string) bool {
	nMatches := 0
	for _, attr := range n.Attr {
		if val, ok := attrs[attr.Key]; ok && val == attr.Val {
			nMatches++
		}
	}
	return nMatches == len(attrs)
}
