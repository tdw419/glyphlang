package vfs

import (
	"testing"
)

// SEC-1.1: Inode struct and inode table

func TestInodeTypeString(t *testing.T) {
	tests := []struct {
		ft  FileType
		str string
	}{
		{FileRegular, "regular"},
		{FileDirectory, "directory"},
		{FileMount, "mount"},
	}
	for _, tt := range tests {
		if got := tt.ft.String(); got != tt.str {
			t.Errorf("FileType(%d).String() = %q, want %q", tt.ft, got, tt.str)
		}
	}
}

func TestNewInodeTable(t *testing.T) {
	table := NewInodeTable()
	if table == nil {
		t.Fatal("NewInodeTable() returned nil")
	}
	// Root inode (ino 0) should be pre-allocated as a directory
	root := table.Get(0)
	if root == nil {
		t.Fatal("expected root inode at index 0")
	}
	if root.Type != FileDirectory {
		t.Errorf("root inode type = %v, want directory", root.Type)
	}
	if root.Nlink != 2 { // "." and ".." both point to root
		t.Errorf("root nlink = %d, want 2", root.Nlink)
	}
}

func TestInodeTableAlloc(t *testing.T) {
	table := NewInodeTable()

	ino := table.Alloc(FileRegular, 0644)
	if ino == nil {
		t.Fatal("Alloc() returned nil")
	}
	if ino.Ino != 1 { // 0 is root
		t.Errorf("first alloc ino = %d, want 1", ino.Ino)
	}
	if ino.Type != FileRegular {
		t.Errorf("type = %v, want regular", ino.Type)
	}
	if ino.Mode != 0644 {
		t.Errorf("mode = %o, want 0644", ino.Mode)
	}
	if ino.Nlink != 1 {
		t.Errorf("nlink = %d, want 1", ino.Nlink)
	}
	if ino.Size != 0 {
		t.Errorf("size = %d, want 0", ino.Size)
	}
}

func TestInodeTableGet(t *testing.T) {
	table := NewInodeTable()

	// Existing inode
	ino := table.Get(0)
	if ino == nil {
		t.Fatal("Get(0) returned nil for root inode")
	}

	// Non-existent inode
	if table.Get(99) != nil {
		t.Error("Get(99) should return nil for non-existent inode")
	}
}

func TestInodeTableFree(t *testing.T) {
	table := NewInodeTable()

	ino := table.Alloc(FileRegular, 0644)
	inoNum := ino.Ino

	table.Free(inoNum)

	if table.Get(inoNum) != nil {
		t.Error("Get() should return nil after Free()")
	}
}

func TestInodeTableFreeRootPanics(t *testing.T) {
	table := NewInodeTable()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Free(0) on root should panic")
		}
	}()
	table.Free(0)
}

func TestInodeRefCount(t *testing.T) {
	table := NewInodeTable()

	ino := table.Alloc(FileRegular, 0644)
	if ino.RefCount != 0 {
		t.Errorf("initial refcount = %d, want 0", ino.RefCount)
	}

	ino.IncRef()
	if ino.RefCount != 1 {
		t.Errorf("after IncRef = %d, want 1", ino.RefCount)
	}

	ino.DecRef()
	if ino.RefCount != 0 {
		t.Errorf("after DecRef = %d, want 0", ino.RefCount)
	}
}

func TestInodeSetData(t *testing.T) {
	table := NewInodeTable()
	ino := table.Alloc(FileRegular, 0644)

	data := []byte("hello world")
	ino.SetData(data)

	if ino.Size != uint64(len(data)) {
		t.Errorf("size = %d, want %d", ino.Size, len(data))
	}
	// Verify data is copied, not shared
	modified := append(data, '!')
	if string(ino.Data()) == string(modified) {
		t.Error("inode data should be a copy, not shared with caller")
	}
	if string(ino.Data()) != "hello world" {
		t.Errorf("data = %q, want %q", string(ino.Data()), "hello world")
	}
}

func TestInodeTableCapacity(t *testing.T) {
	table := NewInodeTable()

	// Root takes slot 0. Alloc MaxInodes-1 more should work.
	for i := 1; i < int(MaxInodes); i++ {
		ino := table.Alloc(FileRegular, 0644)
		if ino == nil {
			t.Fatalf("Alloc() %d returned nil", i)
		}
	}

	// Next alloc should fail (table full)
	ino := table.Alloc(FileRegular, 0644)
	if ino != nil {
		t.Error("Alloc() beyond MaxInodes should return nil")
	}
}

func TestInodePermissions(t *testing.T) {
	table := NewInodeTable()

	ino := table.Alloc(FileRegular, 0755)
	if !ino.Mode.IsRead(User) {
		t.Error("owner should have read permission for 0755")
	}
	if !ino.Mode.IsWrite(User) {
		t.Error("owner should have write permission for 0755")
	}
	if !ino.Mode.IsExecute(User) {
		t.Error("owner should have execute permission for 0755")
	}
	if !ino.Mode.IsRead(Group) {
		t.Error("group should have read permission for 0755")
	}
	if ino.Mode.IsWrite(Group) {
		t.Error("group should NOT have write permission for 0755")
	}
	if !ino.Mode.IsExecute(Group) {
		t.Error("group should have execute permission for 0755")
	}
	if !ino.Mode.IsRead(Other) {
		t.Error("other should have read permission for 0755")
	}
	if ino.Mode.IsWrite(Other) {
		t.Error("other should NOT have write permission for 0755")
	}
}
