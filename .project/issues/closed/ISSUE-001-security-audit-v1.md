# ISSUE-001: Security Audit v1 — Phase 4 Assessment

**Severity**: Overall HIGH (individual findings range LOW to HIGH)
**Type**: Security
**Discovered By**: security-engineer
**Discovered During**: Phase 4, Tasks 4.2.1–4.2.5
**Affected Files**: See individual findings below
**Assigned To**: backend-engineer (Phase 5 remediation)
**Status**: Resolved

---

## Executive Summary

The ctsnare codebase demonstrates solid security fundamentals. SQL injection defenses are correctly implemented with parameterized queries and an allowlist for dynamic ORDER BY columns. One HIGH-severity finding was identified: missing `PRAGMA busy_timeout` causes silent data loss under concurrent writes (confirmed by QA test failures). Two MEDIUM findings relate to defense-in-depth hardening in the HTTP poller layer. Three LOW/informational findings are documented for completeness.

**govulncheck**: No known vulnerabilities in dependencies.
**go mod verify**: All modules verified — no integrity issues.

---

## Findings

### FINDING-01: Unbounded HTTP Response Body Read (MEDIUM)

**CVSS 3.1**: 5.3 (Medium) — AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H
**Affected File**: `internal/poller/ctlog.go:111-148` (doGet method)
**CWE**: CWE-400 (Uncontrolled Resource Consumption)

**Description**:
The `doGet` method returns `resp.Body` (an `io.ReadCloser`) directly to callers (`GetSTH`, `GetEntries`) without limiting the response body size. The callers then decode JSON from the body using `json.NewDecoder(body).Decode()`.

If a CT log server (or a man-in-the-middle attacker) returns a very large response body, the JSON decoder will attempt to read the entire body into memory, potentially causing out-of-memory conditions and process crash.

**Evidence**:
```go
// ctlog.go:126-127 — body returned without size limit
if resp.StatusCode == http.StatusOK {
    return resp.Body, nil
}
```

Callers decode directly:
```go
// ctlog.go:61 — unbounded decode
json.NewDecoder(body).Decode(&sth)

// ctlog.go:79 — unbounded decode
json.NewDecoder(body).Decode(&resp)
```

**Impact**:
A compromised or malicious CT log server could send a multi-gigabyte response, exhausting the host's memory and crashing the ctsnare process. In a long-running monitoring scenario, this could disrupt surveillance operations.

**Mitigating Factors**:
- CT log URLs are configured locally (hardcoded defaults or TOML config file), not user-supplied at runtime
- Requires a compromised or malicious CT log server, or active network interception
- The HTTP client has a 30-second timeout, which limits (but does not prevent) large downloads on fast connections

**Suggested Fix**:
Wrap the response body with `io.LimitReader` before returning it. A reasonable limit for CT log API responses is 50-100 MB (large batches of entries can be several megabytes):

```go
const maxResponseBodySize = 50 * 1024 * 1024 // 50 MB

if resp.StatusCode == http.StatusOK {
    limited := io.NopCloser(io.LimitReader(resp.Body, maxResponseBodySize))
    return limited, nil
}
```

Note: This requires wrapping in a struct that implements `io.ReadCloser` to preserve the `Close` method, or using a helper that chains the close.

---

### FINDING-02: HTTP Client Follows Redirects Without Restriction (MEDIUM)

**CVSS 3.1**: 4.3 (Medium) — AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:L
**Affected File**: `internal/poller/ctlog.go:42-44` (NewCTLogClient)
**CWE**: CWE-918 (Server-Side Request Forgery)

**Description**:
The `http.Client` created in `NewCTLogClient` uses the default redirect policy, which follows up to 10 redirects automatically. A compromised CT log server (or DNS hijack) could return an HTTP redirect to an internal network address (e.g., `http://169.254.169.254/latest/meta-data/` on AWS, or `http://127.0.0.1:8080/admin`), causing ctsnare to make requests to internal services.

**Evidence**:
```go
// ctlog.go:42-44 — default http.Client follows redirects
httpClient: &http.Client{
    Timeout: 30 * time.Second,
},
```

The default `CheckRedirect` policy follows up to 10 redirects. No custom redirect policy is set.

**Impact**:
In cloud environments, SSRF via redirect could expose cloud metadata endpoints (AWS IMDSv1, GCP metadata). On corporate networks, it could probe internal services. The impact is limited because:
- CT log URLs come from trusted configuration
- The response is parsed as JSON (CT log format), so data exfiltration via this vector is unlikely
- The tool runs locally, not as a server, reducing the attack surface

**Mitigating Factors**:
- URLs come from local config, not from external user input
- This is a client-side CLI tool, not a server
- Cloud metadata SSRF requires the tool to be running in a cloud environment with IMDSv1 enabled

