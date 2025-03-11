package fetch

import (
	"net/url"
	"strings"
)

// urlToString returns the string representation of a URL. If the URL is nil,
// an empty string is returned.
func urlToString(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}

// resolveURL resolves a URL against a base URL and returns the resolved URL if
// it is prefix matches one of the allowedPrefixes. If the URL is invalid or
// its prefix after resolving does not match any of the allowed prefixes, nil
// is returned. If the base URL is nil, u will be resolved to itself. If the
// allowed prefixes are nil, the URL is always allowed.
func resolveURL(u string, baseURL *url.URL, allowedPrefixes []string) *url.URL {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil
	}
	if parsedURL.String() == "" {
		return nil
	}

	resolvedURL := parsedURL
	if baseURL != nil && !parsedURL.IsAbs() {
		resolvedURL = baseURL.ResolveReference(parsedURL)
	}

	if allowedPrefixes == nil {
		return resolvedURL
	}
	for _, allowed := range allowedPrefixes {
		if strings.HasPrefix(resolvedURL.String(), allowed) {
			return resolvedURL
		}
	}

	return nil
}
