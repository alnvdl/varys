package mem_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/list"
	"github.com/alnvdl/varys/internal/list/mem"
)

// genRandomTestFileName generates a random file name under /tmp.
func genRandomTestFileName() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 5)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return filepath.Join(os.TempDir(), "varys_test_"+string(b))
}

func TestListSave(t *testing.T) {
	t.Parallel()
	feeds := map[string]*feed.Feed{
		"feed1": {
			Name: "Feed 1",
			Type: "xml",
			URL:  "http://example.com/feed1",
			Items: map[string]*feed.Item{
				"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}},
			},
		},
		"feed2": {
			Name: "Feed 2",
			Type: "xml",
			URL:  "http://example.com/feed2",
			Items: map[string]*feed.Item{
				"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}},
			},
		},
	}

	// Save the l to a buffer in JSON.
	l := mem.NewList(mem.ListParams{})
	mem.SetFeedsMap(l, feeds)
	var buf bytes.Buffer
	err := mem.Save(l, &buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Load the list from the buffer.
	var data mem.SerializedList
	err = json.Unmarshal(buf.Bytes(), &data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(data.Feeds) != len(feeds) {
		t.Fatalf("expected %d feeds, got %d", len(feeds), len(data.Feeds))
	}
	for key, f := range feeds {
		savedFeed, ok := data.Feeds[key]
		if !ok {
			t.Fatalf("expected feed %s to be present", key)
		}
		checkFeed(t, *savedFeed, *f)
	}
}

type errorWriter struct{}

func (ew *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("simulated write error")
}

func TestListSaveError(t *testing.T) {
	t.Parallel()
	l := mem.NewList(mem.ListParams{})
	mem.SetFeedsMap(l, make(map[string]*feed.Feed))

	err := mem.Save(l, &errorWriter{})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	expectedErr := "cannot serialize feed list: simulated write error"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err.Error())
	}
}

func TestListLoad(t *testing.T) {
	t.Parallel()
	feeds := map[string]*feed.Feed{
		"feed1": {
			Name: "Feed 1",
			Type: "xml",
			URL:  "http://example.com/feed1",
			Items: map[string]*feed.Item{
				"item1": {RawItem: feed.RawItem{URL: "http://example.com/item1", Title: "Item 1"}},
			},
		},
		"feed2": {
			Name: "Feed 2",
			Type: "xml",
			URL:  "http://example.com/feed2",
			Items: map[string]*feed.Item{
				"item2": {RawItem: feed.RawItem{URL: "http://example.com/item2", Title: "Item 2"}},
			},
		},
	}

	// Serialize the feeds to JSON.
	var buf bytes.Buffer
	var data mem.SerializedList
	data.Feeds = feeds
	err := json.NewEncoder(&buf).Encode(&data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Load the feeds from the JSON.
	loadedList := mem.NewList(mem.ListParams{})
	err = mem.Load(loadedList, &buf)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	loadedFeeds := mem.FeedsMap(loadedList)
	if len(loadedFeeds) != len(feeds) {
		t.Fatalf("expected %d feeds, got %d", len(feeds), len(loadedFeeds))
	}

	for key, f := range feeds {
		loadedFeed, ok := loadedFeeds[key]
		if !ok {
			t.Fatalf("expected feed %s to be present", key)
		}
		checkFeed(t, *loadedFeed, *f)
	}
}

func TestListLoadError(t *testing.T) {
	t.Parallel()
	corruptedJSON := `{"feeds": {"feed1":`

	list := mem.NewList(mem.ListParams{})
	err := mem.Load(list, bytes.NewBufferString(corruptedJSON))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	expectedErr := "cannot deserialize feed list: unexpected EOF"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err.Error())
	}
}

func TestPersistence(t *testing.T) {
	t.Parallel()

	// Generate a random file name.
	dbFilePath := genRandomTestFileName()

	// Check the file does not exist.
	if _, err := os.Stat(dbFilePath); !os.IsNotExist(err) {
		t.Fatalf("expected file %s to not exist", dbFilePath)
	}

	persistNotify := make(chan error)
	// Create a new list with a 5s persistence interval.
	l := mem.NewList(mem.ListParams{
		DBFilePath:      dbFilePath,
		PersistInterval: 1 * time.Second,
		PersistCallback: func(err error) {
			persistNotify <- err
		},
	})

	// Check that the file was created.
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		t.Fatalf("expected file %s to be created", dbFilePath)
	}

	// Load some feed data.
	feeds := []*list.InputFeed{{
		Name: "Feed 1",
		URL:  "http://example.com/feed1",
		Type: "xml",
	}, {
		Name: "Feed 2",
		URL:  "http://example.com/feed2",
		Type: "xml",
	}}
	l.LoadFeeds(feeds)

	// Wait for the file to be persisted.
	select {
	case <-time.After(2 * time.Second):
		t.Fatalf("expected persistence to be triggered")
	case err := <-persistNotify:
		if err != nil {
			t.Fatalf("expected no persistence error, got %v", err)
		}
	}

	// Check that the file was written and has the expected content.
	file, err := os.Open(dbFilePath)
	if err != nil {
		t.Fatalf("expected no error opening file, got %v", err)
	}
	defer file.Close()

	var data mem.SerializedList
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		t.Fatalf("expected no error decoding file content, got %v", err)
	}

	if len(data.Feeds) != len(feeds) {
		t.Fatalf("expected %d feeds, got %d", len(feeds), len(data.Feeds))
	}

	for _, f := range feeds {
		if _, ok := data.Feeds[feed.UID(f.URL)]; !ok {
			t.Fatalf("expected feed %s to be present", f.URL)
		}
	}

	// Close the list and check it was persisted again.
	go l.Close()
	select {
	case <-time.After(1 * time.Second):
		t.Fatalf("expected close persistence to be triggered")
	case err := <-persistNotify:
		if err != nil {
			t.Fatalf("expected no persistence error, got %v", err)
		}
	}
}

func TestNoPersistenceIfLoadFails(t *testing.T) {
	t.Parallel()

	// Generate a random file name.
	dbFilePath := genRandomTestFileName()

	// Write corrupted JSON content to the file.
	file, err := os.Create(dbFilePath)
	if err != nil {
		t.Fatalf("expected no error creating file, got %v", err)
	}
	_, err = file.WriteString("{")
	if err != nil {
		t.Fatalf("expected no error writing to file, got %v", err)
	}
	file.Close()

	persistNotify := make(chan error, 1)
	l := mem.NewList(mem.ListParams{
		DBFilePath:      dbFilePath,
		PersistInterval: 500 * time.Millisecond,
		PersistCallback: func(err error) {
			persistNotify <- err
		},
	})

	// Wait for a short period to ensure no persistence is triggered.
	select {
	case <-time.After(1 * time.Second):
		// Expected path: no persistence callback should be invoked.
	case err := <-persistNotify:
		t.Fatalf("expected no persistence callback call, got %v", err)
	}

	// Close the list.
	l.Close()
}
