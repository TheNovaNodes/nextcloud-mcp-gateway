package webdav_test

import (
	"testing"

	"github.com/TheNovaNodes/nextcloud-mcp-gateway/internal/webdav"
)

func TestNormalizePath_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/", "/"},
		{"", "/"},
		{"docs", "/docs"},
		{"/docs/", "/docs"},
		{"docs/file.txt", "/docs/file.txt"},
		{"/a/b/c/", "/a/b/c"},
		{"  /spaced/path  ", "/spaced/path"},
	}

	for _, tt := range tests {
		got, err := webdav.NormalizePath(tt.input)
		if err != nil {
			t.Errorf("NormalizePath(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("NormalizePath(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizePath_Traversal(t *testing.T) {
	traversals := []string{
		"..",
		"../",
		"/../",
		"/docs/../../etc/passwd",
		"a/b/../c",
		"/foo/..",
	}

	for _, p := range traversals {
		_, err := webdav.NormalizePath(p)
		if err != webdav.ErrPathTraversal {
			t.Errorf("NormalizePath(%q) expected ErrPathTraversal, got %v", p, err)
		}
	}
}
