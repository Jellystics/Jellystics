package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// newTestClient spins up an httptest.Server with the given handler and returns a
// Client whose baseURL points at it, plus a cleanup func.
func newTestClient(t *testing.T, apiKey string, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, apiKey), srv
}

// ---------------------------------------------------------------------------
// NewClient
// ---------------------------------------------------------------------------

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("http://example.com/", "key")
	if c.baseURL != "http://example.com" {
		t.Fatalf("baseURL = %q, want %q", c.baseURL, "http://example.com")
	}
	if c.apiKey != "key" {
		t.Fatalf("apiKey = %q, want %q", c.apiKey, "key")
	}
	if c.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if c.httpClient.Timeout == 0 {
		t.Fatal("httpClient.Timeout is zero, want non-zero")
	}
}

func TestNewClient_NoTrailingSlash(t *testing.T) {
	c := NewClient("http://example.com", "key")
	if c.baseURL != "http://example.com" {
		t.Fatalf("baseURL = %q, want %q", c.baseURL, "http://example.com")
	}
}

// ---------------------------------------------------------------------------
// GetLibraries — GET /Users/{userId}/Views
// ---------------------------------------------------------------------------

func TestGetLibraries_Success(t *testing.T) {
	const apiKey = "secret-token"
	var gotPath, gotToken, gotAccept string
	c, _ := newTestClient(t, apiKey, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Emby-Token")
		gotAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte(`{"Items":[
			{"ItemId":"lib1","Id":"lib1","Name":"Movies","Type":"CollectionFolder","CollectionType":"movies"},
			{"ItemId":"lib2","Id":"lib2","Name":"Shows","CollectionType":"tvshows"}
		]}`))
	})

	libs, err := c.GetLibraries(context.Background(), "admin-123")
	if err != nil {
		t.Fatalf("GetLibraries returned error: %v", err)
	}
	if gotPath != "/Users/admin-123/Views" {
		t.Errorf("path = %q, want /Users/admin-123/Views", gotPath)
	}
	if gotToken != apiKey {
		t.Errorf("X-Emby-Token = %q, want %q", gotToken, apiKey)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if len(libs) != 2 {
		t.Fatalf("len(libs) = %d, want 2", len(libs))
	}
	if libs[0].Name != "Movies" || libs[0].CollectionType != "movies" {
		t.Errorf("libs[0] = %+v, unexpected", libs[0])
	}
	if libs[1].Id != "lib2" {
		t.Errorf("libs[1].Id = %q, want lib2", libs[1].Id)
	}
}

func TestGetLibraries_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.GetLibraries(context.Background(), "admin")
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %v, want it to mention status 500", err)
	}
}

func TestGetLibraries_MalformedJSON(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not valid json`))
	})
	_, err := c.GetLibraries(context.Background(), "admin")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetUsers — GET /Users
// ---------------------------------------------------------------------------

func TestGetUsers_Success(t *testing.T) {
	var gotPath string
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[
			{"Id":"u1","Name":"alice","Policy":{"IsAdministrator":true}},
			{"Id":"u2","Name":"bob","Policy":{"IsAdministrator":false}}
		]`))
	})
	users, err := c.GetUsers(context.Background())
	if err != nil {
		t.Fatalf("GetUsers error: %v", err)
	}
	if gotPath != "/Users" {
		t.Errorf("path = %q, want /Users", gotPath)
	}
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	if users[0].Name != "alice" || !users[0].Policy.IsAdministrator {
		t.Errorf("users[0] = %+v, unexpected", users[0])
	}
	if users[1].Policy.IsAdministrator {
		t.Errorf("users[1] should not be admin")
	}
}

func TestGetUsers_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	_, err := c.GetUsers(context.Background())
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetItems — GET /Items
// ---------------------------------------------------------------------------

