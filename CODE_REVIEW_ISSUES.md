# GlyphLang Code Review Issues

> Updated: 2026-03-28
> Status: Most P0 and P1 issues have been resolved

## P0 -- Release Blockers

### All P0 Issues Resolved ✅

| Issue | Status | Fix |
|-------|--------|-----|
| P0-1: `time.now()` hardcoded | ✅ Fixed | Uses `time.Now().Unix()` |
| P0-2: Async data race | ✅ Fixed | Copies maps before goroutine |
| P0-3: Struct field defaults | ✅ Fixed | Parser handles `=` for defaults |
| P0-4: Variable reassignment | ✅ Fixed | Supports bare `x = value` |
| P0-5: No body size limit | ✅ Fixed | `http.MaxBytesReader` 10MB |
| P0-6: Error exposure | ✅ Fixed | User-safe error messages |
| P0-7: WebSocket origins | ✅ Fixed | `CheckOrigin` validates list |

---

## P1 -- Security & Correctness

### P1-1: VM string builtins are ASCII-only ✅ FIXED

**Status:** Fixed - Uses Go's unicode-aware stdlib (`strings.ToLower`, `strings.Contains`, etc.)

### P1-2: `SanitizeSQL` function is fundamentally flawed ⚠️ DEPRECATED

**Status:** Deprecated with warning. Use parameterized queries instead.

### P1-3: `EscapeHTML` uses non-deterministic map iteration ✅ FIXED

**Status:** Fixed in `pkg/security/xss.go` - Now uses `html.EscapeString(s)` directly.

### P1-4: `SendFile` lacks path traversal protection ✅ FIXED

**Status:** Fixed in `pkg/web/web.go` - Uses `isSubPath()` check after `filepath.EvalSymlinks()`.

### P1-5: `X-Forwarded-For` header trusted unconditionally ✅ FIXED

**Status:** Fixed - `TrustProxy` flag defaults to `false`, must be explicitly enabled.

### P1-6: Rate limiter maps grow unbounded ✅ FIXED

**Status:** Fixed in `pkg/server/middleware.go` - Background cleanup every 60s, max 10000 entries cap.

### P1-7: No CSRF protection middleware ✅ FIXED

**Status:** Fixed - `CSRFMiddleware()` implemented with cookie + header/form validation.

### P1-8: `GetLastInsertID` uses original values instead of sanitized ⚠️ LOW RISK

**Status:** Low priority - Only affects specific PostgreSQL use case. Parameterized queries recommended.

### P1-9: Raw SQL `Query()` exposed to interpreter ⚠️ BY DESIGN

**Status:** By design - Developer responsibility. Documented in API. Use parameterized queries.

### P1-10: Unbounded LLM response body read ✅ FIXED

**Status:** Fixed in `pkg/llm/handler.go:616` - Uses `io.LimitReader` with 50MB cap.

### P1-11: WebSocket default config allows unlimited connections ✅ FIXED

**Status:** Fixed in `pkg/websocket/config.go` - Defaults: 10000 per hub, 1000 per room.

---

## P2 -- Code Quality

These should be addressed before v1.0.

- P2-1: Run `gofmt` across codebase (72 files have formatting issues)
- P2-2: Split `cmd/glyph/main.go` (2,905 lines) into multiple files
- P2-3: Decompose `evaluateFunctionCall` (618 lines)
- P2-4: Move AST types out of `pkg/interpreter`
- P2-5: Remove `MockInterpreter` from production code
- P2-6: Replace stub tests with real assertions
- P2-7: Resolve all 31 TODO comments
- P2-8: Remove dead code
- P2-9: Consistent pointer vs value types in compiler switch
- P2-10: Make version string injectable via ldflags
- P2-11: Add CI enforcement for formatting and linting
- P2-12: `os.Exit` called from non-main functions
- P2-13: Password included in connection string (logging risk)

---

## Summary

| Priority | Total | Fixed | Remaining |
|----------|-------|-------|-----------|
| P0 | 7 | 7 | 0 |
| P1 | 11 | 8 | 3 (low risk/by design) |
| P2 | 13 | 0 | 13 |

**All P0 release blockers and most P1 security issues have been resolved!**

---

## Next Steps

1. **Self-Compilation** - Make compiler.glyph compile itself (v0.8.0 milestone)
2. **GPU Native** - Port bytecode execution to WebGPU compute shaders
3. **P2 Code Quality** - gofmt, split large files, CI enforcement
4. **Documentation** - Update README with v0.7.0 features
