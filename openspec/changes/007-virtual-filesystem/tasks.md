# Tasks: Virtual Filesystem (VFS) Layer

## 1. Inode table and path resolution
- [x] 1.1 Define the inode struct in VM memory: file type (regular/dir/mount), size, data pointer, permissions, reference count. Reserve a fixed region in the VM address space for the inode table.
- [x] 1.2 Implement path resolution that splits `/`-delimited paths, resolves `.` and `..`, and walks the directory tree from root. Return ENOENT/EINVAL errors for invalid paths.
- [x] 1.3 Add directory inode support: list entries, create/delete entries, maintain parent links for `..` traversal.

## 2. File I/O opcodes
- [ ] 2.1 Add `OpOpen`, `OpClose`, `OpRead`, `OpWrite`, `OpSeek` opcodes to the VM opcode table. Each opcode takes the appropriate operands (path string for open, fd + buffer for read/write).
- [ ] 2.2 Implement file descriptor table per VM instance: allocate fds on open, release on close, enforce fd limits (default 256 per process).
- [ ] 2.3 Wire read/write through the mount backend dispatch so operations go to the correct backend based on the file's mount point.

## 3. Mount point backends
- [ ] 3.1 Implement the memory buffer backend: files stored as byte slices in a map. Used for `/tmp` and test fixtures. Zero persistence overhead.
- [ ] 3.2 Implement the real filesystem backend: translate VFS paths to host OS paths within a chroot-like base directory. Read/write delegate to host OS syscalls.
- [ ] 3.3 Implement the database backend: serialize file data to a key-value store table keyed by inode number. Support lazy loading of file contents.