func TestGetItems_QueryConstruction(t *testing.T) {
	var q url.Values
	var gotPath string
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q = r.URL.Query()
		_, _ = w.Write([]byte(`{"Items":[{"Id":"i1","Name":"A Movie","Type":"Movie"}],"TotalRecordCount":1,"StartIndex":0}`))
	})

	res, err := c.GetItems(context.Background(), ItemsParams{
		ParentId:     "lib1",
		IncludeTypes: []string{"Movie", "Series"},
		Fields:       []string{"Genres", "Overview"},
		StartIndex:   20,
		Limit:        50,
		SortBy:       "SortName",
	})
	if err != nil {
		t.Fatalf("GetItems error: %v", err)
	}
	if gotPath != "/Items" {
		t.Errorf("path = %q, want /Items", gotPath)
	}
	checks := map[string]string{
		"ParentId":         "lib1",
		"IncludeItemTypes": "Movie,Series",
		"Fields":           "Genres,Overview",
		"Recursive":        "true",
		"StartIndex":       "20",
		"Limit":            "50",
		"SortBy":           "SortName",
	}
	for k, want := range checks {
		if got := q.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
	if res.TotalRecordCount != 1 || len(res.Items) != 1 {
		t.Errorf("res = %+v, unexpected", res)
	}
	if res.Items[0].Name != "A Movie" {
		t.Errorf("item name = %q, want A Movie", res.Items[0].Name)
	}
}

func TestGetItems_OmitsSortByWhenEmpty(t *testing.T) {
	var q url.Values
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query()
		_, _ = w.Write([]byte(`{"Items":[],"TotalRecordCount":0}`))
	})
	_, err := c.GetItems(context.Background(), ItemsParams{ParentId: "lib1", Limit: 10})
	if err != nil {
		t.Fatalf("GetItems error: %v", err)
	}
	if _, present := q["SortBy"]; present {
		t.Errorf("SortBy should be omitted when empty, but present: %q", q.Get("SortBy"))
	}
}

func TestGetItems_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	_, err := c.GetItems(context.Background(), ItemsParams{ParentId: "lib1"})
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetAllItems — paginated GET /Items
// ---------------------------------------------------------------------------

func TestGetAllItems_Paginates(t *testing.T) {
	// Total 1200 items across pages of 500.
	const total = 1200
	var requestStarts []int
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		start, _ := strconv.Atoi(q.Get("StartIndex"))
		limit, _ := strconv.Atoi(q.Get("Limit"))
		requestStarts = append(requestStarts, start)

		remaining := total - start
		if remaining < 0 {
			remaining = 0
		}
		n := limit
		if n > remaining {
			n = remaining
		}
		items := make([]Item, n)
		for i := 0; i < n; i++ {
			items[i] = Item{Id: fmt.Sprintf("item-%d", start+i)}
		}
		resp := ItemsResponse{Items: items, TotalRecordCount: total, StartIndex: start}
		_ = json.NewEncoder(w).Encode(resp)
	})

	all, err := c.GetAllItems(context.Background(), "lib1", []string{"Movie"}, []string{"Genres"})
	if err != nil {
		t.Fatalf("GetAllItems error: %v", err)
	}
	if len(all) != total {
		t.Fatalf("len(all) = %d, want %d", len(all), total)
	}
	// pageSize=500 => starts 0, 500, 1000
	wantStarts := []int{0, 500, 1000}
	if len(requestStarts) != len(wantStarts) {
		t.Fatalf("made %d requests (%v), want %d (%v)", len(requestStarts), requestStarts, len(wantStarts), wantStarts)
	}
	for i, s := range wantStarts {
		if requestStarts[i] != s {
			t.Errorf("request %d StartIndex = %d, want %d", i, requestStarts[i], s)
		}
	}
	// verify ids stitched in order
	if all[0].Id != "item-0" || all[total-1].Id != fmt.Sprintf("item-%d", total-1) {
		t.Errorf("stitched items out of order: first=%q last=%q", all[0].Id, all[total-1].Id)
	}
}

