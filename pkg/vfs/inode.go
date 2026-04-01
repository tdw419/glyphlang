package vfs

import (
	"fmt"
)

// FileType classifies what kind of file an inode represents.
type FileType uint8

const (
	FileRegular   FileType = iota // Regular file
	FileDirectory                 // Directory
	FileMount                     // Mount point
)

func (ft FileType) String() string {
	switch ft {
	case FileRegular:
		return "regular"
	case FileDirectory:
		return "directory"
	case FileMount:
		return "mount"
	default:
		return fmt.Sprintf("unknown(%d)", ft)
	}
}

// MaxInodes is the maximum number of inodes in the table.
const MaxInodes = 4096

// PermSubject identifies who is being checked for a permission.
type PermSubject uint8

const (
	User  PermSubject = iota // Owner
	Group                    // Group
	Other                    // Others
)

// Mode represents Unix-style permission bits (lower 9 bits).
type Mode uint16

func (m Mode) IsRead(s PermSubject) bool {
	return (m & Mode(modeBit(s, 2))) != 0
}

func (m Mode) IsWrite(s PermSubject) bool {
	return (m & Mode(modeBit(s, 1))) != 0
}

func (m Mode) IsExecute(s PermSubject) bool {
	return (m & Mode(modeBit(s, 0))) != 0
}

func modeBit(s PermSubject, bit int) uint16 {
	shift := uint8(0)
	switch s {
	case Other:
		shift = 0
	case Group:
		shift = 3
	case User:
		shift = 6
	}
	return 1 << (shift + uint8(bit))
}

// Inode represents a file metadata entry.
type Inode struct {
	Ino      uint32   // Inode number
	Type     FileType // File type
	Mode     Mode     // Permission bits
	Size     uint64   // File size in bytes
	Nlink    uint32   // Hard link count
	RefCount uint32   // Active open references
	data     []byte   // File content (regular files)

	// Directory entries: name -> child inode number
	// Only set when Type == FileDirectory
	DirEntries map[string]uint32
}

// Data returns a copy of the inode's data.
func (in *Inode) Data() []byte {
	if in.data == nil {
		return nil
	}
	cp := make([]byte, len(in.data))
	copy(cp, in.data)
	return cp
}

// SetData copies data into the inode and updates Size.
func (in *Inode) SetData(d []byte) {
	in.data = make([]byte, len(d))
	copy(in.data, d)
	in.Size = uint64(len(d))
}

// IncRef increments the reference count.
func (in *Inode) IncRef() {
	in.RefCount++
}

// DecRef decrements the reference count.
func (in *Inode) DecRef() {
	if in.RefCount > 0 {
		in.RefCount--
	}
}

// DirEntry is returned by ListEntries for directory listings.
type DirEntry struct {
	Name string
	Ino  uint32
}

// ListEntries returns the directory entries as a slice, including "." and "..".
func (in *Inode) ListEntries() []DirEntry {
	if in.DirEntries == nil {
		return nil
	}
	entries := make([]DirEntry, 0, len(in.DirEntries))
	for name, ino := range in.DirEntries {
		entries = append(entries, DirEntry{Name: name, Ino: ino})
	}
	return entries
}

// InodeTable manages a fixed-size table of inodes.
type InodeTable struct {
	inodes []*Inode
	free   []uint32 // free inode numbers for reuse
}

// NewInodeTable creates a new inode table with root directory pre-allocated.
func NewInodeTable() *InodeTable {
	t := &InodeTable{
		inodes: make([]*Inode, 0, MaxInodes),
		free:   make([]uint32, 0),
	}

	// Allocate root inode (ino 0)
	root := &Inode{
		Ino:        0,
		Type:       FileDirectory,
		Mode:       0755,
		Nlink:      2, // "." and ".." both point to root
		DirEntries: make(map[string]uint32),
	}
	root.DirEntries["."] = 0
	root.DirEntries[".."] = 0
	t.inodes = append(t.inodes, root)

	return t
}

// Get returns the inode by number, or nil if not allocated.
func (t *InodeTable) Get(ino uint32) *Inode {
	if ino >= uint32(len(t.inodes)) {
		return nil
	}
	return t.inodes[ino]
}

// Alloc creates a new inode and returns it. Returns nil if table is full.
func (t *InodeTable) Alloc(ft FileType, mode uint16) *Inode {
	var inoNum uint32

	if len(t.free) > 0 {
		// Reuse a freed slot
		inoNum = t.free[len(t.free)-1]
		t.free = t.free[:len(t.free)-1]
	} else {
		inoNum = uint32(len(t.inodes))
		if inoNum >= MaxInodes {
			return nil
		}
	}

	in := &Inode{
		Ino:   inoNum,
		Type:  ft,
		Mode:  Mode(mode),
		Nlink: 1,
	}

	if ft == FileDirectory {
		in.DirEntries = make(map[string]uint32)
	}

	if inoNum < uint32(len(t.inodes)) {
		t.inodes[inoNum] = in
	} else {
		t.inodes = append(t.inodes, in)
	}

	return in
}

// Free marks an inode as deallocated.
func (t *InodeTable) Free(ino uint32) {
	if ino == 0 {
		panic("cannot free root inode")
	}
	if ino >= uint32(len(t.inodes)) || t.inodes[ino] == nil {
		return
	}
	t.inodes[ino] = nil
	t.free = append(t.free, ino)
}
