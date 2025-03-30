package web_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alnvdl/varys/internal/feed"
	"github.com/alnvdl/varys/internal/timeutil"
	"github.com/alnvdl/varys/internal/web"
)

type mockFeedLister struct {
	feeds []*feed.FeedSummary
}

func (m *mockFeedLister) Summary() []*feed.FeedSummary {
	return m.feeds
}

func (m *mockFeedLister) FeedSummary(uid string) *feed.FeedSummary {
	for _, f := range m.feeds {
		if f.UID == uid {
			return f
		}
	}
	return nil
}

func (m *mockFeedLister) FeedItem(fuid, iuid string) *feed.ItemSummary {
	for _, f := range m.feeds {
		if f.UID == fuid {
			for _, item := range f.Items {
				if item.UID == iuid {
					return item
				}
			}
		}
	}
	return nil
}

func (m *mockFeedLister) MarkRead(fuid, iuid string, before int64) bool {
	for _, f := range m.feeds {
		if f.UID == fuid {
			// Marking a feed as read.
			if iuid == "" {
				for _, item := range f.Items {
					if item.Timestamp <= before {
						item.Read = true
					}
				}
				return true
			}
			// Marking an item as read.
			for _, item := range f.Items {
				if item.UID == iuid {
					item.Read = true
					return true
				}
			}
		}
	}
	return false
}

func (m *mockFeedLister) Refresh() {}

type performLoginParams struct {
	Token         string
	Body          string
	ExpectSuccess bool
}

func performLogin(t *testing.T, h http.Handler, params performLoginParams) *http.Cookie {
	var bodyBytes []byte
	if params.Body != "" {
		bodyBytes = []byte(params.Body)
	} else {
		body := map[string]string{"token": params.Token}
		bodyBytes, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(bodyBytes))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Errorf("expected Content-Security-Policy header to be set")
	}

	if params.ExpectSuccess {
		if rr.Code != http.StatusOK {
			t.Fatalf("expected login status %v, got %v", http.StatusOK, rr.Code)
		}

		cookie := rr.Header().Get("Set-Cookie")
		if cookie == "" {
			t.Fatalf("expected session cookie to be set")
		}

		parts := strings.Split(cookie, ";")
		sessionCookie := strings.Split(parts[0], "=")
		return &http.Cookie{Name: sessionCookie[0], Value: sessionCookie[1]}
	} else {
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected login status %v, got %v", http.StatusUnauthorized, rr.Code)
		}

		// Return a bogus cookie for testing authentication failures.
		return &http.Cookie{Name: "session", Value: "bogus-session"}
	}
}

func compareItems(t *testing.T, expected, actual []*feed.ItemSummary) {
	if len(expected) != len(actual) {
		t.Errorf("expected %d items, got %d", len(expected), len(actual))
	}

	for i, item := range actual {
		if item.UID != expected[i].UID || item.Title != expected[i].Title || item.URL != expected[i].URL ||
			item.FeedUID != expected[i].FeedUID || item.FeedName != expected[i].FeedName ||
			item.Timestamp != expected[i].Timestamp || item.Authors != expected[i].Authors ||
			item.Read != expected[i].Read || item.Content != expected[i].Content {
			t.Errorf("expected item %#v, got %#v", expected[i], item)
		}
	}
}

func compareFeeds(t *testing.T, expected, actual []*feed.FeedSummary) {
	if len(expected) != len(actual) {
		t.Errorf("expected %d feeds, got %d", len(expected), len(actual))
	}

	for i, feed := range actual {
		if feed.UID != expected[i].UID || feed.Name != expected[i].Name || feed.URL != expected[i].URL ||
			feed.LastUpdated != expected[i].LastUpdated || feed.LastError != expected[i].LastError ||
			feed.ItemCount != expected[i].ItemCount || feed.ReadCount != expected[i].ReadCount {
			t.Errorf("expected feed %v, got %v", expected[i], feed)
		}
		compareItems(t, expected[i].Items, feed.Items)
	}
}

func TestLogin(t *testing.T) {
	handlerParams := &web.HandlerParams{
		AccessToken: "valid-token",
		SessionKey:  []byte("test-session-key"),
	}
	h := web.NewHandler(handlerParams)

	tests := []struct {
		desc           string
		params         performLoginParams
		expectedStatus int
		expectCookie   bool
	}{{
		desc: "success",
		params: performLoginParams{
			Token:         "valid-token",
			ExpectSuccess: true,
		},
		expectedStatus: http.StatusOK,
		expectCookie:   true,
	}, {
		desc: "failure",
		params: performLoginParams{
			Token:         "invalid-token",
			ExpectSuccess: false,
		},
		expectedStatus: http.StatusUnauthorized,
		expectCookie:   false,
	}, {
		desc: "invalid JSON",
		params: performLoginParams{
			Body:          `{"token": "invalid-token"`,
			ExpectSuccess: false,
		},
		expectedStatus: http.StatusBadRequest,
		expectCookie:   false,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			performLogin(t, h, test.params)
		})
	}
}