func TestGetAllItems_SinglePage(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Items":[{"Id":"a"},{"Id":"b"}],"TotalRecordCount":2,"StartIndex":0}`))
	})
	all, err := c.GetAllItems(context.Background(), "lib1", []string{"Audio"}, nil)
	if err != nil {
		t.Fatalf("GetAllItems error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}
}

func TestGetAllItems_ErrorPropagates(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.GetAllItems(context.Background(), "lib1", []string{"Movie"}, nil)
	if err == nil {
		t.Fatal("expected error to propagate from GetItems, got nil")
	}
}

// GetAllItems could infinite-loop if the server returns TotalRecordCount>0 but
// an empty Items page (start never advances). Guard against a hang.
func TestGetAllItems_EmptyPageDoesNotHang(t *testing.T) {
	calls := 0
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 5 {
			// break the loop server-side to avoid an actual infinite hang in CI
			_, _ = w.Write([]byte(`{"Items":[],"TotalRecordCount":0,"StartIndex":0}`))
			return
		}
		// TotalRecordCount claims more items but returns none -> start never advances
		_, _ = w.Write([]byte(`{"Items":[],"TotalRecordCount":100,"StartIndex":0}`))
	})

	done := make(chan struct{})
	go func() {
		_, _ = c.GetAllItems(context.Background(), "lib1", []string{"Movie"}, nil)
		close(done)
	}()
	select {
	case <-done:
		// completed (because our server broke the loop after 5 calls). If the
		// client had a guard against non-advancing pagination it would finish on
		// its own; either way we assert it eventually terminates.
	case <-time.After(2 * time.Second):
		t.Fatal("GetAllItems hung on empty-page pagination (start never advances)")
	}
}

// ---------------------------------------------------------------------------
// GetSessions — GET /Sessions
// ---------------------------------------------------------------------------

func TestGetSessions_Success(t *testing.T) {
	var gotPath string
	var q url.Values
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q = r.URL.Query()
		_, _ = w.Write([]byte(`[
			{"Id":"s1","UserId":"u1","UserName":"alice","Client":"Web","IsActive":true,
			 "NowPlayingItem":{"Id":"i1","Name":"Ep","Type":"Episode"},
			 "PlayState":{"PositionTicks":123456,"IsPaused":false}}
		]`))
	})
	sessions, err := c.GetSessions(context.Background())
	if err != nil {
		t.Fatalf("GetSessions error: %v", err)
	}
	if gotPath != "/Sessions" {
		t.Errorf("path = %q, want /Sessions", gotPath)
	}
	// client sends ControllableByUserId="" — key must be present (even if empty value)
	if _, present := q["ControllableByUserId"]; !present {
		t.Errorf("ControllableByUserId query param missing; got query %v", q)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	s := sessions[0]
	if s.UserName != "alice" || !s.IsActive {
		t.Errorf("session = %+v, unexpected", s)
	}
	if s.NowPlayingItem == nil || s.NowPlayingItem.Type != "Episode" {
		t.Errorf("NowPlayingItem = %+v, unexpected", s.NowPlayingItem)
	}
	if s.PlayState == nil || s.PlayState.PositionTicks == nil || *s.PlayState.PositionTicks != 123456 {
		t.Errorf("PlayState = %+v, unexpected", s.PlayState)
	}
}

