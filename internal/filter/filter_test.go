package filter

import "testing"

func TestNewFilterAndShouldFilter(t *testing.T) {
	// Valid patterns
	f, err := NewFilter([]string{`.*\.tmp$`, `(?i)trash`, `^/home/.+?/Downloads/`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should match by extension
	if !f.ShouldFilter("/home/user/file.tmp") {
		t.Errorf("expected .tmp file to be filtered")
	}

	// Should match by case-insensitive 'trash'
	if !f.ShouldFilter("/home/user/.local/share/Trash/files/foo.txt") {
		t.Errorf("expected path containing 'Trash' to be filtered")
	}

	// Should match by prefix
	if !f.ShouldFilter("/home/user/Downloads/somefile.iso") {
		t.Errorf("expected Downloads path to be filtered")
	}

	// Should not match
	if f.ShouldFilter("/var/log/syslog") {
		t.Errorf("did not expect /var/log/syslog to be filtered")
	}
}

func TestNewFilterInvalidPattern(t *testing.T) {
	if _, err := NewFilter([]string{"[unclosed"}); err == nil {
		t.Fatalf("expected error for invalid pattern, got nil")
	}
}

func TestNilFilter(t *testing.T) {
	var f *Filter
	if f.ShouldFilter("/any/path") {
		t.Errorf("nil filter should not filter anything")
	}
}