func TestGetFeeds(t *testing.T) {
	tests := []struct {
		desc           string
		feeds          []*feed.FeedSummary
		expectedFeeds  []*feed.FeedSummary
		token          string
		expectedStatus int
		authSuccess    bool
	}{{
		desc:           "success: without feeds",
		token:          "valid-token",
		authSuccess:    true,
		feeds:          []*feed.FeedSummary{},
		expectedFeeds:  []*feed.FeedSummary{},
		expectedStatus: http.StatusOK,
	}, {
		desc:        "success: with feeds",
		token:       "valid-token",
		authSuccess: true,
		feeds: []*feed.FeedSummary{
			{UID: "1", Name: "Feed 1"},
			{UID: "2", Name: "Feed 2"},
		},
		expectedFeeds: []*feed.FeedSummary{
			{UID: "1", Name: "Feed 1"},
			{UID: "2", Name: "Feed 2"},
		},
		expectedStatus: http.StatusOK,
	}, {
		desc:           "failure: authentication with invalid cookie",
		token:          "invalid-token",
		authSuccess:    false,
		feeds:          []*feed.FeedSummary{},
		expectedFeeds:  []*feed.FeedSummary{},
		expectedStatus: http.StatusUnauthorized,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			feedList := &mockFeedLister{feeds: test.feeds}
			handlerParams := &web.HandlerParams{
				FeedList:    feedList,
				AccessToken: "valid-token",
				SessionKey:  []byte("test-session-key"),
			}
			h := web.NewHandler(handlerParams)

			cookie := performLogin(t, h, performLoginParams{
				Token:         test.token,
				ExpectSuccess: test.authSuccess,
			})

			req, _ := http.NewRequest("GET", "/api/feeds", nil)
			if cookie != nil {
				req.AddCookie(cookie)
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)
			if rr.Header().Get("Content-Type") == "" {
				t.Errorf("expected Content-Type header to be set")
			}
			if rr.Header().Get("Content-Security-Policy") == "" {
				t.Errorf("expected Content-Security-Policy header to be set")
			}

			if rr.Code != test.expectedStatus {
				t.Errorf("expected status %v, got %v", test.expectedStatus, rr.Code)
			}

			if test.authSuccess {
				var feeds []*feed.FeedSummary
				err := json.NewDecoder(rr.Body).Decode(&feeds)
				if err != nil {
					t.Fatalf("could not decode response: %v", err)
				}
				compareFeeds(t, test.expectedFeeds, feeds)
			}
		})
	}
}

func TestGetFeed(t *testing.T) {
	tests := []struct {
		desc           string
		feeds          []*feed.FeedSummary
		fuid           string
		expectedFeed   *feed.FeedSummary
		token          string
		expectedStatus int
		authSuccess    bool
	}{{
		desc:           "success: feed found",
		token:          "valid-token",
		authSuccess:    true,
		fuid:           "1",
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1"}},
		expectedFeed:   &feed.FeedSummary{UID: "1", Name: "Feed 1"},
		expectedStatus: http.StatusOK,
	}, {
		desc:           "failure: feed not found",
		token:          "valid-token",
		authSuccess:    true,
		fuid:           "2",
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1"}},
		expectedFeed:   nil,
		expectedStatus: http.StatusNotFound,
	}, {
		desc:           "failure: authentication with invalid cookie",
		token:          "invalid-token",
		authSuccess:    false,
		fuid:           "1",
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1"}},
		expectedFeed:   nil,
		expectedStatus: http.StatusUnauthorized,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			feedList := &mockFeedLister{feeds: test.feeds}
			handlerParams := &web.HandlerParams{
				FeedList:    feedList,
				AccessToken: "valid-token",
				SessionKey:  []byte("test-session-key"),
			}
			h := web.NewHandler(handlerParams)

			cookie := performLogin(t, h, performLoginParams{
				Token:         test.token,
				ExpectSuccess: test.authSuccess,
			})

			req, _ := http.NewRequest("GET", "/api/feeds/"+test.fuid, nil)
			if cookie != nil {
				req.AddCookie(cookie)
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)
			if rr.Header().Get("Content-Type") == "" {
				t.Errorf("expected Content-Type header to be set")
			}
			if rr.Header().Get("Content-Security-Policy") == "" {
				t.Errorf("expected Content-Security-Policy header to be set")
			}

			if rr.Code != test.expectedStatus {
				t.Errorf("expected status %v, got %v", test.expectedStatus, rr.Code)
			}

			if test.authSuccess && test.expectedFeed != nil {
				var f feed.FeedSummary
				err := json.NewDecoder(rr.Body).Decode(&f)
				if err != nil {
					t.Fatalf("could not decode response: %v", err)
				}
				compareFeeds(t, []*feed.FeedSummary{test.expectedFeed}, []*feed.FeedSummary{&f})
			}
		})
	}
}

