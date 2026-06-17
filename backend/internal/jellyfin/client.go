package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a typed Jellyfin API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(host, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(host, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := fmt.Sprintf("%s%s", c.baseURL, path)
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Token", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetLibraries returns all virtual folders for the given admin user.
func (c *Client) GetLibraries(ctx context.Context, userId string) ([]Library, error) {
	var res LibrariesResponse
	err := c.get(ctx, fmt.Sprintf("/Users/%s/Views", userId), nil, &res)
	return res.Items, err
}

// GetUsers returns all users.
func (c *Client) GetUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := c.get(ctx, "/Users", nil, &users)
	return users, err
}

// GetItems fetches library items with pagination.
func (c *Client) GetItems(ctx context.Context, params ItemsParams) (*ItemsResponse, error) {
	q := url.Values{}
	q.Set("ParentId", params.ParentId)
	q.Set("IncludeItemTypes", strings.Join(params.IncludeTypes, ","))
	q.Set("Fields", strings.Join(params.Fields, ","))
	q.Set("Recursive", "true")
	q.Set("StartIndex", fmt.Sprint(params.StartIndex))
	q.Set("Limit", fmt.Sprint(params.Limit))
	if params.SortBy != "" {
		q.Set("SortBy", params.SortBy)
	}

	var res ItemsResponse
	err := c.get(ctx, "/Items", q, &res)
	return &res, err
}

// GetAllItems fetches all items of the given types from a library by paginating automatically.
func (c *Client) GetAllItems(ctx context.Context, libraryId string, types []string, fields []string) ([]Item, error) {
	const pageSize = 500
	var all []Item
	start := 0

	for {
		res, err := c.GetItems(ctx, ItemsParams{
			ParentId:     libraryId,
			IncludeTypes: types,
			Fields:       fields,
			StartIndex:   start,
			Limit:        pageSize,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res.Items...)
		start += len(res.Items)
		if start >= res.TotalRecordCount {
			break
		}
	}
	return all, nil
}

// GetSessions returns currently active sessions.
func (c *Client) GetSessions(ctx context.Context) ([]SessionInfo, error) {
	var sessions []SessionInfo
	err := c.get(ctx, "/Sessions", url.Values{"ControllableByUserId": []string{""}}, &sessions)
	return sessions, err
}

// AuthenticateUser authenticates a user by username/password and returns the token.
func (c *Client) AuthenticateUser(ctx context.Context, username, password string) (*AuthResponse, error) {
	body := fmt.Sprintf(`{"Username":%q,"Pw":%q}`, username, password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/Users/AuthenticateByName",
		strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Authorization",
		`MediaBrowser Client="Jellystics", Device="Server", DeviceId="jellystics-server", Version="1.0"`)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid credentials")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("authenticate: status %d", resp.StatusCode)
	}
	var auth AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, err
	}
	return &auth, nil
}


// ItemsParams controls GetItems / GetAllItems behaviour.
type ItemsParams struct {
	ParentId     string
	IncludeTypes []string
	Fields       []string
	StartIndex   int
	Limit        int
	SortBy       string
}

// StandardFields are the Jellyfin item fields we always request.
var StandardFields = []string{
	"DateCreated", "Genres", "MediaSources", "MediaStreams",
	"Overview", "ParentId", "Path", "PremiereDate", "ProductionYear",
	"RunTimeTicks", "SortName", "Status", "CommunityRating",
	"ImageBlurHashes", "AlbumArtist", "AlbumArtists", "ArtistItems",
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("POST %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetInstalledPlugins returns the list of installed Jellyfin plugins.
func (c *Client) GetInstalledPlugins(ctx context.Context) ([]Plugin, error) {
	var plugins []Plugin
	err := c.get(ctx, "/Plugins", nil, &plugins)
	return plugins, err
}

// GetItem fetches a single Jellyfin item by ID.
// Endpoint: GET /Items/{id}
func (c *Client) GetItem(ctx context.Context, id string, fields []string) (*Item, error) {
	q := url.Values{}
	if len(fields) > 0 {
		q.Set("Fields", strings.Join(fields, ","))
	}
	var res ItemsResponse
	err := c.get(ctx, fmt.Sprintf("/Items?Ids=%s", id), q, &res)
	if err != nil {
		return nil, err
	}
	if len(res.Items) == 0 {
		return nil, fmt.Errorf("item %s not found", id)
	}
	return &res.Items[0], nil
}

// GetRecentlyAdded fetches the latest items for a given user and library.
// Endpoint: GET /Users/{userId}/Items/Latest
func (c *Client) GetRecentlyAdded(ctx context.Context, userId, libraryId string, limit int, fields []string) ([]Item, error) {
	q := url.Values{}
	q.Set("Limit", fmt.Sprint(limit))
	if libraryId != "" {
		q.Set("ParentId", libraryId)
	}
	if len(fields) > 0 {
		q.Set("Fields", strings.Join(fields, ","))
	}

	var items []Item
	err := c.get(ctx, fmt.Sprintf("/Users/%s/Items/Latest", userId), q, &items)
	return items, err
}

// GetSystemInfo fetches Jellyfin system info from a given base URL and API key.
// This is a standalone function (not a method) because it is used before the client is configured.
func GetSystemInfo(ctx context.Context, baseURL, apiKey string) (map[string]interface{}, int, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	u := strings.TrimRight(baseURL, "/") + "/System/Info"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Emby-Token", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("status %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, resp.StatusCode, err
	}
	return info, resp.StatusCode, nil
}

// SubmitCustomQuery posts a SQL query to the PlaybackReporting plugin and returns the raw rows.
// Endpoint: POST /user_usage_stats/submit_custom_query
func (c *Client) SubmitCustomQuery(ctx context.Context, query string) ([]PlaybackReportingRow, error) {
	payload := map[string]string{"CustomQueryString": query}
	var res CustomQueryResponse
	if err := c.post(ctx, "/user_usage_stats/submit_custom_query", payload, &res); err != nil {
		return nil, err
	}
	return res.Results, nil
}
