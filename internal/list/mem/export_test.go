package mem

import "github.com/alnvdl/varys/internal/feed"

func SetFeeds(l *List, feeds map[string]*feed.Feed) {
	l.feeds = feeds
}

func Feeds(l *List) map[string]*feed.Feed {
	return l.feeds
}

var Save = (*List).save
var Load = (*List).load

type SerializedList serializedList

var AllFeed = allFeed
