package wizard

import (
	"net/url"
	"strings"
)

// parseDestination extracts a Confluence space key and optional parent page ID from a
// pasted URL such as https://team.atlassian.net/wiki/spaces/DOCS/pages/123456/Home
// (key=DOCS, parent=123456). A space-only URL yields an empty parent (space root). If s
// is not a URL with a /spaces/ segment, it is treated as a bare space key with no parent,
// so power users can still type the key directly.
func parseDestination(s string) (spaceKey, parentID string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if u, err := url.Parse(s); err == nil && u.Path != "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		var key, parent string
		for i, p := range parts {
			switch p {
			case "spaces":
				if i+1 < len(parts) {
					key = parts[i+1]
				}
			case "pages":
				if i+1 < len(parts) {
					parent = parts[i+1]
				}
			}
		}
		if key != "" {
			return key, parent
		}
	}
	return s, ""
}
