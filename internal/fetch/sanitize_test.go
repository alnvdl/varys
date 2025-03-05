package fetch_test

import (
	"strings"
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
		expected: `<img alt="image"/>`,
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
				<img src="ftp://example.com/image.jpg" data-attr="whatever" alt="image">
				<figcaption>Image</figcaption>
			</figure>
			<script>alert('xss3')</script>
			<p>Paragraph</p>
		</div>`,
		expected: `<div><a href="http://example.com" title="example">Example</a><figure><img alt="image"/><figcaption>Image</figcaption></figure><p>Paragraph</p></div>`,
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
			t.Errorf("SilentlySanitizeHTML(`%s`) = `%s`; want `%s`", test.input, result, test.expected)
		}
	}
}

func TestSanitizePlainText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{{
		input:    `just some text`,
		expected: `just some text`,
	}, {
		input:    `Paragraph 1<div><p>Paragraph 2</p></div>`,
		expected: `Paragraph 1`,
	}, {
		input:    `Example<a href="http://example.com" title="example">Link</a>`,
		expected: `Example`,
	}, {
		input:    `<img src="http://example.com/image.jpg" alt="image">`,
		expected: ``,
	}, {
		input:    `<div><figure><img src="http://example.com/image.jpg" alt="image"><figcaption>Image</figcaption></figure></div>Content`,
		expected: `Content`,
	}, {
		input: `
			Content 1
			<div>
				<script>alert('xss1')</script>
				<p>Paragraph 2</p>
				<script>alert('xss2')</script>
			</div>
			Content 3`,
		expected: "Content 1\n\t\t\t\n\t\t\tContent 3",
	}}

	renderWhitespace := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\n", "\\n"), "\t", "\\t")
	}

	for _, test := range tests {
		result := fetch.SilentlySanitizePlainText(test.input)
		if result != test.expected {
			t.Errorf("SilentlySanitizePlainText(`%s`) = `%s`; want `%s`", test.input, renderWhitespace(result), renderWhitespace(test.expected))
		}
	}
}
