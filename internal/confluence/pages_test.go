package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestFindPage_Found(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wiki/api/v2/pages" {
			t.Errorf("path = %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("space-id") != "42" || q.Get("title") != "Home" {
			t.Errorf("query = %v", q)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"7","title":"Home","version":{"number":3}}]}`))
	})
	p, err := c.FindPage(context.Background(), "42", "Home")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || p.ID != "7" || p.Title != "Home" || p.Version != 3 {
		t.Errorf("page = %+v", p)
	}
}

func TestFindPage_EmptyAndNotFound(t *testing.T) {
	t.Run("empty results", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"results":[]}`))
		})
		p, err := c.FindPage(context.Background(), "42", "X")
		if err != nil || p != nil {
			t.Errorf("p=%+v err=%v, want nil,nil", p, err)
		}
	})
	t.Run("404", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		p, err := c.FindPage(context.Background(), "42", "X")
		if err != nil || p != nil {
			t.Errorf("p=%+v err=%v, want nil,nil", p, err)
		}
	})
}

func TestCreatePage(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/wiki/api/v2/pages" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		if got["spaceId"] != "42" || got["title"] != "New" || got["status"] != "current" {
			t.Errorf("body = %v", got)
		}
		b := got["body"].(map[string]any)
		if b["representation"] != "storage" || b["value"] != "<p>hi</p>" {
			t.Errorf("body.body = %v", b)
		}
		_, _ = w.Write([]byte(`{"id":"9","title":"New","version":{"number":1}}`))
	})
	p, err := c.CreatePage(context.Background(), CreatePageInput{SpaceID: "42", Title: "New", Body: "<p>hi</p>"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "9" || p.Version != 1 {
		t.Errorf("page = %+v", p)
	}
}

func TestUpdatePage_VersionIncremented(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/wiki/api/v2/pages/9" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var got map[string]any
		_ = json.Unmarshal(body, &got)
		ver := got["version"].(map[string]any)
		if ver["number"].(float64) != 4 {
			t.Errorf("version.number = %v, want 4", ver["number"])
		}
		_, _ = w.Write([]byte(`{"id":"9","title":"Up","version":{"number":4}}`))
	})
	p, err := c.UpdatePage(context.Background(), UpdatePageInput{ID: "9", Title: "Up", Body: "<p>x</p>", Version: 3})
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != 4 {
		t.Errorf("version = %d, want 4", p.Version)
	}
}
