# GlyphLang Package Manager MVP

## Goal

Enable real-world projects with external dependencies. Without a package manager, every serious project hits a wall.

## Design Principles

1. **Start simple** - Leverage Go modules initially
2. **GitHub as registry** - No custom registry needed
3. **Single source of truth** - `glyph.mod` file
4. **Interoperable** - Can import Go packages when needed

## File Format: glyph.mod

```glyph
module github.com/user/my-api

glyph 0.6.0

require (
    github.com/glyphlang/stdlib v0.1.0
    github.com/glyphlang/auth v0.2.0
)
```

## CLI Commands

### `glyph mod init`
Create a new `glyph.mod` file in the current directory.

### `glyph mod add <package>`
Add a dependency:
```bash
glyph mod add github.com/glyphlang/stdlib
```

### `glyph mod tidy`
Clean up unused dependencies and download missing ones.

### `glyph mod verify`
Verify dependencies against their checksums.

## Package Structure

A GlyphLang package is a directory containing:
```
my-package/
├── glyph.mod        # Required
├── main.glyph       # Entry point (for executables)
├── lib.glyph        # Library code
└── README.md        # Documentation
```

## Import Syntax

```glyph
# Import a module
% stdlib: github.com/glyphlang/stdlib
% auth: github.com/glyphlang/auth

@ GET /protected {
  % user: auth.require()
  > {user: user}
}
```

## Implementation Phases

### Phase 1: File Format (1 day)
- Define `glyph.mod` parser
- Create `ModFile` struct
- Add `glyph mod init` command

### Phase 2: Dependency Resolution (2 days)
- Parse `require` block
- Resolve versions from GitHub tags
- Download to `~/.glyph/cache/`

### Phase 3: Import Integration (2 days)
- Extend parser to support `% module: url` imports
- Wire imports to cached packages
- Support relative imports (`./lib.glyph`)

### Phase 4: Build Integration (1 day)
- `glyph build` resolves deps before compiling
- Bundle dependencies into output binary
- Support `--static` flag for fully self-contained binaries

## MVP Scope

For v0.6.0, ship:
- ✅ `glyph.mod` file format
- ✅ `glyph mod init`
- ✅ `glyph mod add`
- ✅ `glyph mod tidy`
- ✅ GitHub-based package resolution
- ✅ Basic import syntax

Future (v0.7.0+):
- Custom registry support
- Private package support
- Version locking
- Dependency caching with checksums

## Example Session

```bash
$ mkdir my-api && cd my-api
$ glyph mod init
Created glyph.mod

$ glyph mod add github.com/glyphlang/stdlib
Added github.com/glyphlang/stdlib v0.1.0

$ glyph mod add github.com/glyphlang/auth
Added github.com/glyphlang/auth v0.2.0

$ cat glyph.mod
module github.com/user/my-api

glyph 0.6.0

require (
    github.com/glyphlang/stdlib v0.1.0
    github.com/glyphlang/auth v0.2.0
)

$ glyph run main.glyph
Server started on :3000
```

## Why Not Just Use Go Modules?

Go modules work great for Go code. But GlyphLang needs its own package format for:

1. **GlyphLang-specific metadata** - entry points, route definitions
2. **Simpler dependency spec** - no complex version constraints
3. **Future extensions** - native spatial packages, WASM modules
4. **AI-friendly format** - minimal syntax, easy for LLMs to generate

The MVP uses GitHub as the registry (like Go) but with GlyphLang-native tooling.
