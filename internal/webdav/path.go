package webdav

import (
	"errors"
	"path"
	"strings"
)

// ErrPathTraversal indicates a path traversal attempt was detected.
var ErrPathTraversal = errors.New("invalid path: traversal detected")

// NormalizePath cleans paths for WebDAV requests and strictly prevents directory traversal attacks.
func NormalizePath(rawPath string) (string, error) {
	p := strings.TrimSpace(rawPath)

	// Split by '/' and check for '..' component to avoid traversal
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == ".." {
			return "", ErrPathTraversal
		}
	}

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	clean := path.Clean(p)
	return clean, nil
}
