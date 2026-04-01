package vfs

import (
	"testing"
)

// SEC-1.2: Path resolution

func TestResolveAbsolutePath(t *testing.T) {
	fs := NewFileSystem()
	// Root always exists
	ino, err := fs.ResolvePath("/")
	if err != nil {
		t.Fatalf("ResolvePath(\"/\") error: %v", err)
	}
	if ino.Ino != 0 {
		t.Errorf("root ino = %d, want 0", ino.Ino)
	}
}

func TestResolveNonExistentPath(t *testing.T) {
	fs := NewFileSystem()
	_, err := fs.ResolvePath("/no/such/path")
	if err == nil {
		t.Error("expected ENOENT for non-existent path")
	}
	if err != ErrENOENT {
		t.Errorf("error = %v, want ErrENOENT", err)
	}
}

func TestResolveDotComponents(t *testing.T) {
	fs := NewFileSystem()
	// Create /foo directory
	fs.Mkdir("/foo", 0755)

	ino, err := fs.ResolvePath("/foo/./bar")
	_ = ino
	if err == nil {
		// /foo/bar doesn't exist yet, so this should be ENOENT...
		// actually, ./ is just "." which means "current", so /foo/. = /foo
		// /foo/./bar doesn't exist, correct
	}
	// But /foo/. should resolve to /foo
	ino, err = fs.ResolvePath("/foo/.")
	if err != nil {
		t.Fatalf("ResolvePath(\"/foo/.\") error: %v", err)
	}
	if ino.Type != FileDirectory {
		t.Errorf("/foo/. type = %v, want directory", ino.Type)
	}

	// And /foo/./ should also resolve to /foo
	ino, err = fs.ResolvePath("/foo/./")
	if err != nil {
		t.Fatalf("ResolvePath(\"/foo/./\") error: %v", err)
	}
	if ino.Type != FileDirectory {
		t.Errorf("/foo/./ type = %v, want directory", ino.Type)
	}
}

func TestResolveDotDot(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/foo", 0755)
	fs.Mkdir("/foo/bar", 0755)

	// /foo/bar/.. should resolve to /foo
	ino, err := fs.ResolvePath("/foo/bar/..")
	if err != nil {
		t.Fatalf("ResolvePath(\"/foo/bar/..\") error: %v", err)
	}
	// Check it's the /foo directory
	if ino.Type != FileDirectory {
		t.Errorf("/foo/bar/.. type = %v, want directory", ino.Type)
	}

	// /foo/../.. should resolve to / (root)
	ino, err = fs.ResolvePath("/foo/../..")
	if err != nil {
		t.Fatalf("ResolvePath(\"/foo/../..\") error: %v", err)
	}
	if ino.Ino != 0 {
		t.Errorf("/foo/../.. ino = %d, want 0 (root)", ino.Ino)
	}
}

func TestResolveDotDotAtRoot(t *testing.T) {
	fs := NewFileSystem()
	// "/.." should resolve to root (can't go above root)
	ino, err := fs.ResolvePath("/..")
	if err != nil {
		t.Fatalf("ResolvePath(\"/..\") error: %v", err)
	}
	if ino.Ino != 0 {
		t.Errorf("/.. ino = %d, want 0 (root)", ino.Ino)
	}
}

func TestResolveInvalidPaths(t *testing.T) {
	fs := NewFileSystem()

	tests := []struct {
		name string
		path string
		err  error
	}{
		{"empty string", "", ErrEINVAL},
		{"relative path", "foo/bar", ErrEINVAL},
		{"double slash", "//foo", ErrEINVAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := fs.ResolvePath(tt.path)
			if err != tt.err {
				t.Errorf("ResolvePath(%q) error = %v, want %v", tt.path, err, tt.err)
			}
		})
	}
}

func TestResolveDeepPath(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/a", 0755)
	fs.Mkdir("/a/b", 0755)
	fs.Mkdir("/a/b/c", 0755)

	ino, err := fs.ResolvePath("/a/b/c")
	if err != nil {
		t.Fatalf("ResolvePath(\"/a/b/c\") error: %v", err)
	}
	if ino.Type != FileDirectory {
		t.Errorf("/a/b/c type = %v, want directory", ino.Type)
	}
}

func TestResolveIntermediateNotDir(t *testing.T) {
	fs := NewFileSystem()
	// Create a regular file at /foo
	ino := fs.Table.Alloc(FileRegular, 0644)
	fs.Table.Get(0).DirEntries["foo"] = ino.Ino

	// Try to resolve /foo/bar — /foo is a file, not a directory
	_, err := fs.ResolvePath("/foo/bar")
	if err != ErrENOTDIR {
		t.Errorf("ResolvePath(\"/foo/bar\") error = %v, want ErrENOTDIR", err)
	}
}
