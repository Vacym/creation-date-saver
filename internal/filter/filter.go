package filter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// IsTemporaryFile checks if the file is a temporary or unwanted file.
func IsTemporaryFile(filename string) bool {
	// Check for common temporary file patterns.
	return strings.HasPrefix(filepath.Base(filename), ".") || strings.Contains(filename, "goutputstream")
}

// IsRenameToTrash determines if the rename event is a move to trash.
func IsRenameToTrash(oldPath, newPath string) bool {
	// Add logic to identify if the rename is related to trash (platform-specific).
	// Example: if moved to a "Trash" folder or specific naming conventions.
	return strings.Contains(strings.ToLower(newPath), "trash") || strings.Contains(strings.ToLower(oldPath), "recycle")
}

// Filter holds a set of compiled regular expressions and can decide
// whether a given path should be filtered (i.e., matched by any pattern).
type Filter struct {
	exprs []*regexp.Regexp
}

// NewFilter creates a new Filter by compiling the provided regex patterns.
// Returns an error if any pattern fails to compile.
func NewFilter(patterns []string) (*Filter, error) {
	exprs := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if strings.TrimSpace(p) == "" {
			// Skip empty patterns silently.
			continue
		}
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("compile pattern %q: %w", p, err)
		}
		exprs = append(exprs, re)
	}
	return &Filter{exprs: exprs}, nil
}

// ShouldFilter returns true if the provided path matches at least one
// of the filter's regex patterns.
func (f *Filter) ShouldFilter(path string) bool {
	if f == nil {
		return false
	}
	for _, re := range f.exprs {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}
