## ADDED Requirements

### Requirement: Example capability exists as a smoke-test fixture

The system SHALL provide a placeholder capability under the example change directory so that tooling can exercise spec-parsing without referencing a real capability.

#### Scenario: Fixture is parseable
- **WHEN** `beads-plan compile openspec/changes/example --dry-run` runs
- **THEN** it succeeds and reports one section with one task
