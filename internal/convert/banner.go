package convert

// MirrorBanner returns the Confluence storage-format XHTML snippet for the
// "this page is mirrored" info panel. The sync command prepends it to the top
// of a mirrored page's body so readers know the page is generated and must not
// be edited in Confluence directly.
func MirrorBanner() string {
	return `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>` +
		`This page is automatically mirrored from a Git repository by md2wiki. ` +
		`<strong>Do not edit it here</strong> — make changes in the source repo. ` +
		`Any edits will be overwritten on the next sync.` +
		`</p></ac:rich-text-body></ac:structured-macro>`
}
