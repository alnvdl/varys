package fetch_test

import (
	"net/url"
	"testing"

	"github.com/alnvdl/varys/internal/fetch"
)

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		desc string

		input   string
		baseURL string

		expected string
	}{{
		desc:     "plain text",
		input:    `just some text`,
		expected: `just some text`,
		baseURL:  "https://example.com",
	}, {
		desc: "script tags",
		input: `<div>&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;
			<script>alert('xss1')</script><p>Paragraph</p></div>`,
		baseURL: "",
		expected: `<div>&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;
			<p>Paragraph</p></div>`,
	}, {
		desc:     "nested script tags",
		input:    `<div><script>alert('xss1')</script><p>Paragraph</p></div>`,
		baseURL:  "https://example.com",
		expected: `<div><p>Paragraph</p></div>`,
	}, {
		desc:     "valid absolute URL in a href",
		input:    `<a href="http://example.com" title="example">Example</a>`,
		baseURL:  "https://example.com",
		expected: `<a href="http://example.com" title="example">Example</a>`,
	}, {
		desc:     "valid relative URL in a href",
		input:    `<a href="/path" title="example">Example</a>`,
		baseURL:  "http://example.com",
		expected: `<a href="http://example.com/path" title="example">Example</a>`,
	}, {
		desc:     "valid relative URL in a href and no baseURL",
		input:    `<a href="/path" title="example">Example</a>`,
		baseURL:  "",
		expected: `<a href="/path" title="example">Example</a>`,
	}, {
		desc:     "valid absolure URL in img src",
		input:    `<img src="http://example.com/image.jpg" alt="image">`,
		baseURL:  "https://example.com",
		expected: `<img src="http://example.com/image.jpg" alt="image"/>`,
	}, {
		desc:     "valid relative URL in img src",
		input:    `<img src="path/image.jpg" alt="image">`,
		baseURL:  "https://example.com",
		expected: `<img src="https://example.com/path/image.jpg" alt="image"/>`,
	}, {
		desc:     "valid relative URL in img src and no baseURL",
		input:    `<img src="path/image.jpg" alt="image">`,
		baseURL:  "",
		expected: `<img src="path/image.jpg" alt="image"/>`,
	}, {
		desc:     "figure with img and figcaption",
		input:    `<figure><img src="http://example.com/image.jpg" alt="image"><figcaption>Image</figcaption></figure>`,
		baseURL:  "https://example.com",
		expected: `<figure><img src="http://example.com/image.jpg" alt="image"/><figcaption>Image</figcaption></figure>`,
	}, {
		desc:     "invalid tag",
		input:    `<div><badtag><img src="http://example.com/image.jpg" alt="image"></badtag></div>`,
		baseURL:  "https://example.com",
		expected: `<div></div>`,
	}, {
		desc:     "invalid base URL",
		input:    `<div><img src=":#!@#!@#" alt="image1"><badtag><img src="path/image2.jpg" alt="image2"></badtag></div>`,
		baseURL:  "https://example.com",
		expected: `<div><img alt="image1"/></div>`,
	}, {
		desc: "complex HTML",
		input: `
		<div>
			<script>alert('xss1')</script>
			<badtag>
				<img src="http://example.com/image.jpg" data-attr="whatever" alt="image that should not be there" />
			</badtag>
			<script>alert('xss2')</script>
			<a href="http://example.com" title="example">Example</a><script>alert('xss')</script>
			<figure>
				<img src="https://example.com/image.jpg" data-attr="whatever" alt="image">
				<figcaption>Image</figcaption>
			</figure>
			<script>alert('xss3')</script>
			<p>Paragraph</p>
		</div>`,
		baseURL:  "https://example.com",
		expected: "<div>\n\t\t\t\n\t\t\t\n\t\t\t\n\t\t\t<a href=\"http://example.com\" title=\"example\">Example</a>\n\t\t\t<figure>\n\t\t\t\t<img src=\"https://example.com/image.jpg\" alt=\"image\"/>\n\t\t\t\t<figcaption>Image</figcaption>\n\t\t\t</figure>\n\t\t\t\n\t\t\t<p>Paragraph</p>\n\t\t</div>",
	}, {
		desc: "form with action",
		input: `
		<div onclick="alert('click')">Click me</div>
		<form action="/submit" method="post">
			<input type="text" name="name">
			<input type="submit">
		</form>`,
		expected: `<div>Click me</div>`,
		baseURL:  "https://example.com",
	}}

	for _, test := range tests {
		var baseURL *url.URL
		if test.baseURL != "" {
			var err error
			baseURL, err = url.Parse(test.baseURL)
			if err != nil {
				t.Fatalf("failed to parse base URL for %s: %s: %v", test.desc, test.baseURL, err)
				continue
			}
		}
		result := fetch.SilentlySanitizeHTML(test.input, baseURL)
		if result != test.expected {
			t.Errorf("unexpected sanitized HTML for %s: got %#v, want %#v", test.desc, result, test.expected)
		}
	}
}
