package list

import "github.com/alnvdl/varys/internal/feed"

func SetFeeds(l *Simple, feeds map[string]*feed.Feed) {
	l.feeds = feeds
}

func Feeds(l *Simple) map[string]*feed.Feed {
	return l.feeds
}

type SerializedList serializedSimpleStore

var SimpleAllFeed = simpleStoreAllFeed
