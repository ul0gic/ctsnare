---
name: security-check
description: Run a security audit on the codebase or specific files against OWASP/CWE standards.
disable-model-invocation: true
---

# Security Audit

Delegate this task to the `security-engineer` agent. This is a **read-only analysis** — do not modify any files.

## Target

Audit: $ARGUMENTS

If no target specified, audit the entire codebase.

## Process

1. **Map attack surface** — identify all inputs, outputs, trust boundaries, API endpoints
2. **Check OWASP Top 10** — systematic pass against common vulnerabilities
3. **Review authentication** — token handling, session management, credential storage
4. **Review authorization** — RBAC/ABAC, privilege escalation paths, missing endpoint checks
5. **Scan for injection** — SQL, XSS, SSRF, command injection, path traversal
6. **Check secrets** — hardcoded keys, .env exposure, secrets in logs/errors
7. **Review dependencies** — known CVEs, abandoned packages, outdated libraries
8. **Check configs** — CORS, CSP, security headers, TLS settings

## Output

For each finding:
- Severity (Critical / High / Medium / Low / Info)
- File path and line reference
- Description of the vulnerability
- Reproduction steps or proof
- Remediation recommendation
- CWE/OWASP mapping

End with a summary table and prioritized fix list.