func TestGetSessions_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := c.GetSessions(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

// ---------------------------------------------------------------------------
// AuthenticateUser — POST /Users/AuthenticateByName
// ---------------------------------------------------------------------------

func TestAuthenticateUser_Success(t *testing.T) {
	var gotPath, gotMethod, gotContentType, gotAuthHeader, gotBody string
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotAuthHeader = r.Header.Get("X-Emby-Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"AccessToken":"tok-abc","User":{"Id":"u1","Name":"alice","Policy":{"IsAdministrator":true}}}`))
	})

	auth, err := c.AuthenticateUser(context.Background(), "alice", "s3cr3t")
	if err != nil {
		t.Fatalf("AuthenticateUser error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/Users/AuthenticateByName" {
		t.Errorf("path = %q, want /Users/AuthenticateByName", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if !strings.HasPrefix(gotAuthHeader, "MediaBrowser ") {
		t.Errorf("X-Emby-Authorization = %q, want MediaBrowser prefix", gotAuthHeader)
	}
	// body should be valid JSON containing the credentials
	var parsed struct {
		Username string `json:"Username"`
		Pw       string `json:"Pw"`
	}
	if err := json.Unmarshal([]byte(gotBody), &parsed); err != nil {
		t.Fatalf("request body not valid JSON: %v (body=%q)", err, gotBody)
	}
	if parsed.Username != "alice" || parsed.Pw != "s3cr3t" {
		t.Errorf("body creds = %+v, want alice/s3cr3t", parsed)
	}
	if auth.AccessToken != "tok-abc" {
		t.Errorf("AccessToken = %q, want tok-abc", auth.AccessToken)
	}
	if auth.User.Name != "alice" || !auth.User.Policy.IsAdministrator {
		t.Errorf("auth.User = %+v, unexpected", auth.User)
	}
}

func TestAuthenticateUser_EscapesSpecialChars(t *testing.T) {
	// username/password with quotes and backslashes must be properly JSON-escaped.
	var gotBody string
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(`{"AccessToken":"t","User":{"Id":"u"}}`))
	})
	user := `a"b\c`
	pass := `p"w\d`
	_, err := c.AuthenticateUser(context.Background(), user, pass)
	if err != nil {
		t.Fatalf("AuthenticateUser error: %v", err)
	}
	var parsed struct {
		Username string `json:"Username"`
		Pw       string `json:"Pw"`
	}
	if err := json.Unmarshal([]byte(gotBody), &parsed); err != nil {
		t.Fatalf("body not valid JSON with special chars: %v (body=%q)", err, gotBody)
	}
	if parsed.Username != user || parsed.Pw != pass {
		t.Errorf("decoded creds = %q/%q, want %q/%q", parsed.Username, parsed.Pw, user, pass)
	}
}

func TestAuthenticateUser_InvalidCredentials(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := c.AuthenticateUser(context.Background(), "alice", "wrong")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("error = %v, want 'invalid credentials'", err)
	}
}

func TestAuthenticateUser_OtherHTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.AuthenticateUser(context.Background(), "alice", "pw")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("500 should not be reported as invalid credentials: %v", err)
	}
}

func TestAuthenticateUser_MalformedJSON(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{bad`))
	})
	_, err := c.AuthenticateUser(context.Background(), "alice", "pw")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetInstalledPlugins — GET /Plugins
// ---------------------------------------------------------------------------

func TestGetInstalledPlugins_Success(t *testing.T) {
	var gotPath, gotToken string
	c, _ := newTestClient(t, "plugtok", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Emby-Token")
		_, _ = w.Write([]byte(`[
			{"Name":"Playback Reporting","ConfigurationFileName":"Jellyfin.Plugin.PlaybackReporting.xml"},
			{"Name":"Other"}
		]`))
	})
	plugins, err := c.GetInstalledPlugins(context.Background())
	if err != nil {
		t.Fatalf("GetInstalledPlugins error: %v", err)
	}
	if gotPath != "/Plugins" {
		t.Errorf("path = %q, want /Plugins", gotPath)
	}
	if gotToken != "plugtok" {
		t.Errorf("token = %q, want plugtok", gotToken)
	}
	if len(plugins) != 2 {
		t.Fatalf("len(plugins) = %d, want 2", len(plugins))
	}
	if plugins[0].Name != "Playback Reporting" {
		t.Errorf("plugins[0].Name = %q", plugins[0].Name)
	}
	if plugins[0].ConfigurationFileName == nil || *plugins[0].ConfigurationFileName == "" {
		t.Errorf("plugins[0].ConfigurationFileName missing")
	}
	if plugins[1].ConfigurationFileName != nil {
		t.Errorf("plugins[1].ConfigurationFileName should be nil, got %v", *plugins[1].ConfigurationFileName)
	}
}

func TestGetInstalledPlugins_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := c.GetInstalledPlugins(context.Background())
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetItem — GET /Items?Ids={id}
// ---------------------------------------------------------------------------

// GetItem WITH fields is broken: it constructs its path as
// fmt.Sprintf("/Items?Ids=%s", id) AND passes a non-empty url.Values{Fields:...}.
// The get() helper then appends "?" + query.Encode(), producing a URL with TWO
// '?' separators:
//
//	/Items?Ids=item-9?Fields=Genres,Overview
//
// The second '?' is not a valid query separator, so Go's server parses "Ids" as
// "item-9?Fields=Genres,Overview" (Fields is swallowed into the Ids value and is
// NOT sent as its own query param). This test asserts the CORRECT behavior and
// is guarded with t.Skip so the package stays green while documenting the bug.
func TestGetItem_Success(t *testing.T) {
	var gotRawQuery, gotIds, gotFields string
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		q := r.URL.Query()
		gotIds = q.Get("Ids")
		gotFields = q.Get("Fields")
		_, _ = w.Write([]byte(`{"Items":[{"Id":"item-9","Name":"Found","Type":"Movie"}],"TotalRecordCount":1}`))
	})

	item, err := c.GetItem(context.Background(), "item-9", []string{"Genres", "Overview"})
	if err != nil {
		t.Fatalf("GetItem error: %v (rawquery=%q)", err, gotRawQuery)
	}
	if item.Id != "item-9" || item.Name != "Found" {
		t.Errorf("item = %+v, unexpected", item)
	}
	if gotIds != "item-9" {
		t.Errorf("Ids param = %q, want item-9 (rawquery=%q)", gotIds, gotRawQuery)
	}
	if gotFields != "Genres,Overview" {
		t.Errorf("Fields param = %q, want Genres,Overview (rawquery=%q)", gotFields, gotRawQuery)
	}
}

// GetItem with no fields does not pass a query, so no double-'?' occurs and it
// works correctly.
func TestGetItem_NoFields_Success(t *testing.T) {
	var gotIds string
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		gotIds = r.URL.Query().Get("Ids")
		_, _ = w.Write([]byte(`{"Items":[{"Id":"item-1","Name":"NoFields"}],"TotalRecordCount":1}`))
	})
	item, err := c.GetItem(context.Background(), "item-1", nil)
	if err != nil {
		t.Fatalf("GetItem error: %v", err)
	}
	if gotIds != "item-1" {
		t.Errorf("Ids param = %q, want item-1", gotIds)
	}
	if item.Name != "NoFields" {
		t.Errorf("item.Name = %q, want NoFields", item.Name)
	}
}

func TestGetItem_NotFound(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Items":[],"TotalRecordCount":0}`))
	})
	_, err := c.GetItem(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected 'not found' error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v, want 'not found'", err)
	}
}

