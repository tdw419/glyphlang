# Learnings: SEC-1 VFS Inode Table and Path Resolution

## What worked
- Creating the VFS as a standalone package (`pkg/vfs/`) separate from the VM keeps concerns clean. The inode table and path resolution have zero dependency on the VM package.
- Modeling the inode table as a Go slice with free-list reuse (instead of a map) gives O(1) allocation and lookup.
- Pre-allocating root inode (ino 0) with "." and ".." pointing to itself simplifies all path resolution — no special cases for root traversal.
- Sentinel errors (ErrENOENT, ErrEINVAL, etc.) matching POSIX names makes the API immediately familiar.

## What didn't
- The `Mode` type alias for `uint16` required explicit casts in the permission check methods. Minor friction but worth the type safety.

## What would I do differently
- Consider splitting `path.go` into two files: `errors.go` for sentinel errors and `path.go` for resolution + filesystem operations. The current file is manageable at ~200 lines but will grow as more operations are added in sections 2 and 3.
- The `InodeTable` capacity test (`TestInodeTableCapacity`) takes MaxInodes (4096) iterations. Could be slow if MaxInodes increases — consider a test-specific override or keeping MaxInodes modest.

## Implementation notes
- Filesystem operations (Mkdir, Create, Unlink, Rmdir) all live in `path.go` as methods on `FileSystem`. This centralizes path parsing/validation in `resolveParent()` and `cleanPath()`.
- `Inode.Data()` returns a copy (not the original slice) to prevent callers from mutating inode state directly. `SetData()` also copies.
- `Rmdir` checks for entries other than "." and ".." to determine emptiness — correct POSIX behavior.
- Unlink decrements Nlink and only frees the inode when both Nlink and RefCount reach zero, supporting the file descriptor table that will be added in section 2.

## pattern

- **[pattern]** (from SEC-1) [modified] bootstrap/interpreter.glyph

- **[pattern]** (from SEC-1) [added] pkg/vfs/path_test.go

- **[pattern]** (from SEC-1) [added] pkg/vfs/inode_test.go

- **[pattern]** (from SEC-1) [added] pkg/vfs/inode.go

- **[pattern]** (from SEC-1) [added] pkg/vfs/directory_test.go

## discovery

- **[discovery]** (from SEC-1) Agent strategy: created 6 files, modified 2 files, added tests, fix attempt

- **[discovery]** (from SEC-1) Tests improved by 1 (53 -> 54)

- **[pattern]** (from SEC-2) [modified] bootstrap/compiler.glyph

- **[discovery]** (from SEC-2) Agent strategy: modified 1 file

- **[pattern]** (from SEC-3) [modified] glyph

- **[discovery]** (from SEC-3) Agent strategy: modified 1 file
