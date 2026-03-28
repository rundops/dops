package adapters

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "bare tilde",
			path: "~",
			want: home,
		},
		{
			name: "tilde with forward slash",
			path: "~/documents/file.txt",
			want: filepath.Join(home, "documents/file.txt"),
		},
		{
			name: "no tilde",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "tilde in middle is not expanded",
			path: "/foo/~/bar",
			want: "/foo/~/bar",
		},
		{
			name: "empty string",
			path: "",
			want: "",
		},
		{
			name: "tilde only prefix no sep",
			path: "~username",
			want: "~username",
		},
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name string
			path string
			want string
		}{
			name: "tilde with backslash",
			path: `~\documents\file.txt`,
			want: filepath.Join(home, `documents\file.txt`),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandHome(tt.path)
			if got != tt.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
