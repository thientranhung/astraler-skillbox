package filesystem

import (
	"testing"
)

func TestNormalizeAbs_Valid(t *testing.T) {
	got, err := NormalizeAbs("/tmp/../tmp/foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/foo" {
		t.Errorf("got %q want %q", got, "/tmp/foo")
	}
}

func TestNormalizeAbs_TrailingSlash(t *testing.T) {
	got, err := NormalizeAbs("/tmp/foo/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/foo" {
		t.Errorf("got %q want %q", got, "/tmp/foo")
	}
}

func TestNormalizeAbs_Relative(t *testing.T) {
	_, err := NormalizeAbs("relative/path")
	if err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestNormalizeAbs_Empty(t *testing.T) {
	_, err := NormalizeAbs("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}
