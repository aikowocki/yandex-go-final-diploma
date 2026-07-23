package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldLaunchTUI(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args launches TUI", args: nil, want: true},
		{name: "explicit tui arg launches TUI", args: []string{"tui"}, want: true},
		{name: "cli command does not launch TUI", args: []string{"login"}, want: false},
		{name: "tui plus extra args does not match", args: []string{"tui", "extra"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldLaunchTUI(tt.args); got != tt.want {
				t.Errorf("shouldLaunchTUI(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestFormatBuildDate(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "valid RFC3339", raw: "2026-07-18T10:00:00Z", want: "2026-07-18"},
		{name: "unknown passthrough", raw: "unknown", want: "unknown"},
		{name: "garbage passthrough", raw: "not-a-date", want: "not-a-date"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatBuildDate(tt.raw); got != tt.want {
				t.Errorf("formatBuildDate(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "present.txt")
	if err := os.WriteFile(existing, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if !fileExists(existing) {
		t.Errorf("fileExists(%q) = false, want true", existing)
	}
	if fileExists(filepath.Join(dir, "missing.txt")) {
		t.Errorf("fileExists(missing) = true, want false")
	}
}