**Suggested Fix**:
Disable redirects entirely (CT log APIs should not redirect) or validate redirect targets:

```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse
    },
},
```

---

### FINDING-03: fmt.Sprintf Used for ORDER BY Clause Construction (LOW / Informational)

**CVSS 3.1**: 0.0 (Informational — not exploitable)
**Affected File**: `internal/storage/hits.go:159`
**CWE**: CWE-89 (SQL Injection) — Mitigated

**Description**:
The `QueryHits` function constructs the ORDER BY clause using `fmt.Sprintf`:

```go
query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)
```

Both `sortBy` and `sortDir` are sanitized before interpolation:
- `sortBy` passes through `sanitizeSortColumn()` which uses a strict allowlist (hits.go:241-256)
- `sortDir` is validated against "ASC", defaulting to "DESC" (hits.go:156-158)

**Evidence**:
The current implementation is **not vulnerable** to SQL injection. The `sanitizeSortColumn` allowlist correctly prevents arbitrary column names, and the sort direction is binary (ASC or DESC).

**Impact**: None — this is not exploitable.

**Rationale for Reporting**:
This is a code smell that could become vulnerable if future maintainers modify the allowlist or sort direction validation incorrectly. Using `fmt.Sprintf` to construct SQL is a pattern that code scanners will flag, and it requires comments or documentation to explain why it's safe.

**Suggested Fix (Optional)**:
Add a comment explaining the safety invariant:

```go
// SECURITY: sortBy is sanitized through sanitizeSortColumn() allowlist;
// sortDir is limited to "ASC"/"DESC" by the check above. Both are safe
// for direct interpolation. ORDER BY does not support parameterized placeholders.
query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)
```

---

### FINDING-04: No Config File Size Limit (LOW / Informational)

**CVSS 3.1**: 0.0 (Informational)
**Affected File**: `internal/config/config.go:67`
**CWE**: CWE-400 (Uncontrolled Resource Consumption)

**Description**:
`config.Load()` reads the config file with `os.ReadFile(path)` which loads the entire file into memory. There is no file size check. A very large config file could cause memory exhaustion.

```go
data, err := os.ReadFile(path)
```

**Impact**: Negligible. This is a local CLI tool where the config file path is provided by the user. A user would not DoS themselves. This would only matter if ctsnare were integrated into a service that accepts untrusted config files.

**Suggested Fix**: No action needed for current use case. If ctsnare is ever used as a library or service, add a size check:

```go
fi, err := os.Stat(path)
if err == nil && fi.Size() > 1<<20 { // 1 MB limit
    return nil, fmt.Errorf("config file too large: %d bytes", fi.Size())
}
```

---

### FINDING-05: DB Path From Config Not Validated (LOW / Informational)

**CVSS 3.1**: 0.0 (Informational)
**Affected File**: `internal/config/config.go:87-96`, `internal/storage/db.go:23-25`
**CWE**: CWE-22 (Path Traversal) — Not exploitable in current context

**Description**:
The `db_path` value from the TOML config file is passed directly to `storage.NewDB()`, which calls `os.MkdirAll()` on the parent directory and `sql.Open()` on the path. No validation is performed to ensure the path is within expected directories.

If a user clones a repository containing a malicious `.ctsnare.toml` with `db_path = "/etc/cron.d/evil"`, the tool would attempt to create directories and a SQLite database at that path (though it would typically fail due to permissions).

**Impact**: Negligible for current use case. The user runs the tool locally and controls the config. The `--db` flag is explicitly user-provided. Directory creation uses 0700 permissions.

**Suggested Fix**: No action required for a CLI tool. If paranoia is desired, validate that the resolved path is under `$HOME` or `$XDG_DATA_HOME`.

---

### FINDING-06: Missing PRAGMA busy_timeout Causes Data Loss Under Concurrent Writes (HIGH)

**CVSS 3.1**: 6.5 (Medium) — AV:L/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N
**Affected File**: `internal/storage/db.go:23-52`
**CWE**: CWE-362 (Concurrent Execution Using Shared Resource with Improper Synchronization)

**Description**:
The SQLite database connection enables WAL mode but does not set `PRAGMA busy_timeout`. Without this, concurrent writes from multiple poller goroutines receive immediate `SQLITE_BUSY` errors instead of retrying. This causes silent data loss — discovered hits are logged as warnings and dropped.

**Evidence**:
QA concurrent write tests show 22-64% data loss rates. Expected 100 inserts from 5 concurrent writers, got 36-78. Each `SQLITE_BUSY` error means a legitimate hit was permanently lost.

**Impact**: Silent data loss in production. The more CT logs configured (more concurrent poller goroutines), the worse the problem. Hits are permanently lost with only a warning log entry.

