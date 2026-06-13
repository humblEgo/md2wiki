# md2wiki

[![CI](https://github.com/humblEgo/md2wiki/actions/workflows/ci.yml/badge.svg)](https://github.com/humblEgo/md2wiki/actions/workflows/ci.yml)

> A CLI that treats your Git repo's Markdown as the single source of truth (SSOT) and mirrors it one-way into a Confluence wiki, cleanly.

Maintaining docs in two places means they drift apart fast. md2wiki keeps **the Markdown in your repo as the truth** and automatically reconciles Confluence to match it. So—

- **Developers** work directly with the `.md` files in the repo, and
- **PMs and non-developers** read the same content in Confluence and use it as the source for product specs.

When the repo changes, a single `md2wiki sync` brings the wiki up to date. It only updates pages that actually changed, so it's safe to run repeatedly.

## At a glance

Given a directory like this (the first `#` heading in each file becomes the page title):

```
docs/
├── README.md        # "Project Overview"
├── setup.md         # "Setup Guide"
└── api/
    ├── README.md    # "API Reference"
    └── auth.md      # "Authentication"
```

With the default `readme-body` layout it mirrors like this — each folder's `README.md` becomes the **body** of that folder's page, and the other `.md` files become **child pages**:

```
Project Overview          ← docs/README.md
├── Setup Guide           ← docs/setup.md
└── API Reference         ← docs/api/README.md
    └── Authentication    ← docs/api/auth.md
```

## ✨ Features

- **Interactive setup** — `md2wiki init` generates `md2wiki.yaml` through a guided wizard (arrow-key selects), verifies your Confluence connection, and opens the API-token page for you. The token is never written to the file.
- **Directory tree mirroring** — maps your folder structure onto a Confluence page hierarchy. Two layouts are supported:
  - `readme-body` (default): a folder's `README.md` becomes the folder page's body, and the remaining `.md` files become child pages.
  - `mirror`: a 1:1 reflection of the filesystem. `README.md` becomes an ordinary page too.
- **Cross-document link conversion** — rewrites relative `.md` links into the corresponding Confluence page links. External links and links with no matching target are left untouched.
- **Automatic title-conflict handling** — Confluence requires titles to be unique within a space. When several documents share a title, md2wiki disambiguates deterministically by appending the path (`Title (path/to/file.md)`), so syncs don't break.
- **Mermaid diagrams, three ways** — choose with `--mermaid-mode`:
  - `details` (default): the rendered image plus the original source in a collapsible (expand) region (when you want both)
  - `render`: the rendered image (PNG) only
  - `raw`: the original mermaid as a code block only (works without any external tooling)
- **Table of contents** — put `<!-- toc -->` anywhere in a Markdown file and md2wiki replaces it in place with Confluence's native table-of-contents macro, which auto-builds from the page headings. Opt-in per file; pages without the marker are unaffected.
- **Idempotent sync** — stores a content hash on each page and updates **only the pages that changed**. Running it again with the same input changes nothing.
- **Mirror notice banner** — every mirrored page gets an info panel at the top telling readers the page is generated from a Git repo and not to edit it in Confluence. On by default; turn it off with `--banner=false` (sync) or `banner: false` in `md2wiki.yaml` (apply, global or per-mapping).

## Requirements

- **Go 1.26+** (when building from source)
- **`mmdc`** ([mermaid-cli](https://github.com/mermaid-js/mermaid-cli)) — only needed for the mermaid `render`/`details` modes. Not required for `raw` mode.
- A **Confluence Cloud account** and an [API token](https://id.atlassian.com/manage-profile/security/api-tokens)

## Installation

```bash
go install github.com/humblEgo/md2wiki/cmd/md2wiki@latest
```

Or from source:

```bash
git clone https://github.com/humblEgo/md2wiki
cd md2wiki
make build   # produces bin/md2wiki
```

## Quick start with `md2wiki init`

If you'd rather not hand-write `md2wiki.yaml`, run the interactive wizard:

```bash
md2wiki init
```

It first asks for the easy stuff — layout/mermaid defaults and one or more
directory→destination mappings — then your Confluence connection last, so you can walk
through it even without credentials handy. For each mapping you just **paste the
Confluence URL** of the page you want to mirror under (e.g.
`https://your-team.atlassian.net/wiki/spaces/DOCS/pages/123456/Home`); the wizard pulls
the space key and parent page out of it (a bare space key works too).

It can open the [API token page](https://id.atlassian.com/manage-profile/security/api-tokens)
in your browser, and — if you paste a token — verifies the connection to each space before
finishing. The token is **never written to the file**; the wizard prints the
`export MD2WIKI_API_TOKEN=...` line for you to set in your shell.

If `md2wiki.yaml` already exists, the wizard asks before overwriting and lets you enter a
different filename (use it later with `md2wiki apply --config <path>`).

## Quick start

The API token is read **from an environment variable only**, so it can't be exposed by accident.

```bash
export MD2WIKI_API_TOKEN='<your-confluence-api-token>'

md2wiki sync ./docs \
  --base-url https://your-team.atlassian.net \
  --email you@your-team.com \
  --space DOCS
```

When it finishes, it reports what changed:

```
created=3 updated=1 skipped=12
```

## Use case: mirroring a specific directory under a specific page

Combine three things to decide "what / which space / under which page" to publish:

- **What** — pick the repo directory to mirror with the `<dir>` argument (e.g. `./docs/api`).
- **Which space** — set the target space key with `--space`.
- **Under which page** — set the parent page with `--parent-id` (optional). If omitted, content goes at the top level of the space.

For example, to mirror the `docs/api` directory under a page (ID `123456`) in space `DOCS`:

```bash
md2wiki sync ./docs/api \
  --space DOCS \
  --parent-id 123456 \
  --base-url https://your-team.atlassian.net \
  --email you@your-team.com
```

You can find the parent page ID in the page's URL — it's the `123456` part of
`https://your-team.atlassian.net/wiki/spaces/DOCS/pages/123456/Title`.

If you omit `--parent-id`, the root of the mirror tree is created at the top level of the space.

## Many mappings in one file: `md2wiki apply`

Instead of typing long flags every time, you can place a `md2wiki.yaml` at your repo root and declare several directory→space/page mappings to sync them all at once.

```yaml
# md2wiki.yaml
baseUrl: https://your-team.atlassian.net   # optional (overridable via --base-url / env)
email: you@your-team.com                    # optional (overridable via --email / env)
layoutMode: readme-body                     # optional, global default (defaults to readme-body)
mermaidMode: details                        # optional, global default (defaults to details)
banner: true                                # optional, global default (defaults to true); per-mapping override allowed
mappings:
  - source: docs/product_specs              # required, relative to this file
    space: PROD                             # required
    rootPage: "1897267223"                  # optional, mirror under this page
  - source: docs/runbooks
    space: OPS
    rootPage: "555111"
    layoutMode: mirror                      # optional, overrides for this mapping only
```

```bash
export MD2WIKI_API_TOKEN='<your-confluence-api-token>'
md2wiki apply                               # uses md2wiki.yaml in the current directory
md2wiki apply --config infra/md2wiki.yaml   # point at a different path
```

It syncs each mapping **in order** and reports the result of each on one line:

```
[PROD] docs/product_specs: created=3 updated=1 skipped=12
[OPS] docs/runbooks: created=0 updated=2 skipped=8
```

- If a mapping fails, it **stops immediately** (mappings that already succeeded are already applied). Fix the issue and run `apply` again — it's idempotent, so it picks up where it left off.
- `source` is relative to the location of `md2wiki.yaml`.
- The API token is not kept in the file here either; it's read only from the `MD2WIKI_API_TOKEN` environment variable.

## Configuration

Set via flags or environment variables (flags take precedence). Environment variables use the `MD2WIKI_` prefix.

| Setting | Flag | Env var | Default | Required |
|---------|------|---------|---------|----------|
| Confluence base URL | `--base-url` | `MD2WIKI_BASE_URL` | — | ✓ |
| Account email | `--email` | `MD2WIKI_EMAIL` | — | ✓ |
| Space key | `--space` | `MD2WIKI_SPACE` | — | ✓ |
| Parent page ID | `--parent-id` | `MD2WIKI_PARENT_ID` | (top level of space) | |
| API token | (no flag) | `MD2WIKI_API_TOKEN` | — | ✓ |
| Layout mode | `--layout-mode` | `MD2WIKI_LAYOUT_MODE` | `readme-body` | |
| Mermaid mode | `--mermaid-mode` | `MD2WIKI_MERMAID_MODE` | `details` | |
| Mirror banner | `--banner` | `MD2WIKI_BANNER` | `true` | |

> The API token deliberately has no flag, so it won't end up in the process list or your shell history.

## How it works

It runs in two passes:

1. **Indexing** — walks the directory to build a document tree (applying the layout mode), derives each page title from the first `#` heading (falling back to the file/folder name), and resolves conflicts globally. It also builds a path→page index used for link conversion.
2. **Conversion & sync** — converts each document to the Confluence storage format (including link rewriting and mermaid handling), then compares content hashes to decide create / update / skip. Rendered mermaid images are uploaded as attachments.

During the walk, dotfiles/dotfolders, non-`.md` files, and symbolic links are skipped.

## Known limitations

It's still an MVP, so there are a few boundaries:

- **Renaming leaves the old page behind.** md2wiki treats the repo as **read-only** (it never touches source files) and finds pages by title. So when a document's title changes (file move, `#` heading change, conflict resolution, etc.), a new page is created and the old one remains in Confluence. A reconcile step to clean up orphaned pages is the next task.
- **Local image attachments** (as opposed to external URLs), multiple spaces, and page-ID write-back (rename safety) are not yet in scope.

## Development

```bash
make build   # build the binary
make test    # run all tests
make lint    # golangci-lint
make vet     # go vet
```

Design and implementation notes are kept per milestone:

- `docs/superpowers/specs/` — design decisions for each stage
- `docs/superpowers/plans/` — implementation plan for each stage

## Contributing

Issues and PRs are welcome. For larger changes, please open an issue first so we can align on direction. When sending code, please make sure `make test` and `make lint` pass.

## License

Released under the [MIT License](LICENSE).

## Status

The MVP core is complete — directory mirroring, title-conflict resolution, link conversion, the three mermaid modes, and idempotent sync all work. CI runs on every push to `main` and on pull requests (build, vet, test, lint). Release binaries via goreleaser are wired and ready; the tag trigger is enabled when cutting the first release.
