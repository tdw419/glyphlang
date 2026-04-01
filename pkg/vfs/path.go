package vfs

import (
	"errors"
	"strings"
)

// Sentinel errors matching POSIX conventions.
var (
	ErrENOENT   = errors.New("no such file or directory")
	ErrEINVAL   = errors.New("invalid argument")
	ErrEEXIST   = errors.New("file already exists")
	ErrENOTDIR  = errors.New("not a directory")
	ErrEISDIR   = errors.New("is a directory")
	ErrENOTEMPTY = errors.New("directory not empty")
)

// FileSystem holds the inode table and provides high-level VFS operations.
type FileSystem struct {
	Table *InodeTable
}

// NewFileSystem creates a new virtual filesystem with a root directory.
func NewFileSystem() *FileSystem {
	return &FileSystem{
		Table: NewInodeTable(),
	}
}

// resolveParent splits a path into its parent directory path and the final component.
// Returns (parent inode, basename, error).
func (fs *FileSystem) resolveParent(path string) (*Inode, string, error) {
	clean, err := cleanPath(path)
	if err != nil {
		return nil, "", err
	}

	// Special case: path is root
	if clean == "/" {
		return nil, "", ErrEINVAL
	}

	lastSlash := strings.LastIndex(clean, "/")
	parentPath := clean[:lastSlash]
	if parentPath == "" {
		parentPath = "/"
	}
	baseName := clean[lastSlash+1:]

	parent, err := fs.ResolvePath(parentPath)
	if err != nil {
		return nil, "", err
	}
	if parent.Type != FileDirectory {
		return nil, "", ErrENOTDIR
	}

	return parent, baseName, nil
}

// cleanPath validates and normalizes a path string.
func cleanPath(path string) (string, error) {
	if path == "" {
		return "", ErrEINVAL
	}
	if path[0] != '/' {
		return "", ErrEINVAL
	}
	// Reject double slashes
	if strings.Contains(path, "//") {
		return "", ErrEINVAL
	}
	// Remove trailing slash (unless it's root)
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path, nil
}

// ResolvePath resolves an absolute path to its inode, walking the directory tree.
// It handles "." and ".." components and returns ENOENT for missing entries.
func (fs *FileSystem) ResolvePath(path string) (*Inode, error) {
	clean, err := cleanPath(path)
	if err != nil {
		return nil, err
	}

	// Root
	if clean == "/" {
		return fs.Table.Get(0), nil
	}

	// Split into components
	parts := strings.Split(clean[1:], "/") // skip leading '/'

	current := fs.Table.Get(0) // start at root

	for _, part := range parts {
		if part == "" {
			continue
		}

		if current.Type != FileDirectory {
			return nil, ErrENOTDIR
		}

		switch part {
		case ".":
			// Stay in current directory
			continue
		case "..":
			// Go to parent
			parentIno, ok := current.DirEntries[".."]
			if !ok {
				// Root's ".." points to itself
				continue
			}
			current = fs.Table.Get(parentIno)
			if current == nil {
				return nil, ErrENOENT
			}
		default:
			childIno, ok := current.DirEntries[part]
			if !ok {
				return nil, ErrENOENT
			}
			child := fs.Table.Get(childIno)
			if child == nil {
				return nil, ErrENOENT
			}
			current = child
		}
	}

	return current, nil
}

// Mkdir creates a new directory at the given absolute path.
func (fs *FileSystem) Mkdir(path string, mode uint16) (*Inode, error) {
	parent, name, err := fs.resolveParent(path)
	if err != nil {
		return nil, err
	}

	// Check if already exists
	if _, ok := parent.DirEntries[name]; ok {
		return nil, ErrEEXIST
	}

	ino := fs.Table.Alloc(FileDirectory, mode)
	if ino == nil {
		return nil, errors.New("inode table full")
	}

	// Set up "." and ".."
	ino.DirEntries["."] = ino.Ino
	ino.DirEntries[".."] = parent.Ino
	ino.Nlink = 2 // "." and the entry in parent

	// Link into parent
	parent.DirEntries[name] = ino.Ino
	parent.Nlink++ // ".." from child

	return ino, nil
}

// Create creates a new regular file at the given absolute path.
func (fs *FileSystem) Create(path string, mode uint16) (*Inode, error) {
	parent, name, err := fs.resolveParent(path)
	if err != nil {
		return nil, err
	}

	if _, ok := parent.DirEntries[name]; ok {
		return nil, ErrEEXIST
	}

	ino := fs.Table.Alloc(FileRegular, mode)
	if ino == nil {
		return nil, errors.New("inode table full")
	}

	parent.DirEntries[name] = ino.Ino

	return ino, nil
}

// Unlink removes a regular file from the filesystem.
func (fs *FileSystem) Unlink(path string) error {
	parent, name, err := fs.resolveParent(path)
	if err != nil {
		return err
	}

	inoNum, ok := parent.DirEntries[name]
	if !ok {
		return ErrENOENT
	}

	ino := fs.Table.Get(inoNum)
	if ino == nil {
		return ErrENOENT
	}

	if ino.Type == FileDirectory {
		return ErrEISDIR
	}

	delete(parent.DirEntries, name)
	ino.Nlink--
	if ino.Nlink == 0 && ino.RefCount == 0 {
		fs.Table.Free(inoNum)
	}

	return nil
}

// Rmdir removes an empty directory.
func (fs *FileSystem) Rmdir(path string) error {
	parent, name, err := fs.resolveParent(path)
	if err != nil {
		return err
	}

	inoNum, ok := parent.DirEntries[name]
	if !ok {
		return ErrENOENT
	}

	ino := fs.Table.Get(inoNum)
	if ino == nil {
		return ErrENOENT
	}

	if ino.Type != FileDirectory {
		return ErrENOTDIR
	}

	// Check if directory is empty (only "." and ".." entries)
	for entryName := range ino.DirEntries {
		if entryName != "." && entryName != ".." {
			return ErrENOTEMPTY
		}
	}

	delete(parent.DirEntries, name)
	parent.Nlink-- // remove ".." reference

	// Clear the directory's own links
	ino.Nlink -= 2 // "." and ".."
	if ino.Nlink == 0 && ino.RefCount == 0 {
		fs.Table.Free(inoNum)
	}

	return nil
}
