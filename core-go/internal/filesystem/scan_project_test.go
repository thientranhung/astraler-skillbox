package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// ValidateProjectPath
// ---------------------------------------------------------------------------

func TestValidateProjectPath_OK(t *testing.T) {
	dir := t.TempDir()
	if err := ValidateProjectPath(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProjectPath_NotAbsolute(t *testing.T) {
	err := ValidateProjectPath("relative/path")
	if err == nil {
		t.Fatal("expected error for relative path")
	}
	fe := assertFSError(t, err)
	if fe.Code != ErrNotAbsolute {
		t.Errorf("code: got %s want %s", fe.Code, ErrNotAbsolute)
	}
}

func TestValidateProjectPath_Missing(t *testing.T) {
	err := ValidateProjectPath("/tmp/skillbox-no-such-project-xyzabc")
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	fe := assertFSError(t, err)
	if fe.Code != ErrPathNotFound {
		t.Errorf("code: got %s want %s", fe.Code, ErrPathNotFound)
	}
}

func TestValidateProjectPath_NotDir(t *testing.T) {
	dir := t.TempDir()
	f, _ := os.CreateTemp(dir, "file")
	f.Close()
	err := ValidateProjectPath(f.Name())
	if err == nil {
		t.Fatal("expected error for regular file")
	}
	fe := assertFSError(t, err)
	if fe.Code != ErrNotADirectory {
		t.Errorf("code: got %s want %s", fe.Code, ErrNotADirectory)
	}
}

// ValidateProjectPath must NOT check writability — read-only project dirs are valid.
func TestValidateProjectPath_DoesNotRequireWritable(t *testing.T) {
	// Use a TempDir (always writable) as a proxy — key assertion is that the
	// function never calls checkWritable.  We verify by ensuring a normal
	// readable dir succeeds without error.
	dir := t.TempDir()
	if err := ValidateProjectPath(dir); err != nil {
		t.Fatalf("ValidateProjectPath should succeed for any readable dir: %v", err)
	}
}

// ---------------------------------------------------------------------------
// StatPathInfo
// ---------------------------------------------------------------------------

func TestStatPathInfo_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	pi, err := StatPathInfo(dir)
	if err != nil {
		t.Fatalf("StatPathInfo: %v", err)
	}
	if !pi.Exists {
		t.Error("Exists: want true")
	}
	if !pi.IsDir {
		t.Error("IsDir: want true")
	}
	if !pi.Readable {
		t.Error("Readable: want true")
	}
}

func TestStatPathInfo_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	f, _ := os.CreateTemp(dir, "f")
	f.Close()
	pi, err := StatPathInfo(f.Name())
	if err != nil {
		t.Fatalf("StatPathInfo: %v", err)
	}
	if !pi.Exists {
		t.Error("Exists: want true")
	}
	if pi.IsDir {
		t.Error("IsDir: want false for regular file")
	}
}

func TestStatPathInfo_Missing_NoError(t *testing.T) {
	pi, err := StatPathInfo("/tmp/skillbox-no-such-path-xyzabc9")
	if err != nil {
		t.Fatalf("ENOENT should not return error, got: %v", err)
	}
	if pi.Exists {
		t.Error("Exists: want false for missing path")
	}
}

func TestStatPathInfo_FollowsSymlinkToDir(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	pi, err := StatPathInfo(link)
	if err != nil {
		t.Fatalf("StatPathInfo: %v", err)
	}
	if !pi.Exists {
		t.Error("Exists: want true (symlink target exists)")
	}
	if !pi.IsDir {
		t.Error("IsDir: want true (symlink to dir, os.Stat follows)")
	}
}

func TestStatPathInfo_BrokenSymlink_NoError(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "broken")
	if err := os.Symlink("/tmp/skillbox-no-target-xyzabc9", link); err != nil {
		t.Fatal(err)
	}
	pi, err := StatPathInfo(link)
	if err != nil {
		t.Fatalf("broken symlink should not return error: %v", err)
	}
	if pi.Exists {
		t.Error("Exists: want false for broken symlink target")
	}
}

// ---------------------------------------------------------------------------
// ScanProjectSkills
// ---------------------------------------------------------------------------

func TestScanProjectSkills_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ScanProjectSkills(dir)
	if err != nil {
		t.Fatalf("ScanProjectSkills: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestScanProjectSkills_PlainDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "my-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := ScanProjectSkills(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Name != "my-skill" {
		t.Errorf("Name: %q", e.Name)
	}
	if !e.IsDir {
		t.Error("IsDir: want true")
	}
	if e.IsSymlink {
		t.Error("IsSymlink: want false")
	}
	if e.Broken {
		t.Error("Broken: want false")
	}
}

func TestScanProjectSkills_ValidSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real-skill")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "linked-skill")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanProjectSkills(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(t, entries, "linked-skill")
	if !e.IsSymlink {
		t.Error("IsSymlink: want true")
	}
	if e.Broken {
		t.Error("Broken: want false")
	}
	if e.ResolvedTarget == "" {
		t.Error("ResolvedTarget: want non-empty")
	}
	if e.SymlinkTargetRaw == "" {
		t.Error("SymlinkTargetRaw: want non-empty")
	}
	if e.ResolveError != nil {
		t.Errorf("ResolveError: want nil, got %v", e.ResolveError)
	}
}

func TestScanProjectSkills_BrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "broken-skill")
	if err := os.Symlink("/tmp/skillbox-no-such-target-xyzabc", link); err != nil {
		t.Fatal(err)
	}

	entries, err := ScanProjectSkills(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := findEntry(t, entries, "broken-skill")
	if !e.IsSymlink {
		t.Error("IsSymlink: want true")
	}
	if !e.Broken {
		t.Error("Broken: want true")
	}
	if e.ResolvedTarget != "" {
		t.Error("ResolvedTarget: want empty for broken symlink")
	}
	// SymlinkTargetRaw should be the raw target regardless of whether it exists.
	if e.SymlinkTargetRaw == "" {
		t.Error("SymlinkTargetRaw: want set even for broken symlinks")
	}
}

func TestScanProjectSkills_MissingDir_ReturnsError(t *testing.T) {
	_, err := ScanProjectSkills("/tmp/skillbox-no-such-dir-xyzabc")
	if err == nil {
		t.Fatal("expected error for missing skillsPath")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func assertFSError(t *testing.T, err error) *FilesystemError {
	t.Helper()
	fe, ok := err.(*FilesystemError)
	if !ok {
		t.Fatalf("expected *FilesystemError, got %T: %v", err, err)
	}
	return fe
}

func findEntry(t *testing.T, entries []ProjectEntry, name string) ProjectEntry {
	t.Helper()
	for _, e := range entries {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("entry %q not found in %v", name, entries)
	return ProjectEntry{}
}