func TestGetItem(t *testing.T) {
	tests := []struct {
		desc           string
		feeds          []*feed.FeedSummary
		fuid           string
		iuid           string
		expectedItem   *feed.ItemSummary
		token          string
		expectedStatus int
		authSuccess    bool
	}{{
		desc:           "success: item found",
		token:          "valid-token",
		authSuccess:    true,
		fuid:           "1",
		iuid:           "1",
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1", Items: []*feed.ItemSummary{{UID: "1", Title: "Item 1"}}}},
		expectedItem:   &feed.ItemSummary{UID: "1", Title: "Item 1"},
		expectedStatus: http.StatusOK,
	}, {
		desc:           "failure: item not found",
		token:          "valid-token",
		authSuccess:    true,
		fuid:           "1",
		iuid:           "2",
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1", Items: []*feed.ItemSummary{{UID: "1", Title: "Item 1"}}}},
		expectedItem:   nil,
		expectedStatus: http.StatusNotFound,
	}, {
		desc:           "failure: authentication with invalid cookie",
		token:          "invalid-token",
		authSuccess:    false,
		fuid:           "1",
		iuid:           "1",
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1", Items: []*feed.ItemSummary{{UID: "1", Title: "Item 1"}}}},
		expectedItem:   nil,
		expectedStatus: http.StatusUnauthorized,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			feedList := &mockFeedLister{feeds: test.feeds}
			handlerParams := &web.HandlerParams{
				FeedList:    feedList,
				AccessToken: "valid-token",
				SessionKey:  []byte("test-session-key"),
			}
			h := web.NewHandler(handlerParams)

			cookie := performLogin(t, h, performLoginParams{
				Token:         test.token,
				ExpectSuccess: test.authSuccess,
			})

			req, _ := http.NewRequest("GET", "/api/feeds/"+test.fuid+"/items/"+test.iuid, nil)
			if cookie != nil {
				req.AddCookie(cookie)
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)
			if rr.Header().Get("Content-Type") == "" {
				t.Errorf("expected Content-Type header to be set")
			}
			if rr.Header().Get("Content-Security-Policy") == "" {
				t.Errorf("expected Content-Security-Policy header to be set")
			}

			if rr.Code != test.expectedStatus {
				t.Errorf("expected status %v, got %v", test.expectedStatus, rr.Code)
			}

			if test.authSuccess && test.expectedItem != nil {
				var item feed.ItemSummary
				err := json.NewDecoder(rr.Body).Decode(&item)
				if err != nil {
					t.Fatalf("could not decode response: %v", err)
				}
				compareItems(t, []*feed.ItemSummary{test.expectedItem}, []*feed.ItemSummary{&item})
			}
		})
	}
}

