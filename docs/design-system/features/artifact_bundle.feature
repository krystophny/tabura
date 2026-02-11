Feature: Artifact bundle review
  As a desktop user
  I want canonical turn artifacts for every meaningful prompt
  So that output is reviewable and attributable

  @SCN-ART-001 @happy_path
  Scenario: Canonical text artifact is always present
    Given a prompt turn completes
    When artifacts are persisted
    Then canonical text artifact exists

  @SCN-ART-002 @safety_guardrail
  Scenario: Diff artifacts are optional and explicit
    Given a prompt turn completes
    When no patch-like output exists
    Then canonical text exists without synthetic diff artifact

  @SCN-ART-003 @failure_recovery
  Scenario: Artifact write failure emits recoverable warning
    Given a prompt turn completes
    When artifact file persistence fails
    Then runtime emits warning and keeps turn metadata
