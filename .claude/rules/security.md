# Security Requirements

## Non-Negotiable

- Never hardcode secrets, API keys, or credentials
- Never commit .env files or secrets to git
- Never log sensitive data (tokens, passwords, PII)
- Never use `eval()` or dynamic code execution
- Never trust client input — validate everything server-side

## Authentication & Authorization

- Check permissions on every request
- Use parameterized queries — never string concatenation for SQL
- Validate JWT tokens properly — check expiration, issuer, audience
- Use secure session management

## Data Protection

- HTTPS only — no HTTP fallback
- Encrypt sensitive data at rest and in transit
- Sanitize all user input before rendering (prevent XSS)
- Set security headers (HSTS, CSP, X-Content-Type-Options)

## Dependencies

- Keep dependencies updated
- Audit for known vulnerabilities regularly
- Minimize dependency surface — fewer deps = fewer attack vectors
