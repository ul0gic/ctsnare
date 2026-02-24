---
paths:
  - "**/routes/**"
  - "**/api/**"
  - "**/handlers/**"
  - "**/controllers/**"
  - "**/resolvers/**"
  - "**/*.graphql"
  - "**/*.gql"
---

# API Design Rules

## REST Conventions

- Use nouns for resources, HTTP verbs for actions — `GET /users` not `GET /getUsers`
- Plural resource names — `/users`, `/orders`, not `/user`, `/order`
- Nest for relationships — `/users/:id/orders`
- Use query params for filtering, sorting, pagination — not path segments
- Return appropriate status codes — 201 for created, 204 for no content, 404 for not found

## Error Responses

- Consistent error shape across all endpoints:
  ```json
  { "error": { "code": "VALIDATION_ERROR", "message": "human-readable", "details": [] } }
  ```
- Machine-readable error codes — not just HTTP status
- Never expose stack traces, internal paths, or implementation details in errors
- Validation errors return all field errors at once, not one at a time

## Versioning

- Version in the URL path (`/v1/`) or header — pick one per project
- Never break existing clients — additive changes only within a version
- Deprecate before removing — warn in response headers

## Input/Output

- Validate all input at the boundary — use Zod, Pydantic, or equivalent
- Consistent casing — camelCase for JSON, snake_case if Python ecosystem
- Pagination on all list endpoints — no unbounded queries
- Include `total`, `page`, `limit` in paginated responses
- Use ISO 8601 for dates — always UTC

## Security

- Authentication on every endpoint unless explicitly public
- Authorization checked at the handler level — not just middleware
- Rate limiting on all public endpoints
- No sensitive data in URLs or query params — use headers or body
- CORS configured explicitly — no wildcard `*` in production

## Performance

- No N+1 queries — use joins, batching, or dataloaders
- Timeouts on all downstream calls
- Cache headers on read-heavy endpoints
- Keep payloads lean — return only what the client needs
