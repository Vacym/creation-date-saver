package filter

import (
	"path/filepath"
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
