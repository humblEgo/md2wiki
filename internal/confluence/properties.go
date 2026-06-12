package confluence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Property is a minimal representation of a page content property.
type Property struct {
	ID      string
	Key     string
	Value   string
	Version int
}

type propJSON struct {
	ID      string          `json:"id"`
	Key     string          `json:"key"`
	Value   json.RawMessage `json:"value"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
}

func (p propJSON) toProperty() *Property {
	val := string(p.Value)
	var s string
	if json.Unmarshal(p.Value, &s) == nil {
		val = s
	}
	return &Property{ID: p.ID, Key: p.Key, Value: val, Version: p.Version.Number}
}

// GetContentProperty looks up a content property by key. If no property with that key
// exists it returns (nil, nil); a 404 from the API is also treated as "not found" rather
// than an error, so callers can decide between creating and updating the property.
func (c *Client) GetContentProperty(ctx context.Context, pageID, key string) (*Property, error) {
	u := c.v2("/pages/"+pageID+"/properties") + "?" + url.Values{"key": {key}}.Encode()
	var resp struct {
		Results []propJSON `json:"results"`
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
	return resp.Results[0].toProperty(), nil
}

// SetContentProperty creates a content property when existing is nil, or updates it
// otherwise. As with page updates, an update must carry the next version number, so the
// request is sent with version number existing.Version+1.
func (c *Client) SetContentProperty(ctx context.Context, pageID, key, value string, existing *Property) error {
	if existing == nil {
		body := map[string]any{"key": key, "value": value}
		_, err := c.doJSON(ctx, http.MethodPost, c.v2("/pages/"+pageID+"/properties"), body, nil)
		return err
	}
	body := map[string]any{
		"key":     key,
		"value":   value,
		"version": map[string]any{"number": existing.Version + 1},
	}
	_, err := c.doJSON(ctx, http.MethodPut, c.v2("/pages/"+pageID+"/properties/"+existing.ID), body, nil)
	return err
}
