package mem

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

// initPersist initializes the persistence mechanism. If the dbFilePath is not
// empty, the feed list is loaded from the file. If the file does not exist, it
// is created. If the file cannot be opened, an error is logged. If the file
// cannot be parsed, an error is logged. If dbFilePath is empty (meaning
// persistence is disabled), the function returns immediately. If the file is
// successfully loaded, the auto-persistence mechanism is started.
func (l *List) initPersist() {
	log := slog.With(slog.String("dbFilePath", l.dbFilePath))

	if l.dbFilePath != "" {
		var inputFile *os.File
		var err error
		inputFile, err = os.OpenFile(l.dbFilePath, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Error("cannot open feed list file for reading",
				slog.String("err", err.Error()))
		} else {
			defer inputFile.Close()
			err = l.load(inputFile)
			if err != nil && !errors.Is(err, io.EOF) {
				log.Error("error parsing input file",
					slog.String("err", err.Error()))
			}
			if errors.Is(err, io.EOF) {
				err = nil
			}
		}

		// Only enable auto persistence if the file was successfully loaded.
		if err == nil {
			l.wg.Add(1)
			go func() {
				l.autoPersist()
				l.wg.Done()
			}()
		} else {
			slog.Info("auto-persist not configured due to error",
				slog.String("err", err.Error()))
		}
	} else {
		log.Info("no persistence configured")
	}
}

func (l *List) Close() {
	close(l.close)
	l.wg.Wait()
}

// persist tries to save the feed list to the file. If the operation fails, the
// error is logged. The callback is called, with an error or nil in case of
// success.
func (l *List) persist(reason string) {
	log := slog.With(
		slog.String("reason", reason),
		slog.String("dbFilePath", l.dbFilePath),
		slog.Duration("persistInterval", l.persistInterval),
	)
	outputFile, err := os.OpenFile(l.dbFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	if err != nil {
		log.Error("cannot open feed list file for writing",
			slog.String("err", err.Error()),
		)
		if l.persistCallback != nil {
			l.persistCallback(err)
		}
		return
	}
	if err := l.save(outputFile); err != nil {
		log.Error("cannot persist feed list to file",
			slog.String("err", err.Error()),
		)
	}
	errClose := outputFile.Close()
	if l.persistCallback != nil {
		l.persistCallback(errors.Join(err, errClose))
	}
	log.Info("persisted feed list to file")
}

// autoPersist periodically saves the feed list to the file. The interval is
// defined by the persistInterval field. If the interval is 0, the function
// returns immediately.
func (l *List) autoPersist() {
	if l.persistInterval == 0 {
		slog.Info("auto-persist disabled")
		return
	}

	log := slog.With(
		slog.String("dbFilePath", l.dbFilePath),
		slog.Duration("persistInterval", l.persistInterval),
	)
	log.Info("auto-persist enabled")
	for {
		select {
		case <-l.close:
			log.Info("stopping auto-persist")
			l.persist("close")
			return
		case <-time.After(l.persistInterval):
			log.Info("auto-persist interval reached")
			l.persist("auto-refresh")
		case <-l.persistBackoff:
			// Do nothing, and the persistence timer will restart.
			log.Debug("auto-persist backoff received, resetting timer")
		}
	}
}

// delayPersist sends a signal to the auto-persist mechanism to delay the next
// persistence operation. If the signal cannot be sent, a log message is
// printed.
func (l *List) delayPersist() {
	select {
	case l.persistBackoff <- true:
	default:
		slog.Info("failed attempt to delay auto-persist; auto-persist may not be running")
	}
}

// save serializes the feed list to the given writer.
func (l *List) save(w io.Writer) error {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	enc := json.NewEncoder(w)
	if os.Getenv("DEBUG") != "" {
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
	}
	err := enc.Encode(serializedList{Feeds: l.feeds})
	if err != nil {
		return fmt.Errorf("cannot serialize feed list: %w", err)
	}
	return nil
}

// load deserializes the feed list from the given reader.
func (l *List) load(r io.Reader) error {
	l.muFeeds.Lock()
	defer l.muFeeds.Unlock()

	dec := json.NewDecoder(r)
	data := serializedList{}
	err := dec.Decode(&data)
	if err != nil {
		return fmt.Errorf("cannot deserialize feed list: %w", err)
	}
	l.feeds = data.Feeds

	return nil
}
