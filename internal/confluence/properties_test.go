package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestGetContentProperty_FoundAndMissing(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/wiki/api/v2/pages/9/properties" {
				t.Errorf("path = %q", r.URL.Path)
			}
			if r.URL.Query().Get("key") != "md2wiki-hash" {
				t.Errorf("key = %q", r.URL.Query().Get("key"))
			}
			_, _ = w.Write([]byte(`{"results":[{"id":"p1","key":"md2wiki-hash","value":"abc123","version":{"number":2}}]}`))
		})
		p, err := c.GetContentProperty(context.Background(), "9", "md2wiki-hash")
		if err != nil {
			t.Fatal(err)
		}
		if p == nil || p.ID != "p1" || p.Value != "abc123" || p.Version != 2 {
			t.Errorf("prop = %+v", p)
		}
	})
	t.Run("empty", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"results":[]}`))
		})
		p, err := c.GetContentProperty(context.Background(), "9", "k")
		if err != nil || p != nil {
			t.Errorf("p=%+v err=%v, want nil,nil", p, err)
		}
	})
	t.Run("404", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		p, err := c.GetContentProperty(context.Background(), "9", "k")
		if err != nil || p != nil {
			t.Errorf("p=%+v err=%v, want nil,nil", p, err)
		}
	})
}

func TestSetContentProperty_CreateThenUpdate(t *testing.T) {
	t.Run("create (existing nil -> POST)", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/wiki/api/v2/pages/9/properties" {
				t.Errorf("%s %s", r.Method, r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var got map[string]any
			_ = json.Unmarshal(body, &got)
			if got["key"] != "md2wiki-hash" || got["value"] != "h1" {
				t.Errorf("body = %v", got)
			}
			if _, ok := got["version"]; ok {
				t.Errorf("create must not send version")
			}
			_, _ = w.Write([]byte(`{"id":"p1"}`))
		})
		if err := c.SetContentProperty(context.Background(), "9", "md2wiki-hash", "h1", nil); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("update (existing -> PUT version+1)", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/wiki/api/v2/pages/9/properties/p1" {
				t.Errorf("%s %s", r.Method, r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var got map[string]any
			_ = json.Unmarshal(body, &got)
			ver := got["version"].(map[string]any)
			if ver["number"].(float64) != 3 {
				t.Errorf("version = %v, want 3", ver["number"])
			}
			_, _ = w.Write([]byte(`{"id":"p1"}`))
		})
		err := c.SetContentProperty(context.Background(), "9", "md2wiki-hash", "h2",
			&Property{ID: "p1", Key: "md2wiki-hash", Value: "h1", Version: 2})
		if err != nil {
			t.Fatal(err)
		}
	})
}
