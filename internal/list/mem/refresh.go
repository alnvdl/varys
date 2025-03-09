package mem

import (
	"log/slog"
	"sync"
	"time"

	"github.com/alnvdl/varys/internal/fetch"
)

// Refresh fetches all feeds in the list and then refreshes them. The auto flag
// indicates whether this refresh was triggered automatically or manually.
func (l *List) Refresh(auto bool) {
	if !auto {
		defer l.delayPersist()
	}
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	wg := sync.WaitGroup{}

	for _, feed := range l.feeds {
		wg.Add(1)
		go func() {
			feed.Refresh(l.fetcher(fetch.FetchParams{
				URL:        feed.URL,
				FeedName:   feed.Name,
				FeedType:   feed.Type,
				FeedParams: feed.Params,
			}))
			wg.Done()
		}()
	}

	wg.Wait()
	if l.refreshCallback != nil {
		l.refreshCallback()
	}
}

func (l *List) initRefresh() {
	slog.Info("running initial feed refresh")
	l.Refresh(true)

	l.wg.Add(1)
	go func() {
		l.autoRefresh()
		l.wg.Done()
	}()
}

func (l *List) autoRefresh() {
	if l.refreshInterval == 0 {
		slog.Info("auto-refresh disabled")
		return
	}

	log := slog.With(
		slog.Duration("refreshInterval", l.refreshInterval),
	)
	log.Info("auto-refresh enabled")
	for {
		select {
		case <-l.close:
			log.Info("stopping auto-refresh")
			return
		case <-time.After(l.refreshInterval):
			log.Info("auto-refresh interval reached")
			l.Refresh(true)
			log.Info("auto-refresh completed")
		}
	}
}
