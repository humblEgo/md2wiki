package wizard

import "testing"

func TestBrowserCommand(t *testing.T) {
	tests := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"darwin", "open", []string{"https://x"}},
		{"windows", "cmd", []string{"/c", "start", "", "https://x"}},
		{"linux", "xdg-open", []string{"https://x"}},
	}
	for _, tt := range tests {
		name, args := browserCommand(tt.goos, "https://x")
		if name != tt.wantName {
			t.Errorf("%s: name = %q, want %q", tt.goos, name, tt.wantName)
		}
		if len(args) != len(tt.wantArgs) {
			t.Fatalf("%s: args = %v, want %v", tt.goos, args, tt.wantArgs)
		}
		for i := range args {
			if args[i] != tt.wantArgs[i] {
				t.Errorf("%s: args[%d] = %q, want %q", tt.goos, i, args[i], tt.wantArgs[i])
			}
		}
	}
}