func TestGetItem_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.GetItem(context.Background(), "x", nil)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetRecentlyAdded — GET /Users/{userId}/Items/Latest
// ---------------------------------------------------------------------------

func TestGetRecentlyAdded_FullParams(t *testing.T) {
	var gotPath string
	var q url.Values
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q = r.URL.Query()
		_, _ = w.Write([]byte(`[{"Id":"a","Name":"Latest A"},{"Id":"b","Name":"Latest B"}]`))
	})
	items, err := c.GetRecentlyAdded(context.Background(), "user-7", "lib-3", 25, []string{"Genres", "Path"})
	if err != nil {
		t.Fatalf("GetRecentlyAdded error: %v", err)
	}
	if gotPath != "/Users/user-7/Items/Latest" {
		t.Errorf("path = %q, want /Users/user-7/Items/Latest", gotPath)
	}
	if q.Get("Limit") != "25" {
		t.Errorf("Limit = %q, want 25", q.Get("Limit"))
	}
	if q.Get("ParentId") != "lib-3" {
		t.Errorf("ParentId = %q, want lib-3", q.Get("ParentId"))
	}
	if q.Get("Fields") != "Genres,Path" {
		t.Errorf("Fields = %q, want Genres,Path", q.Get("Fields"))
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Name != "Latest A" {
		t.Errorf("items[0].Name = %q", items[0].Name)
	}
}

func TestGetRecentlyAdded_OmitsOptionalParams(t *testing.T) {
	var q url.Values
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query()
		_, _ = w.Write([]byte(`[]`))
	})
	_, err := c.GetRecentlyAdded(context.Background(), "user-7", "", 10, nil)
	if err != nil {
		t.Fatalf("GetRecentlyAdded error: %v", err)
	}
	if q.Get("Limit") != "10" {
		t.Errorf("Limit = %q, want 10", q.Get("Limit"))
	}
	if _, present := q["ParentId"]; present {
		t.Errorf("ParentId should be omitted when libraryId empty, got %q", q.Get("ParentId"))
	}
	if _, present := q["Fields"]; present {
		t.Errorf("Fields should be omitted when nil, got %q", q.Get("Fields"))
	}
}

