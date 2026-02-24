# Testing Standards

## Requirements

- Test behavior, not implementation
- Test error paths, not just happy paths
- Test edge cases and boundary conditions
- No test should depend on another test's state

## Structure

- Unit tests for pure logic and business rules
- Integration tests for system behavior and data flow
- E2E tests for critical user paths

## Practices

- Run tests after every change — zero failures before moving on
- Mock only at system boundaries (network, database, external APIs)
- No over-mocking that hides real bugs
- Test names describe the scenario and expected outcome
- Keep tests fast — slow tests don't get run
