package fetch

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var defaultAllowedTags = map[string]bool{
	"a":          true,
	"abbr":       true,
	"acronym":    true,
	"b":          true,
	"blockquote": true,
	"br":         true,
	"code":       true,
	"del":        true,
	"div":        true,
	"em":         true,
	"figure":     true,
	"figcaption": true,
	"h1":         true,
	"h2":         true,
	"h3":         true,
	"h4":         true,
	"h5":         true,
	"h6":         true,
	"i":          true,
	"img":        true,
	"ins":        true,
	"li":         true,
	"ol":         true,
	"p":          true,
	"pre":        true,
	"s":          true,
	"span":       true,
	"strike":     true,
	"strong":     true,
	"u":          true,
	"ul":         true,
}

var defaultAllowedAttrs = map[string]map[string]bool{
	"a":       {"href": true, "title": true},
	"abbr":    {"title": true},
	"acronym": {"title": true},
	"img":     {"alt": true, "src": true},
}

// safeURLSchemes lists the URL schemes allowed in href and src attributes.
// Everything else (e.g. javascript:, vbscript:) is rejected to prevent XSS via
// scriptable URLs. The data: scheme is handled separately (see isSafeDataURL).
var safeURLSchemes = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
}

// safeDataURLPrefixes lists the data: URL media type prefixes allowed in src
// attributes. Only non-scriptable raster image types are permitted; notably
// image/svg+xml and text/html are excluded because they can execute scripts.
var safeDataURLPrefixes = []string{
	"data:image/png",
	"data:image/jpeg",
	"data:image/gif",
	"data:image/webp",
}

// isSafeDataURL reports whether val is a data: URL with an allowed image media
// type. The check is case-insensitive on the scheme and media type.
func isSafeDataURL(val string) bool {
	lower := strings.ToLower(val)
	for _, prefix := range safeDataURLPrefixes {
		// Checks for patterns like data:image/png;base64,... and
		// data:image/png,...
		if strings.HasPrefix(lower, prefix+";") || strings.HasPrefix(lower, prefix+",") {
			return true
		}
	}
	return false
}

// maxSanitizeDepth bounds how deep the sanitizer recurses into the parsed HTML
// tree. It guards against stack exhaustion from maliciously nested documents
// while staying well above the nesting depth of any legitimate feed content.
const maxSanitizeDepth = 512

// SilentlySanitizeHTML works like sanitizeHTML but it uses a default
// configuration and silences errors.
func silentlySanitizeHTML(input string, baseURL *url.URL) string {
	sanitized, _ := sanitizeHTML(input, defaultAllowedTags, defaultAllowedAttrs, baseURL)
	return sanitized
}

// sanitizeHTML sanitizes the input HTML string by allowing only specific tags
// and attributes in a way known to be safe for including as a fragment inside
// other HTML. It also resolves relative URLs in href and src attrs using the
// provided baseURL if not nil.
func sanitizeHTML(input string, allowedTags map[string]bool, allowedAttrs map[string]map[string]bool, baseURL *url.URL) (string, error) {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return "", fmt.Errorf("cannot parse HTML: %v", err)
	}

	newDoc := &html.Node{
		Type: html.DocumentNode,
	}
	sanitizeNode(doc, newDoc, allowedTags, allowedAttrs, baseURL, 0)

	var buf bytes.Buffer
	if err := html.Render(&buf, newDoc); err != nil {
		return "", fmt.Errorf("cannot render sanitized HTML: %v", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func sanitizeNode(node, newParent *html.Node, allowedTags map[string]bool, allowedAttrs map[string]map[string]bool, baseURL *url.URL, depth int) {
	if depth > maxSanitizeDepth {
		return
	}
	if node.Type == html.ElementNode && allowedTags[node.Data] {
		newNode := &html.Node{
			Type: html.ElementNode,
			Data: node.Data,
		}
		var attrs []html.Attribute
		for _, attr := range node.Attr {
			if allowedAttrs[node.Data][attr.Key] {
				if attr.Key == "href" || attr.Key == "src" {
					parsedURL, err := url.Parse(attr.Val)
					if err != nil {
						continue
					}
					// Reject scriptable schemes such as javascript:
					// and vbscript:. Relative URLs have an empty scheme and
					// are allowed. data: URLs are only allowed for safe image
					// media types.
					scheme := strings.ToLower(parsedURL.Scheme)
					if scheme != "" && !safeURLSchemes[scheme] {
						if scheme != "data" || !isSafeDataURL(attr.Val) {
							continue
						}
					}
					if baseURL != nil && !parsedURL.IsAbs() {
						attr.Val = baseURL.ResolveReference(parsedURL).String()
					}
				}
				attrs = append(attrs, html.Attribute(attr))
			}
		}
		newNode.Attr = attrs
		newParent.AppendChild(newNode)
		newParent = newNode
	}

	isValidTextNode := node.Type == html.TextNode && node.Parent != nil
	if isValidTextNode && (allowedTags[node.Parent.Data] || node.Parent.DataAtom == atom.Body) {
		newNode := &html.Node{
			Type: html.TextNode,
			Data: node.Data,
		}
		newParent.AppendChild(newNode)
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if node.Type == html.DocumentNode ||
			(node.Type == html.ElementNode && allowedTags[node.Data]) ||
			(node.Type == html.ElementNode && (node.DataAtom == atom.Html || node.DataAtom == atom.Body)) {
			sanitizeNode(c, newParent, allowedTags, allowedAttrs, baseURL, depth+1)
		}
	}
}
