package confluence

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Page represents a Confluence page.
type Page struct {
	ID      string
	Title   string
	Version int
}

type pageJSON struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
}

func (p pageJSON) toPage() *Page {
	return &Page{ID: p.ID, Title: p.Title, Version: p.Version.Number}
}

// SpaceID resolves a human-readable space key to its numeric space ID, which the v2 API
// requires for page operations.
func (c *Client) SpaceID(ctx context.Context, spaceKey string) (string, error) {
	u := c.v2("/spaces") + "?" + url.Values{"keys": {spaceKey}}.Encode()
	var resp struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if _, err := c.doJSON(ctx, http.MethodGet, u, nil, &resp); err != nil {
		return "", err
	}
	if len(resp.Results) == 0 {
		return "", fmt.Errorf("confluence: space %q not found", spaceKey)
	}
	return resp.Results[0].ID, nil
}

// CreatePageInput holds the parameters for creating a page.
type CreatePageInput struct {
	SpaceID  string
	Title    string
	ParentID string // when empty, the page is created at the space root
	Body     string // storage format XHTML
}

// UpdatePageInput holds the parameters for updating a page.
type UpdatePageInput struct {
	ID      string
	Title   string
	Body    string // storage format XHTML
	Version int    // the current version; the request is sent with number Version+1
}

// FindPage looks up a page by title within a space. If no matching page exists it returns
// (nil, nil); a 404 from the API is also treated as "not found" rather than an error, so
// callers can use this to decide between creating and updating a page.
func (c *Client) FindPage(ctx context.Context, spaceID, title string) (*Page, error) {
	u := c.v2("/pages") + "?" + url.Values{"space-id": {spaceID}, "title": {title}}.Encode()
	var resp struct {
		Results []pageJSON `json:"results"`
	}
	status, err := c.doJSON(ctx, http.MethodGet, u, nil, &resp)
	if err != nil {
		if status == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	if len(resp.Results) == 0 {
		return nil, nil
	}
	return resp.Results[0].toPage(), nil
}

// CreatePage creates a new page from the given input.
func (c *Client) CreatePage(ctx context.Context, in CreatePageInput) (*Page, error) {
	body := map[string]any{
		"spaceId": in.SpaceID,
		"status":  "current",
		"title":   in.Title,
		"body":    map[string]any{"representation": "storage", "value": in.Body},
	}
	if in.ParentID != "" {
		body["parentId"] = in.ParentID
	}
	var resp pageJSON
	if _, err := c.doJSON(ctx, http.MethodPost, c.v2("/pages"), body, &resp); err != nil {
		return nil, err
	}
	return resp.toPage(), nil
}

// UpdatePage updates an existing page. Confluence requires an explicit, monotonically
// increasing version number, so the request is sent with version number in.Version+1.
func (c *Client) UpdatePage(ctx context.Context, in UpdatePageInput) (*Page, error) {
	body := map[string]any{
		"id":      in.ID,
		"status":  "current",
		"title":   in.Title,
		"body":    map[string]any{"representation": "storage", "value": in.Body},
		"version": map[string]any{"number": in.Version + 1},
	}
	var resp pageJSON
	if _, err := c.doJSON(ctx, http.MethodPut, c.v2("/pages/"+in.ID), body, &resp); err != nil {
		return nil, err
	}
	return resp.toPage(), nil
}
