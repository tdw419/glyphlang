# Virtual Filesystem (VFS) Layer

## Why

The GlyphLang VM currently has no file abstraction. Programs cannot read, write, or manage persistent data. Every real operating system needs a filesystem, and GlyphLang OS is no different. Without a VFS:

- Programs have no way to persist state between executions.
- There is no unified interface for accessing different data sources (files, databases, in-memory buffers).
- The bootstrap self-hosting compiler cannot read `.glyph` source files or write compiled output.

String support (change 002) provides the foundation for path names and file content representation. A VFS layer is the next step toward making the VM a usable runtime.

## What Changes

1. **Inode table in VM memory**: Add an inode-based file metadata table. Each inode tracks file type, size, permissions, and a pointer to its data blocks. This lives in a reserved region of VM memory.

2. **Path resolution engine**: Implement hierarchical path resolution with `/` separators. Support absolute and relative paths. Resolve `.` and `..` components. Path strings use the string support from change 002.

3. **File I/O opcodes**: Add new opcodes to the VM:
   - `OpOpen` — open a file by path, return a file descriptor
   - `OpClose` — close a file descriptor
   - `OpRead` — read bytes from a file descriptor into a buffer
   - `OpWrite` — write bytes from a buffer to a file descriptor
   - `OpSeek` — set the read/write position in a file

4. **Mount point system**: Support mounting different backends at path prefixes:
   - Real filesystem backend — maps to the host OS filesystem (for development)
   - Database backend — stores files as rows in a database (for persistence)
   - Memory buffer backend — ephemeral in-memory files (for tests and IPC)

## Impact

- Enables programs to read source files and write compiled output, unblocking the self-hosting compiler.
- Provides a uniform abstraction over different storage backends.
- Required foundation for IPC (change 009) and syscalls (change 011).
- Adds ~6 new opcodes to the VM.
- No changes to the Go compiler's code generation — these are runtime extensions in the VM.
