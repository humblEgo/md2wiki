// Package sync idempotently mirrors a document tree into Confluence.
package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/humblEgo/md2wiki/internal/confluence"
	"github.com/humblEgo/md2wiki/internal/convert"
	"github.com/humblEgo/md2wiki/internal/link"
	"github.com/humblEgo/md2wiki/internal/tree"
)

const hashKey = "md2wiki-content-hash"

// API is the set of Confluence operations that sync depends on. The concrete
// *confluence.Client satisfies it; tests can substitute a fake implementation.
type API interface {
	SpaceID(ctx context.Context, spaceKey string) (string, error)
	FindPage(ctx context.Context, spaceID, title string) (*confluence.Page, error)
	CreatePage(ctx context.Context, in confluence.CreatePageInput) (*confluence.Page, error)
	UpdatePage(ctx context.Context, in confluence.UpdatePageInput) (*confluence.Page, error)
	UploadAttachment(ctx context.Context, pageID, filename string, data []byte) error
	GetContentProperty(ctx context.Context, pageID, key string) (*confluence.Property, error)
	SetContentProperty(ctx context.Context, pageID, key, value string, existing *confluence.Property) error
}

// Compile-time assertion that *confluence.Client satisfies API. This guards
// against signature drift: if the client's method set ever diverges from the
// interface, the build fails here instead of at some distant call site.
var _ API = (*confluence.Client)(nil)

// Config holds the settings for a sync run.
type Config struct {
	Root            string
	SpaceKey        string
	RootParentID    string // If empty, the mirror tree's root is created at the top of the space. If set, the root is created as a child of this page.
	MermaidMode     convert.MermaidMode
	MermaidRenderer convert.MermaidRenderer
	Banner          bool // If true, prepend a "this page is mirrored" notice banner to the top of every page body.
}

// Result summarizes the outcome of a sync run.
type Result struct {
	Created int
	Updated int
	Skipped int
}

// Run idempotently mirrors the document tree into Confluence. It resolves the
// space, then walks the tree from the root, creating, updating, or skipping a
// page for each node depending on whether its content has changed.
func Run(ctx context.Context, api API, t *tree.Tree, cfg Config) (Result, error) {
	var res Result
	if t == nil || t.Root == nil {
		return res, nil
	}
	spaceID, err := api.SpaceID(ctx, cfg.SpaceKey)
	if err != nil {
		return res, err
	}
	if err := syncNode(ctx, api, t, cfg, t.Root, cfg.RootParentID, spaceID, &res); err != nil {
		return res, err
	}
	return res, nil
}

func syncNode(ctx context.Context, api API, t *tree.Tree, cfg Config, n *tree.Node, parentID, spaceID string, res *Result) error {
	body, atts, err := render(t, cfg, n)
	if err != nil {
		return err
	}
	hash := sha256hex(body)

	existing, err := api.FindPage(ctx, spaceID, n.Title)
	if err != nil {
		return err
	}

	var pageID string
	if existing == nil {
		page, err := api.CreatePage(ctx, confluence.CreatePageInput{
			SpaceID: spaceID, Title: n.Title, ParentID: parentID, Body: body,
		})
		if err != nil {
			return err
		}
		if err := uploadAll(ctx, api, page.ID, atts); err != nil {
			return err
		}
		if err := api.SetContentProperty(ctx, page.ID, hashKey, hash, nil); err != nil {
			return err
		}
		res.Created++
		pageID = page.ID
	} else {
		prop, err := api.GetContentProperty(ctx, existing.ID, hashKey)
		if err != nil {
			return err
		}
		if prop != nil && prop.Value == hash {
			res.Skipped++
			pageID = existing.ID
		} else {
			page, err := api.UpdatePage(ctx, confluence.UpdatePageInput{
				ID: existing.ID, Title: n.Title, Body: body, Version: existing.Version,
			})
			if err != nil {
				return err
			}
			if err := uploadAll(ctx, api, existing.ID, atts); err != nil {
				return err
			}
			if err := api.SetContentProperty(ctx, existing.ID, hashKey, hash, prop); err != nil {
				return err
			}
			res.Updated++
			pageID = page.ID
		}
	}

	for _, c := range n.Children {
		if err := syncNode(ctx, api, t, cfg, c, pageID, spaceID, res); err != nil {
			return err
		}
	}
	return nil
}

func uploadAll(ctx context.Context, api API, pageID string, atts []convert.Attachment) error {
	for _, a := range atts {
		if err := api.UploadAttachment(ctx, pageID, a.Filename, a.Data); err != nil {
			return err
		}
	}
	return nil
}

// render converts a node into a Confluence storage-format body and its
// attachments. When the node has no SourcePath (for example, a folder page that
// has no README), the body starts out empty. When cfg.Banner is set, the mirror
// notice banner is prepended to the top of the body regardless of whether a
// source document exists, so even empty folder pages carry the notice.
func render(t *tree.Tree, cfg Config, n *tree.Node) (string, []convert.Attachment, error) {
	var body string
	var atts []convert.Attachment
	if n.SourcePath != "" {
		src, err := os.ReadFile(filepath.Join(cfg.Root, filepath.FromSlash(n.SourcePath)))
		if err != nil {
			return "", nil, fmt.Errorf("sync: read %q: %w", n.SourcePath, err)
		}
		body, err = convert.Document(src,
			convert.WithLinkResolver(link.NewResolver(t, n.SourcePath)),
			convert.WithMermaidMode(cfg.MermaidMode),
			convert.WithMermaidRenderer(cfg.MermaidRenderer),
			convert.WithAttachmentSink(func(a convert.Attachment) { atts = append(atts, a) }),
		)
		if err != nil {
			return "", nil, fmt.Errorf("sync: convert %q: %w", n.SourcePath, err)
		}
	}
	if cfg.Banner {
		body = convert.MirrorBanner() + body
	}
	return body, atts, nil
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
