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

func (l *List) initPersistence() {
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
			return
		case <-time.After(l.persistInterval):
			log.Info("auto-persist interval reached")
			outputFile, err := os.OpenFile(l.dbFilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
			if err != nil {
				log.Error("cannot open feed list file for writing, will try again after another persist interval",
					slog.String("err", err.Error()),
				)
				if l.persistCallback != nil {
					l.persistCallback(err)
				}
				continue
			}
			log.Info("persisting feed list to file")
			if err := l.save(outputFile); err != nil {
				log.Error("cannot persist feed list to file, will try again after another persist interval",
					slog.String("err", err.Error()),
				)
			}
			outputFile.Close()
			if l.persistCallback != nil {
				l.persistCallback(err)
			}
		case <-l.persistBackoff:
			// Do nothing, and the persistence timer will restart.
			log.Debug("auto-persist backoff received, resetting timer")
		}
	}
}

func (l *List) backoffPersist() {
	l.persistBackoff <- true
}

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
