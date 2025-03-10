package fetch_test

import (
	"testing"

	"github.com/alnvdl/varys/internal/fetch"
)

func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{{
		input:    `just some text`,
		expected: `just some text`,
	}, {
		input: `<div>&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;
			<script>alert('xss1')</script><p>Paragraph</p></div>`,
		expected: `<div>&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;
			<p>Paragraph</p></div>`,
	}, {
		input:    `<div><script>alert('xss1')</script><p>Paragraph</p></div>`,
		expected: `<div><p>Paragraph</p></div>`,
	}, {
		input:    `<div><script>alert('xss1')</script><p>Paragraph</p></div>`,
		expected: `<div><p>Paragraph</p></div>`,
	}, {
		input:    `<a href="http://example.com" title="example">Example</a>`,
		expected: `<a href="http://example.com" title="example">Example</a>`,
	}, {
		input:    `<img src="http://example.com/image.jpg" alt="image">`,
		expected: `<img src="http://example.com/image.jpg" alt="image"/>`,
	}, {
		input:    `<img src="ftp://example.com/image.jpg" alt="image">`,
		expected: `<img src="ftp://example.com/image.jpg" alt="image"/>`,
	}, {
		input:    `<figure><img src="http://example.com/image.jpg" alt="image"><figcaption>Image</figcaption></figure>`,
		expected: `<figure><img src="http://example.com/image.jpg" alt="image"/><figcaption>Image</figcaption></figure>`,
	}, {
		input:    `<div><badtag><img src="http://example.com/image.jpg" alt="image"></badtag></div>`,
		expected: `<div></div>`,
	}, {
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
		expected: "<div>\n\t\t\t\n\t\t\t\n\t\t\t\n\t\t\t<a href=\"http://example.com\" title=\"example\">Example</a>\n\t\t\t<figure>\n\t\t\t\t<img src=\"https://example.com/image.jpg\" alt=\"image\"/>\n\t\t\t\t<figcaption>Image</figcaption>\n\t\t\t</figure>\n\t\t\t\n\t\t\t<p>Paragraph</p>\n\t\t</div>",
	}, {
		input: `
		<div onclick="alert('click')">Click me</div>
		<form action="/submit" method="post">
			<input type="text" name="name">
			<input type="submit">
		</form>`,
		expected: `<div>Click me</div>`,
	},
	}

	for _, test := range tests {
		result := fetch.SilentlySanitizeHTML(test.input)
		if result != test.expected {
			t.Errorf("unexpected sanitized HTML: got %#v, want %#v", result, test.expected)
		}
	}
}
