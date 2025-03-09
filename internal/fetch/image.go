package fetch

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/alnvdl/varys/internal/feed"
	"golang.org/x/net/html"
)

// imageParams defines the parameters for parseImage.
type imageParams struct {
	MimeType string `json:"mime_type"`
	URL      string `json:"url"`
	Title    string `json:"title"`
}

func (p *imageParams) validate() error {
	if p.MimeType == "" {
		return fmt.Errorf("mime_type cannot be empty")
	}
	if p.URL == "" {
		return fmt.Errorf("url cannot be empty")
	}
	if p.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	return nil
}

// parseImage parses image data and returns a single RawItem. This can be used
// for images hosted in the same URL that get updated frequently.
func parseImage(data []byte, params any) ([]feed.RawItem, error) {
	var p imageParams
	if err := parseParams(params, &p); err != nil {
		return nil, fmt.Errorf("cannot parse image params: %v", err)
	}

	date := time.Now().Format("2006-01-02 15:04")
	title := fmt.Sprintf("%s - %s", p.Title, date)

	imgSrc := fmt.Sprintf("data:%s;base64,%s", p.MimeType, base64.StdEncoding.EncodeToString(data))
	imgNode := &html.Node{
		Type: html.ElementNode,
		Data: "img",
		Attr: []html.Attribute{{Key: "src", Val: imgSrc}},
	}
	var buf bytes.Buffer
	err := html.Render(&buf, imgNode)
	if err != nil {
		return nil, fmt.Errorf("cannot render HTML for img: %v", err)
	}

	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash[:])
	urlWithHash := fmt.Sprintf("%s#%s", p.URL, hashStr)

	rawItem := feed.RawItem{
		URL:     urlWithHash,
		Title:   title,
		Content: silentlySanitizeHTML(buf.String()),
	}

	return []feed.RawItem{rawItem}, nil
}
