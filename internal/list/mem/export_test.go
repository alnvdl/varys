package mem

import "github.com/alnvdl/varys/internal/feed"

type SerializedList serializedList

func SetFeedsMap(l *List, feeds map[string]*feed.Feed) {
	l.feeds = feeds
}

func FeedsMap(l *List) map[string]*feed.Feed {
	return l.feeds
}

var Save = (*List).save
var Load = (*List).load
var AllFeed = allFeed
