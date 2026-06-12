package confluence

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// assertMultipartFile asserts that the request carries a multipart "file" part with the
// expected filename and bytes, plus the X-Atlassian-Token: no-check header that Confluence
// requires to bypass its XSRF check on multipart uploads.
func assertMultipartFile(t *testing.T, r *http.Request, wantName, wantBody string) {
	t.Helper()
	if r.Header.Get("X-Atlassian-Token") != "no-check" {
		t.Errorf("missing X-Atlassian-Token: no-check")
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile: %v", err)
	}
	defer func() { _ = f.Close() }()
	if hdr.Filename != wantName {
		t.Errorf("filename = %q, want %q", hdr.Filename, wantName)
	}
	data, _ := io.ReadAll(f)
	if string(data) != wantBody {
		t.Errorf("data = %q, want %q", data, wantBody)
	}
}

// When no attachment exists yet: the lookup returns an empty result, so the upload must
// POST to the create endpoint (the page's child/attachment collection).
func TestUploadAttachment_Create(t *testing.T) {
	const base = "/wiki/rest/api/content/9/child/attachment"
	var sawCreate bool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			if got := r.URL.Query().Get("filename"); got != "diagram.png" {
				t.Errorf("filename query = %q, want diagram.png", got)
			}
			_, _ = w.Write([]byte(`{"results":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == base:
			sawCreate = true
			assertMultipartFile(t, r, "diagram.png", "PNGBYTES")
			_, _ = w.Write([]byte(`{"results":[{"id":"att1"}]}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	if err := c.UploadAttachment(context.Background(), "9", "diagram.png", []byte("PNGBYTES")); err != nil {
		t.Fatal(err)
	}
	if !sawCreate {
		t.Error("create POST not called")
	}
}

// When an attachment already exists (the core of this bug fix): the lookup returns an id,
// so the upload must POST to the per-attachment update-data endpoint instead of the create
// endpoint, avoiding the duplicate-filename conflict.
func TestUploadAttachment_UpdateExisting(t *testing.T) {
	const base = "/wiki/rest/api/content/9/child/attachment"
	var sawUpdate bool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == base:
			_, _ = w.Write([]byte(`{"results":[{"id":"att123"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == base+"/att123/data":
			sawUpdate = true
			assertMultipartFile(t, r, "mermaid-ef6a3b36d254.png", "PNGBYTES")
			_, _ = w.Write([]byte(`{"results":[{"id":"att123"}]}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	if err := c.UploadAttachment(context.Background(), "9", "mermaid-ef6a3b36d254.png", []byte("PNGBYTES")); err != nil {
		t.Fatal(err)
	}
	if !sawUpdate {
		t.Error("update-data POST not called")
	}
}

// When the upload response is non-2xx, the error is propagated as an *APIError.
func TestUploadAttachment_APIError(t *testing.T) {
	const base = "/wiki/rest/api/content/9/child/attachment"
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == base {
			_, _ = w.Write([]byte(`{"results":[]}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad"}`))
	})
	err := c.UploadAttachment(context.Background(), "9", "x.png", []byte("d"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", apiErr.Status)
	}
}

// When the lookup step fails, its error is propagated and the upload is never attempted.
func TestUploadAttachment_LookupError(t *testing.T) {
	const base = "/wiki/rest/api/content/9/child/attachment"
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == base {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
			return
		}
		t.Errorf("unexpected %s %s (upload should not be attempted)", r.Method, r.URL.Path)
	})
	err := c.UploadAttachment(context.Background(), "9", "x.png", []byte("d"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.Status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", apiErr.Status)
	}
}
