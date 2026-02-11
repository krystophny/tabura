Feature: Explicit mode transitions
  As a desktop user
  I want mode transitions to remain explicit and non-destructive
  So that state changes are always understandable

  @SCN-MODE-001 @happy_path
  Scenario: Valid mode transition updates visible state
    Given mode chipbar is visible
    When I choose a valid context and cognitive mode
    Then mode state updates and remains visible

  @SCN-MODE-002 @safety_guardrail
  Scenario: Invalid transition is rejected with guidance
    Given mode chipbar is visible
    When I request an invalid cognitive transition
    Then transition is rejected
    And guidance is shown

  @SCN-MODE-003 @failure_recovery
  Scenario: Transition failure leaves previous state intact
    Given a mode transition is in progress
    When runtime transition handling fails
    Then the previous mode state remains visible and active
