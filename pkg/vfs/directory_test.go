package vfs

import (
	"testing"
)

// SEC-1.3: Directory inode support

func TestMkdir(t *testing.T) {
	fs := NewFileSystem()
	ino, err := fs.Mkdir("/tmp", 0755)
	if err != nil {
		t.Fatalf("Mkdir(\"/tmp\") error: %v", err)
	}
	if ino.Type != FileDirectory {
		t.Errorf("type = %v, want directory", ino.Type)
	}
	if ino.Ino == 0 {
		t.Error("new directory should not be root inode")
	}

	// Verify it's in root's entries
	root := fs.Table.Get(0)
	if _, ok := root.DirEntries["tmp"]; !ok {
		t.Error("root DirEntries should contain 'tmp'")
	}
}

func TestMkdirAlreadyExists(t *testing.T) {
	fs := NewFileSystem()
	_, err := fs.Mkdir("/foo", 0755)
	if err != nil {
		t.Fatalf("first Mkdir error: %v", err)
	}
	_, err = fs.Mkdir("/foo", 0755)
	if err != ErrEEXIST {
		t.Errorf("duplicate Mkdir error = %v, want ErrEEXIST", err)
	}
}

func TestMkdirNested(t *testing.T) {
	fs := NewFileSystem()
	_, err := fs.Mkdir("/a", 0755)
	if err != nil {
		t.Fatalf("Mkdir(\"/a\") error: %v", err)
	}
	ino, err := fs.Mkdir("/a/b", 0755)
	if err != nil {
		t.Fatalf("Mkdir(\"/a/b\") error: %v", err)
	}
	if ino.Type != FileDirectory {
		t.Errorf("/a/b type = %v, want directory", ino.Type)
	}
}

func TestMkdirParentNotExist(t *testing.T) {
	fs := NewFileSystem()
	_, err := fs.Mkdir("/no/such/parent/dir", 0755)
	if err != ErrENOENT {
		t.Errorf("Mkdir with missing parent error = %v, want ErrENOENT", err)
	}
}

func TestDirEntries(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/foo", 0755)
	fs.Mkdir("/bar", 0755)

	root := fs.Table.Get(0)
	entries := root.ListEntries()

	// Should have at least: ".", "..", "foo", "bar"
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}

	if !names["."] {
		t.Error("root entries should contain '.'")
	}
	if !names[".."] {
		t.Error("root entries should contain '..'")
	}
	if !names["foo"] {
		t.Error("root entries should contain 'foo'")
	}
	if !names["bar"] {
		t.Error("root entries should contain 'bar'")
	}
}

func TestCreateFile(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/home", 0755)

	ino, err := fs.Create("/home/file.txt", 0644)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if ino.Type != FileRegular {
		t.Errorf("type = %v, want regular", ino.Type)
	}
	if ino.Size != 0 {
		t.Errorf("new file size = %d, want 0", ino.Size)
	}
}

func TestCreateFileAlreadyExists(t *testing.T) {
	fs := NewFileSystem()
	fs.Create("/file.txt", 0644)
	_, err := fs.Create("/file.txt", 0644)
	if err != ErrEEXIST {
		t.Errorf("duplicate Create error = %v, want ErrEEXIST", err)
	}
}

func TestUnlink(t *testing.T) {
	fs := NewFileSystem()
	ino, _ := fs.Create("/file.txt", 0644)

	err := fs.Unlink("/file.txt")
	if err != nil {
		t.Fatalf("Unlink error: %v", err)
	}

	// File should be gone from directory entries
	root := fs.Table.Get(0)
	if _, ok := root.DirEntries["file.txt"]; ok {
		t.Error("file.txt should be removed from root entries")
	}

	// Inode should be freed (ref count goes to 0)
	if ino.RefCount != 0 {
		t.Errorf("inode refcount = %d, want 0 after unlink", ino.RefCount)
	}
	if fs.Table.Get(ino.Ino) != nil {
		t.Error("inode should be freed after unlink when refcount hits 0")
	}
}

func TestUnlinkNonExistent(t *testing.T) {
	fs := NewFileSystem()
	err := fs.Unlink("/nope")
	if err != ErrENOENT {
		t.Errorf("Unlink non-existent error = %v, want ErrENOENT", err)
	}
}

func TestUnlinkDirectoryFails(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/dir", 0755)
	err := fs.Unlink("/dir")
	if err != ErrEISDIR {
		t.Errorf("Unlink directory error = %v, want ErrEISDIR", err)
	}
}

func TestRmdir(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/emptydir", 0755)

	err := fs.Rmdir("/emptydir")
	if err != nil {
		t.Fatalf("Rmdir error: %v", err)
	}

	root := fs.Table.Get(0)
	if _, ok := root.DirEntries["emptydir"]; ok {
		t.Error("emptydir should be removed from root entries")
	}
}

func TestRmdirNotEmptyFails(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/dir", 0755)
	fs.Create("/dir/file.txt", 0644)

	err := fs.Rmdir("/dir")
	if err != ErrENOTEMPTY {
		t.Errorf("Rmdir non-empty error = %v, want ErrENOTEMPTY", err)
	}
}

func TestRmdirNonExistent(t *testing.T) {
	fs := NewFileSystem()
	err := fs.Rmdir("/nope")
	if err != ErrENOENT {
		t.Errorf("Rmdir non-existent error = %v, want ErrENOENT", err)
	}
}

func TestParentLinks(t *testing.T) {
	fs := NewFileSystem()
	child, _ := fs.Mkdir("/child", 0755)

	// child's ".." should point to root
	parentIno := child.DirEntries[".."]
	if parentIno != 0 {
		t.Errorf("child's '..' = inode %d, want 0 (root)", parentIno)
	}

	// Root's ".." should point to itself
	root := fs.Table.Get(0)
	if root.DirEntries[".."] != 0 {
		t.Errorf("root's '..' = inode %d, want 0 (root itself)", root.DirEntries[".."])
	}
}

func TestNestedParentLinks(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/a", 0755)
	bIno, _ := fs.Mkdir("/a/b", 0755)

	// /a/b's ".." should point to /a
	aIno := fs.Table.Get(1) // first alloc after root
	bParent := bIno.DirEntries[".."]
	if bParent != aIno.Ino {
		t.Errorf("/a/b parent = %d, want %d (/a)", bParent, aIno.Ino)
	}
}

func TestListDir(t *testing.T) {
	fs := NewFileSystem()
	fs.Mkdir("/docs", 0755)
	fs.Create("/docs/a.txt", 0644)
	fs.Create("/docs/b.txt", 0644)

	docsIno, err := fs.ResolvePath("/docs")
	if err != nil {
		t.Fatalf("ResolvePath error: %v", err)
	}

	entries := docsIno.ListEntries()
	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}

	if !names["a.txt"] {
		t.Error("/docs should contain 'a.txt'")
	}
	if !names["b.txt"] {
		t.Error("/docs should contain 'b.txt'")
	}
	if !names["."] {
		t.Error("/docs should contain '.'")
	}
	if !names[".."] {
		t.Error("/docs should contain '..'")
	}
}
