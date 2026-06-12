package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/humblEgo/md2wiki/internal/confluence"
	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/title"
	"github.com/humblEgo/md2wiki/internal/tree"
	"github.com/humblEgo/md2wiki/internal/walker"
)

type fakeAPI struct {
	byTitle   map[string]*confluence.Page
	byID      map[string]*confluence.Page
	parent    map[string]string
	props     map[string]*confluence.Property
	atts      map[string][]string
	bodies    map[string]string
	nextID    int
	creates   int
	updates   int
	createErr error
}

func newFakeAPI() *fakeAPI {
	return &fakeAPI{
		byTitle: map[string]*confluence.Page{},
		byID:    map[string]*confluence.Page{},
		parent:  map[string]string{},
		props:   map[string]*confluence.Property{},
		atts:    map[string][]string{},
		bodies:  map[string]string{},
	}
}

func (f *fakeAPI) SpaceID(context.Context, string) (string, error) { return "SP1", nil }

func (f *fakeAPI) FindPage(_ context.Context, _, t string) (*confluence.Page, error) {
	return f.byTitle[t], nil
}

func (f *fakeAPI) CreatePage(_ context.Context, in confluence.CreatePageInput) (*confluence.Page, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.nextID++
	id := fmt.Sprintf("p%d", f.nextID)
	p := &confluence.Page{ID: id, Title: in.Title, Version: 1}
	f.byTitle[in.Title] = p
	f.byID[id] = p
	f.parent[id] = in.ParentID
	f.bodies[in.Title] = in.Body
	f.creates++
	return p, nil
}

func (f *fakeAPI) UpdatePage(_ context.Context, in confluence.UpdatePageInput) (*confluence.Page, error) {
	p := f.byID[in.ID]
	if p == nil {
		return nil, fmt.Errorf("fake: page %s not found", in.ID)
	}
	p.Version = in.Version + 1
	p.Title = in.Title
	f.bodies[p.Title] = in.Body
	f.updates++
	return p, nil
}

func (f *fakeAPI) UploadAttachment(_ context.Context, pageID, filename string, _ []byte) error {
	f.atts[pageID] = append(f.atts[pageID], filename)
	return nil
}

func (f *fakeAPI) GetContentProperty(_ context.Context, pageID, key string) (*confluence.Property, error) {
	return f.props[pageID+"|"+key], nil
}

func (f *fakeAPI) SetContentProperty(_ context.Context, pageID, key, value string, existing *confluence.Property) error {
	ver := 1
	if existing != nil {
		ver = existing.Version + 1
	}
	f.props[pageID+"|"+key] = &confluence.Property{ID: "prop-" + pageID, Key: key, Value: value, Version: ver}
	return nil
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

func buildAndResolve(t *testing.T, files map[string]string) (*tree.Tree, string) {
	t.Helper()
	root := writeTree(t, files)
	tr, err := walker.Walk(root, walker.ModeReadmeBody)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if err := title.Resolve(root, tr); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return tr, root
}

func TestRun_CreatesPages(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md": "# Home\n",
		"guide.md":  "# Guide\n",
	})
	api := newFakeAPI()
	res, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 2 || res.Updated != 0 {
		t.Errorf("result = %+v, want Created=2 Updated=0", res)
	}
	if api.byTitle["Home"] == nil || api.byTitle["Guide"] == nil {
		t.Errorf("expected pages Home and Guide, have %v", api.byTitle)
	}
}

func TestRun_ParentLinking(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md":     "# Home\n",
		"sub/README.md": "# Sub\n",
		"sub/page.md":   "# Page\n",
	})
	api := newFakeAPI()
	if _, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS"}); err != nil {
		t.Fatal(err)
	}
	homeID := api.byTitle["Home"].ID
	subID := api.byTitle["Sub"].ID
	pageID := api.byTitle["Page"].ID
	if api.parent[homeID] != "" {
		t.Errorf("Home parent = %q, want root (empty)", api.parent[homeID])
	}
	if api.parent[subID] != homeID {
		t.Errorf("Sub parent = %q, want %q", api.parent[subID], homeID)
	}
	if api.parent[pageID] != subID {
		t.Errorf("Page parent = %q, want %q", api.parent[pageID], subID)
	}
}

func TestRun_SetsContentHashProperty(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{"README.md": "# Home\n"})
	api := newFakeAPI()
	if _, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS"}); err != nil {
		t.Fatal(err)
	}
	homeID := api.byTitle["Home"].ID
	if api.props[homeID+"|md2wiki-content-hash"] == nil {
		t.Errorf("expected content-hash property on Home page")
	}
}