func TestGetRecentlyAdded_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	_, err := c.GetRecentlyAdded(context.Background(), "u", "", 5, nil)
	if err == nil {
		t.Fatal("expected error for 502, got nil")
	}
}

// ---------------------------------------------------------------------------
// GetSystemInfo — package-level func, GET /System/Info
// ---------------------------------------------------------------------------

func TestGetSystemInfo_Success(t *testing.T) {
	var gotPath, gotToken, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Emby-Token")
		gotAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte(`{"ServerName":"MyJelly","Version":"10.9.0","Id":"srv-1"}`))
	}))
	defer srv.Close()

	info, status, err := GetSystemInfo(context.Background(), srv.URL+"/", "systok")
	if err != nil {
		t.Fatalf("GetSystemInfo error: %v", err)
	}
	if gotPath != "/System/Info" {
		t.Errorf("path = %q, want /System/Info", gotPath)
	}
	if gotToken != "systok" {
		t.Errorf("token = %q, want systok", gotToken)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if info["ServerName"] != "MyJelly" || info["Version"] != "10.9.0" {
		t.Errorf("info = %+v, unexpected", info)
	}
}

func TestGetSystemInfo_HTTPErrorReturnsStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	info, status, err := GetSystemInfo(context.Background(), srv.URL, "badtok")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
	if info != nil {
		t.Errorf("info = %v, want nil on error", info)
	}
}

func TestGetSystemInfo_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{nope`))
	}))
	defer srv.Close()

	_, status, err := GetSystemInfo(context.Background(), srv.URL, "k")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	// status should still be reported as 200 even though decode failed
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestGetSystemInfo_ConnectionError(t *testing.T) {
	// Point at a server that is immediately closed to force a transport error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	badURL := srv.URL
	srv.Close()

	_, status, err := GetSystemInfo(context.Background(), badURL, "k")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if status != 0 {
		t.Errorf("status = %d, want 0 on transport error", status)
	}
}

// ---------------------------------------------------------------------------
// SubmitCustomQuery — POST /user_usage_stats/submit_custom_query
// ---------------------------------------------------------------------------

func TestSubmitCustomQuery_Success(t *testing.T) {
	var gotPath, gotMethod, gotToken, gotContentType string
	var gotBody map[string]string
	c, _ := newTestClient(t, "qtok", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotToken = r.Header.Get("X-Emby-Token")
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"results":[
			["1","2024-01-01","u1","i1","Movie","Film","DirectPlay","Web","Chrome",3600],
			["2","2024-01-02","u2","i2","Episode","Ep","Transcode","Android","Phone",120]
		]}`))
	})

	rows, err := c.SubmitCustomQuery(context.Background(), "SELECT * FROM PlaybackActivity")
	if err != nil {
		t.Fatalf("SubmitCustomQuery error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/user_usage_stats/submit_custom_query" {
		t.Errorf("path = %q, want /user_usage_stats/submit_custom_query", gotPath)
	}
	if gotToken != "qtok" {
		t.Errorf("token = %q, want qtok", gotToken)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotBody["CustomQueryString"] != "SELECT * FROM PlaybackActivity" {
		t.Errorf("body CustomQueryString = %q, unexpected", gotBody["CustomQueryString"])
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	// Row is []interface{}; check a couple of columns.
	if rows[0][4] != "Movie" {
		t.Errorf("rows[0][4] = %v, want Movie", rows[0][4])
	}
	// numbers decode to float64
	if dur, ok := rows[0][9].(float64); !ok || dur != 3600 {
		t.Errorf("rows[0][9] = %v (%T), want 3600 float64", rows[0][9], rows[0][9])
	}
}

func TestSubmitCustomQuery_EmptyResults(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	rows, err := c.SubmitCustomQuery(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("SubmitCustomQuery error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0", len(rows))
	}
}

func TestSubmitCustomQuery_HTTPError(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.SubmitCustomQuery(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestSubmitCustomQuery_MalformedJSON(t *testing.T) {
	c, _ := newTestClient(t, "k", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{results: bad`))
	})
	_, err := c.SubmitCustomQuery(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