func TestMarkAsRead(t *testing.T) {
	tests := []struct {
		desc           string
		feeds          []*feed.FeedSummary
		fuid           string
		iuid           string
		before         int64
		token          string
		expectedStatus int
		authSuccess    bool
		expectedItems  []*feed.ItemSummary
	}{{
		desc:           "success: item marked as read",
		token:          "valid-token",
		authSuccess:    true,
		fuid:           "1",
		iuid:           "1",
		before:         timeutil.Now(),
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1", Items: []*feed.ItemSummary{{UID: "1", Title: "Item 1", Timestamp: timeutil.Now()}}}},
		expectedStatus: http.StatusOK,
		expectedItems:  []*feed.ItemSummary{{UID: "1", Title: "Item 1", Timestamp: timeutil.Now(), Read: true}},
	}, {
		desc:           "failure: item not found",
		token:          "valid-token",
		authSuccess:    true,
		fuid:           "1",
		iuid:           "2",
		before:         timeutil.Now(),
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1", Items: []*feed.ItemSummary{{UID: "1", Title: "Item 1", Timestamp: timeutil.Now()}}}},
		expectedStatus: http.StatusNotFound,
		expectedItems:  []*feed.ItemSummary{{UID: "1", Title: "Item 1", Timestamp: timeutil.Now(), Read: false}},
	}, {
		desc:           "failure: authentication with invalid cookie",
		token:          "invalid-token",
		authSuccess:    false,
		fuid:           "1",
		iuid:           "1",
		before:         timeutil.Now(),
		feeds:          []*feed.FeedSummary{{UID: "1", Name: "Feed 1", Items: []*feed.ItemSummary{{UID: "1", Title: "Item 1", Timestamp: timeutil.Now()}}}},
		expectedStatus: http.StatusUnauthorized,
		expectedItems:  []*feed.ItemSummary{{UID: "1", Title: "Item 1", Timestamp: timeutil.Now(), Read: false}},
	}, {
		desc:        "success: some feed items marked as read",
		token:       "valid-token",
		authSuccess: true,
		fuid:        "1",
		iuid:        "",
		before:      timeutil.Now() - 1,
		feeds: []*feed.FeedSummary{{UID: "1", Name: "Feed 1", Items: []*feed.ItemSummary{
			{UID: "1", Title: "Item 1", Timestamp: timeutil.Now() - 2, Read: false},
			{UID: "2", Title: "Item 2", Timestamp: timeutil.Now(), Read: false},
		}}},
		expectedStatus: http.StatusOK,
		expectedItems: []*feed.ItemSummary{
			{UID: "1", Title: "Item 1", Timestamp: timeutil.Now() - 2, Read: true},
			{UID: "2", Title: "Item 2", Timestamp: timeutil.Now(), Read: false},
		},
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			feedList := &mockFeedLister{feeds: test.feeds}
			handlerParams := &web.HandlerParams{
				FeedList:    feedList,
				AccessToken: "valid-token",
				SessionKey:  []byte("test-session-key"),
			}
			h := web.NewHandler(handlerParams)

			cookie := performLogin(t, h, performLoginParams{
				Token:         test.token,
				ExpectSuccess: test.authSuccess,
			})

			reqURL := "/api/feeds/" + test.fuid + "/items/" + test.iuid + "/read"
			if test.iuid == "" {
				reqURL = "/api/feeds/" + test.fuid + "/read"
			}
			req, _ := http.NewRequest("POST", reqURL, bytes.NewBufferString(fmt.Sprintf(`{"before": %d}`, test.before)))
			if cookie != nil {
				req.AddCookie(cookie)
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)
			if rr.Header().Get("Content-Security-Policy") == "" {
				t.Errorf("expected Content-Security-Policy header to be set")
			}

			if rr.Code != test.expectedStatus {
				t.Errorf("expected status %v, got %v", test.expectedStatus, rr.Code)
			}

			if test.authSuccess {
				compareItems(t, test.expectedItems, feedList.FeedSummary(test.fuid).Items)
			}
		})
	}
}

func TestStatic(t *testing.T) {
	tests := []struct {
		desc           string
		url            string
		expectedStatus int
	}{{
		desc:           "GET /",
		url:            "/",
		expectedStatus: http.StatusOK,
	}, {
		desc:           "GET /index.html",
		url:            "/index.html",
		expectedStatus: http.StatusMovedPermanently,
	}, {
		desc:           "GET /whatever",
		url:            "/",
		expectedStatus: http.StatusOK,
	}, {
		desc:           "GET /static/varys.css",
		url:            "/static/varys.css",
		expectedStatus: http.StatusOK,
	}, {
		desc:           "GET /static/varys.js",
		url:            "/static/varys.js",
		expectedStatus: http.StatusOK,
	}, {
		desc:           "GET /static/icon.svg",
		url:            "/static/icon.svg",
		expectedStatus: http.StatusOK,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			feedList := &mockFeedLister{}
			handlerParams := &web.HandlerParams{
				FeedList:    feedList,
				AccessToken: "valid-token",
				SessionKey:  []byte("test-session-key"),
			}
			h := web.NewHandler(handlerParams)

			req, _ := http.NewRequest("GET", test.url, nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)
			if rr.Header().Get("Content-Security-Policy") == "" {
				t.Errorf("expected Content-Security-Policy header to be set")
			}

			if rr.Code != test.expectedStatus {
				t.Errorf("expected status %v, got %v", test.expectedStatus, rr.Code)
			}

			if test.expectedStatus == http.StatusOK && rr.Body.Len() == 0 {
				t.Errorf("expected non-empty response body")
			}

			if rr.Header().Get("Cache-Control") != "max-age=86400" {
				t.Errorf("expected Cache-Control header to be 'max-age=86400', got %v", rr.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestStatus(t *testing.T) {
	feedList := &mockFeedLister{}
	handlerParams := &web.HandlerParams{
		FeedList:    feedList,
		AccessToken: "valid-token",
		SessionKey:  []byte("test-session-key"),
	}
	h := web.NewHandler(handlerParams)

	req, _ := http.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, rr.Code)
	}

	var status struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	err := json.NewDecoder(rr.Body).Decode(&status)
	if err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if status.Status != "ok" {
		t.Errorf("expected status 'ok', got %v", status.Status)
	}

	if status.Version == "" {
		t.Errorf("expected non-empty version")
	}
}
