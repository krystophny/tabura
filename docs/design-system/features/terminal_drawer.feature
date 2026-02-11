Feature: Terminal drawer continuity
  As a desktop user
  I want to hide and reopen terminal without session loss
  So that terminal remains a secondary surface

  @SCN-TERM-001 @happy_path
  Scenario: Drawer reopen preserves session continuity
    Given an active session exists
    When I close and reopen terminal drawer
    Then the same session remains active

  @SCN-TERM-002 @safety_guardrail
  Scenario: Session scope remains explicit while toggling drawer
    Given terminal drawer is available
    When I toggle drawer visibility
    Then session identity remains explicit

  @SCN-TERM-003 @failure_recovery
  Scenario: Lost attachment offers reattach action
    Given drawer opens with missing session attachment
    When runtime detects missing attachment
    Then recovery actions are shown