func TestRun_Idempotent(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md": "# Home\n",
		"g.md":      "# G\n",
	})
	api := newFakeAPI()
	cfg := Config{Root: root, SpaceKey: "DOCS"}
	if _, err := Run(context.Background(), api, tr, cfg); err != nil {
		t.Fatal(err)
	}
	res2, err := Run(context.Background(), api, tr, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Created != 0 || res2.Updated != 0 {
		t.Errorf("2nd run = %+v, want Created=0 Updated=0", res2)
	}
	if res2.Skipped != 2 {
		t.Errorf("2nd run Skipped = %d, want 2", res2.Skipped)
	}
}

type fakeMermaid struct{}

func (fakeMermaid) Render([]byte) ([]byte, error) { return []byte("PNGDATA"), nil }

func TestRun_UpdateOnChange(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md": "# Home\n",
		"g.md":      "# G\n\nold\n",
	})
	api := newFakeAPI()
	cfg := Config{Root: root, SpaceKey: "DOCS"}
	if _, err := Run(context.Background(), api, tr, cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "g.md"), []byte("# G\n\nNEW\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res2, err := Run(context.Background(), api, tr, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Updated != 1 || res2.Skipped != 1 {
		t.Errorf("2nd run = %+v, want Updated=1 Skipped=1", res2)
	}
}

func TestRun_UploadsAttachments(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md": "# Home\n",
		"d.md":      "# D\n\n```mermaid\ngraph TD;A-->B\n```\n",
	})
	api := newFakeAPI()
	cfg := Config{Root: root, SpaceKey: "DOCS", MermaidMode: convert.MermaidRender, MermaidRenderer: fakeMermaid{}}
	if _, err := Run(context.Background(), api, tr, cfg); err != nil {
		t.Fatal(err)
	}
	dID := api.byTitle["D"].ID
	if len(api.atts[dID]) != 1 {
		t.Errorf("attachments on D = %v, want 1", api.atts[dID])
	}
}

func TestRun_ConflictTitles(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"a/x.md": "# Same\n",
		"b/x.md": "# Same\n",
	})
	api := newFakeAPI()
	if _, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS"}); err != nil {
		t.Fatal(err)
	}
	if api.byTitle["Same (a/x.md)"] == nil || api.byTitle["Same (b/x.md)"] == nil {
		t.Errorf("expected path-suffixed conflict titles, have %v", keysOf(api.byTitle))
	}
}

func keysOf(m map[string]*confluence.Page) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestRun_FailFast(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{"README.md": "# Home\n"})
	api := newFakeAPI()
	api.createErr = fmt.Errorf("boom")
	_, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS"})
	if err == nil {
		t.Fatal("expected error to propagate (fail-fast)")
	}
}

func TestRun_RootParentID(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md": "# Home\n",
		"g.md":      "# G\n",
	})
	api := newFakeAPI()
	_, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS", RootParentID: "P0"})
	if err != nil {
		t.Fatal(err)
	}
	homeID := api.byTitle["Home"].ID
	if api.parent[homeID] != "P0" {
		t.Errorf("root parent = %q, want P0", api.parent[homeID])
	}
	gID := api.byTitle["G"].ID
	if api.parent[gID] != homeID {
		t.Errorf("child parent = %q, want root %q", api.parent[gID], homeID)
	}
}

func TestRun_BannerDefaultPrepended(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md": "# Home\n",
		"guide.md":  "# Guide\n",
	})
	api := newFakeAPI()
	if _, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS", Banner: true}); err != nil {
		t.Fatal(err)
	}
	for title, body := range api.bodies {
		if !strings.HasPrefix(body, convert.MirrorBanner()) {
			t.Errorf("page %q body does not start with banner:\n%s", title, body)
		}
	}
}

func TestRun_BannerOff(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md": "# Home\n",
	})
	api := newFakeAPI()
	if _, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS", Banner: false}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(api.bodies["Home"], `ac:name="info"`) {
		t.Errorf("banner should be absent when Banner=false:\n%s", api.bodies["Home"])
	}
}

// An empty folder page (a subfolder with no README) still gets the banner, and
// since it has no source body the resulting page body is the banner alone.
func TestRun_BannerOnEmptyFolderPage(t *testing.T) {
	tr, root := buildAndResolve(t, map[string]string{
		"README.md":   "# Home\n",
		"sub/page.md": "# Page\n",
	})
	api := newFakeAPI()
	if _, err := Run(context.Background(), api, tr, Config{Root: root, SpaceKey: "DOCS", Banner: true}); err != nil {
		t.Fatal(err)
	}
	body, ok := api.bodies["sub"]
	if !ok {
		t.Fatalf("expected an empty folder page titled %q; bodies=%v", "sub", api.bodies)
	}
	if body != convert.MirrorBanner() {
		t.Errorf("empty folder page body = %q, want banner-only", body)
	}
}