**Suggested Fix**: Add `PRAGMA busy_timeout=5000` in `NewDB()`. See `ISSUE-002-sqlite-busy-timeout-missing.md` for details.

---

## Positive Observations

The following security practices are well-implemented:

1. **Parameterized SQL Queries**: All database queries in `internal/storage/` use `?` placeholder parameters. No string concatenation is used for query values. The `UpsertHit`, `InsertHit`, `QueryHits`, `ClearSession`, and `Stats` methods are all safe from SQL injection.

2. **Sort Column Allowlist**: `sanitizeSortColumn()` uses a strict map-based allowlist that returns a safe default ("created_at") for unrecognized inputs. This is the correct pattern for dynamic ORDER BY.

3. **Sort Direction Validation**: Binary check against "ASC" with "DESC" as default. Cannot be exploited.

4. **HTTP Client Timeout**: The CT log HTTP client uses a 30-second timeout, preventing indefinite hangs.

5. **Rate Limit Handling**: The poller implements exponential backoff for HTTP 429 responses with a maximum retry count, preventing infinite retry loops.

6. **Graceful Error Handling**: Certificate parsing errors are logged and skipped rather than causing panics. The `extractCertFromLeaf` function has proper bounds checking at every step.

7. **Context Propagation**: All long-running operations accept and respect `context.Context`, enabling clean cancellation and timeout propagation.

8. **Directory Permissions**: Database directory creation uses 0700 (owner-only) permissions.

9. **No Secrets in Code**: No hardcoded API keys, tokens, or credentials. The tool communicates only with public CT log endpoints over HTTPS.

10. **WAL Mode**: SQLite is opened in WAL mode with foreign keys enabled, providing crash safety and concurrent access support.

11. **Dependency Hygiene**: `govulncheck` reports zero known vulnerabilities. `go mod verify` confirms module integrity. The dependency set is minimal and well-chosen.

---

## Strategic Recommendations

1. **Add `io.LimitReader` to HTTP responses** (FINDING-01) — Straightforward fix, high defensive value. Priority: Phase 5.

2. **Disable HTTP redirects for CT log client** (FINDING-02) — Simple one-line fix. CT log APIs should never redirect. Priority: Phase 5.

3. **Add security comment to ORDER BY construction** (FINDING-03) — Documentation improvement. Priority: Low.

4. **Consider adding a `User-Agent` header** to CT log requests — Good practice for API clients, helps log operators identify legitimate monitoring tools.

5. **Consider structured error types** for security-relevant failures (auth failures, rate limits, connection errors) — Would improve monitoring and alerting in production deployments.

---

## Methodology

**Scope**: Full source code audit of `internal/storage/`, `internal/poller/`, and `internal/config/` packages, plus review of `internal/cmd/` for input handling. Read-only assessment — no source code modifications.

**Tools Used**:
- Manual source code review (line-by-line analysis)
- `govulncheck ./...` — Dependency vulnerability scanning (Go official tool)
- `go mod verify` — Module integrity verification
- Pattern matching for common vulnerability signatures (fmt.Sprintf in SQL context, os.ReadFile without limits, http.Client without redirect policy)

**Coverage**:
- SQL injection: All database operations in `internal/storage/` — 100% coverage
- SSRF: All HTTP operations in `internal/poller/` — 100% coverage
- Path traversal: All file path handling in `internal/config/` and `internal/storage/` — 100% coverage
- Input validation: All user-facing input paths via CLI flags through `internal/cmd/` — 100% coverage
- Dependency vulnerabilities: Full dependency tree via govulncheck — 100% coverage

---

*Filed: 2026-02-24*
*Resolved: 2026-02-24 — All CRITICAL/HIGH/MEDIUM findings addressed. LOW/informational findings documented with safety comments.*

### Resolution Summary

- **FINDING-01 (MEDIUM)**: Fixed. Added `io.LimitReader` with 50 MB cap and `limitedReadCloser` wrapper in `internal/poller/ctlog.go` doGet method.
- **FINDING-02 (MEDIUM)**: Fixed. Set `CheckRedirect` on HTTP client to return `http.ErrUseLastResponse`, disabling all redirects.
- **FINDING-03 (LOW)**: Fixed. Added security invariant comment explaining allowlist safety above ORDER BY interpolation in `internal/storage/hits.go`.
- **FINDING-04 (LOW)**: No action needed — informational for CLI tool.
- **FINDING-05 (LOW)**: No action needed — informational for CLI tool.
- **FINDING-06 (HIGH)**: Fixed. Added `PRAGMA busy_timeout=5000` in `internal/storage/db.go` NewDB(). See ISSUE-002 for details.
