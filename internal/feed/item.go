package feed

// RawItem is the representation of an item as it comes from a feed.
type RawItem struct {
	// URL is the URL of the item.
	URL string `json:"url"`
	// Title is a short title or description of the item.
	Title string `json:"title"`
	// Authors is short summary of authors of the item (usually
	// comma-separated).
	Authors string `json:"authors"`
	// Content is the full content of the item, usually a sanitized HTML
	// fragment.
	Content string `json:"content"`
	// Position is the position of the item in the feed when it was first seen.
	// Assuming two items are first seen at the same time, a lower position
	// typically means a newer item (i.e., that's how blogs are typically laid
	// out).
	Position int `json:"position"`
}

// UID returns a unique identifier for the raw item if it is valid. Otherwise,
// returns an empty string.
func (i *RawItem) UID() string {
	if !i.IsValid() {
		return ""
	}
	return UID(i.URL)
}

// IsValid returns true if the item has a URL and a title, thus being
// considered minimally valid.
func (i *RawItem) IsValid() bool {
	return i.URL != "" && i.Title != ""
}

// Item is the representation of an item in the application, with additional
// fields that are not present in the raw item.
type Item struct {
	RawItem

	// FeedUID is the UID of the feed that this item belongs to. This field is
	// populated by the feed itself.
	FeedUID string `json:"feed_uid"`
	// Timestamp is the time when the item was first seen. This field is
	// populated by the feed itself.
	Timestamp int64 `json:"timestamp"`

	// Read is true if the item was marked as read by the user.
	Read bool `json:"read"`
}

// ItemSummary is the external representation of the item (e.g., for presenting
// to users).
type ItemSummary struct {
	// UID is the unique identifier of the item.
	UID string `json:"uid"`

	// FeedUID is the UID of the feed that this item belongs to.
	FeedUID string `json:"feed_uid"`

	// FeedName is the name of the feed that this item belongs to.
	FeedName string `json:"feed_name"`

	// URL is the URL of the item. It comes directly from the feed and is not
	// sanitized. Do not embed it in the page directly without proper measures.
	URL string `json:"url"`

	// Title is a short title or description of the item. It comes directly
	// from the feed and is not sanitized. Do not embed it in the page directly
	// without proper measures.
	Title string `json:"title"`

	// Timestamp is the time when the item was first seen.
	Timestamp int64 `json:"timestamp"`

	// Authors is a short comma-separated summary of authors of the item. It
	// comes directly from the feed and is not sanitized. Do not embed it in
	// the page directly without proper measures.
	Authors string `json:"authors"`

	// Read is true if the item was marked as read by the user.
	Read bool `json:"read"`

	// Content is the full content of the item, usually a sanitized HTML
	// fragment. It comes directly from the feed and may safely be embedded in
	// the page directly.
	Content string `json:"content,omitempty"`
}

// Refresh updates the item with certain changeable fields coming from the new
// raw item r. It returns true if the item was updated, false otherwise.
func (i *Item) Refresh(r RawItem) bool {
	changed := false
	// Only update the position if the item has no URL, meaning this Item is
	// being initialized with r.
	if i.URL == "" && i.Position != r.Position {
		i.Position = r.Position
	}
	if i.URL != r.URL {
		i.URL = r.URL
		changed = true
	}
	if i.Title != r.Title {
		i.Title = r.Title
		changed = true
	}
	if i.Authors != r.Authors {
		i.Authors = r.Authors
		changed = true
	}
	if i.Content != r.Content {
		i.Content = r.Content
		changed = true
	}
	return changed
}

func (i *Item) Summary(f *Feed, includeContent bool) *ItemSummary {
	is := &ItemSummary{
		UID:       i.UID(),
		FeedUID:   f.UID(),
		FeedName:  f.Name,
		URL:       i.URL,
		Title:     i.Title,
		Timestamp: i.Timestamp,
		Authors:   i.Authors,
		Read:      i.Read,
	}
	if includeContent {
		is.Content = i.Content
	}
	return is
}

// MarkRead marks all feed items as read.
func (i *Item) MarkRead() {
	i.Read = true
}
