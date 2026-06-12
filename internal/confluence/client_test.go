package confluence

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL, "me@x.com", "tok")
}

func assertBasicAuth(t *testing.T, r *http.Request, email, token string) {
	t.Helper()
	const p = "Basic "
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, p) {
		t.Fatalf("Authorization = %q, want Basic", h)
	}
	dec, err := base64.StdEncoding.DecodeString(h[len(p):])
	if err != nil {
		t.Fatalf("decode auth: %v", err)
	}
	if string(dec) != email+":"+token {
		t.Errorf("creds = %q, want %q", dec, email+":"+token)
	}
}

func TestSpaceID(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r, "me@x.com", "tok")
		if r.URL.Path != "/wiki/api/v2/spaces" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("keys"); got != "DOCS" {
			t.Errorf("keys = %q", got)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"42"}]}`))
	})
	id, err := c.SpaceID(context.Background(), "DOCS")
	if err != nil {
		t.Fatal(err)
	}
	if id != "42" {
		t.Errorf("id = %q, want 42", id)
	}
}

func TestSpaceID_NotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	if _, err := c.SpaceID(context.Background(), "NOPE"); err == nil {
		t.Fatal("expected error for missing space")
	}
}

func TestAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	_, err := c.SpaceID(context.Background(), "X")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.Status != 500 || !strings.Contains(apiErr.Body, "boom") {
		t.Errorf("apiErr = %+v", apiErr)
	}
}
