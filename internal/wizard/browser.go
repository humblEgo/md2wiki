package wizard

import (
	"os/exec"
	"runtime"
)

// browserCommand returns the OS-appropriate command and args to open url in the
// default browser. Split out as a pure function so it can be unit-tested without
// actually launching anything.
func browserCommand(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		// The empty "" is start's window-title argument; without it a quoted URL
		// would be mistaken for the title.
		return "cmd", []string{"/c", "start", "", url}
	default:
		return "xdg-open", []string{url}
	}
}

// OpenBrowser opens url in the current OS's default browser. It returns once the
// launcher process has started; it does not wait for the browser to appear.
func OpenBrowser(url string) error {
	name, args := browserCommand(runtime.GOOS, url)
	return exec.Command(name, args...).Start()
}
