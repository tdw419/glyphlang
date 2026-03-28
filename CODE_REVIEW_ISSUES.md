# GlyphLang Code Review Issues

> Updated: 2026-03-28
> Status: Most P0 issues have been resolved

## P0 -- Release Blockers

### P0-1: `time.now()` and `now()` return hardcoded value ✅ FIXED

**Status:** Fixed in `pkg/vm/vm.go:1388`

The implementation now correctly uses `time.Now().Unix()`.

### P0-2: Async execution data race in VM ✅ FIXED

**Status:** Fixed in `pkg/vm/vm.go:1983`

The code now creates copies of `constants`, `locals`, `globals`, and `builtins` before launching the goroutine, No more concurrent map access.

### P0-3: Parser does not support struct field default values ✅ FIXED

**Status:** Fixed in `pkg/parser/parser.go:616`

The parser now handles `=` for default values in struct field definitions.

### P0-4: Parser does not support variable reassignment without `$` prefix ✅ FIXED

**Status:** Fixed in `pkg/parser/parser.go:2233`

The parser now supports bare `x = value` reassignment syntax.

### P0-5: No request body size limit in core server handler ✅ FIXED

**Status:** Fixed in `pkg/server/handler.go:111`

The code now uses `http.MaxBytesReader` with a 10MB limit.

### P0-6: Internal error details exposed to clients ✅ FIXED

**Status:** Fixed in `pkg/server/handler.go:160`

The `handleError` function no longer includes `err.Error()` in client responses.

### P0-7: WebSocket upgrader accepts all origins ✅ FIXED

**Status:** Fixed in `pkg/websocket/config.go:154`

The `CheckOrigin` function now validates origins against the `AllowedOrigins` list.

---

## P1 -- Security & Correctness

These should be addressed before v1.0.

### P1-1: VM string builtins are ASCII-only,**File:** `pkg/vm/vm.go:1474-1621`

The VM string functions use custom ASCII-only implementations instead of Go's unicode-aware stdlib.

### P1-2: `SanitizeSQL` function is fundamentally flawed
**File:** `pkg/security/sql_injection.go:133-140`

Regex-based SQL sanitization is not a reliable security measure.

### P1-3: `EscapeHTML` uses non-deterministic map iteration
**File:** `pkg/security/xss.go:293-307`

Map iteration order is non-deterministic, Should use `html.EscapeString`.

### P1-4: `SendFile` lacks path traversal protection
**File:** `pkg/web/web.go:401-421`

No validation that the path is within the allowed root directory.

### P1-5: `X-Forwarded-For` header trusted unconditionally
**File:** `pkg/server/middleware.go:318-332`

Should only trust X-Forwarded-For from configured trusted proxies.

### P1-6: Rate limiter maps grow unbounded
**File:** `pkg/server/middleware.go:342-398`

Maps grow indefinitely without cleanup.

### P1-7: No CSRF protection middleware
**Files:** `pkg/server/middleware.go` (absent)

No CSRF token generation/validation.

### P1-8: `GetLastInsertID` uses original values instead of sanitized
**File:** `pkg/database/postgres.go:354-376`

Uses unsanitized values in query despite validation.

### P1-9: Raw SQL `Query()` exposed to interpreter
**File:** `pkg/database/handler.go:142-145`

Raw SQL queries accessible without safeguards.

### P1-10: Unbounded LLM response body read
**File:** `pkg/llm/handler.go:569`

No size limit on LLM response body.

### P1-11: WebSocket default config allows unlimited connections
**File:** `pkg/websocket/config.go:53-54`

Default `MaxConnectionsPerHub` and `MaxConnectionsPerRoom` are 0 (unlimited).

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
| P1 | 11 | 0 | 11 |
| P2 | 13 | 0 | 13 |

**All P0 release blockers have been resolved!**
