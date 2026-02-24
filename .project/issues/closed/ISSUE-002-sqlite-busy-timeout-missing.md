# ISSUE-002: Missing PRAGMA busy_timeout causes SQLITE_BUSY errors under concurrent writes

**Severity**: HIGH
**Type**: Bug
**Discovered By**: security-engineer
**Discovered During**: Phase 4, Task 4.2.1 (confirmed by QA concurrent write test failures)
**Affected Files**: `internal/storage/db.go`
**Assigned To**: backend-engineer
**Status**: Resolved

---

## Description

The SQLite database connection in `NewDB()` enables WAL mode and foreign keys but does not set `PRAGMA busy_timeout`. Without a busy timeout, any concurrent write that encounters a database lock receives an immediate `SQLITE_BUSY` error instead of retrying after a short delay.

In production, ctsnare runs multiple poller goroutines that all call `UpsertHit()` concurrently. Without a busy timeout, these writes will intermittently fail with `SQLITE_BUSY` errors, silently dropping hits.

## Reproduction / Evidence

The QA engineer's `TestConcurrentReadWrite` test consistently fails with `SQLITE_BUSY` errors when 5 goroutines write simultaneously:

```
Error: upserting hit for writer2-hit0.com: database is locked (5) (SQLITE_BUSY)
```

Expected 100 inserts, got only 36-78 depending on the run — a 22-64% data loss rate under concurrent load.

## Impact

- **Data loss**: Hits discovered by pollers are silently dropped when concurrent writes collide
- **Reliability**: The more CT logs configured, the worse the problem gets (more concurrent writers)
- **Silent failure**: The poller logs a warning and continues, but the hit is permanently lost

## Suggested Fix

Add `PRAGMA busy_timeout` after enabling WAL mode in `internal/storage/db.go`:

```go
// Set busy timeout to wait up to 5 seconds for locks to clear.
if _, err := sqlDB.Exec("PRAGMA busy_timeout=5000"); err != nil {
    sqlDB.Close()
    return nil, fmt.Errorf("setting busy timeout: %w", err)
}
```

5000ms (5 seconds) is a reasonable default for a tool with multiple concurrent poller goroutines. This allows SQLite to internally retry when it encounters a lock, rather than immediately returning SQLITE_BUSY.

This should be placed in `NewDB()` between the WAL mode and foreign key PRAGMAs.

## Resolution

Added `PRAGMA busy_timeout=5000` in `internal/storage/db.go` NewDB() between WAL mode and foreign key PRAGMAs. SQLite now waits up to 5 seconds for locks to clear before returning SQLITE_BUSY, eliminating silent data loss under concurrent writes from multiple poller goroutines.

---

*Filed: 2026-02-24*
*Resolved: 2026-02-24*
