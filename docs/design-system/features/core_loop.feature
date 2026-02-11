Feature: Core local prompt loop
  As a desktop user
  I want to submit prompts from the composer into a local session
  So that I can iterate quickly and review output artifacts

  @SCN-CORE-001 @happy_path
  Scenario: Prompt submission creates canonical text artifact
    Given an active local session exists
    When I submit a prompt from the composer
    Then the prompt is written to the active session
    And a canonical text artifact is persisted

  @SCN-CORE-002 @safety_guardrail
  Scenario: Missing session yields explicit recovery guidance
    Given no active local session exists
    When I submit a prompt from the composer
    Then no prompt is dispatched
    And I receive guidance to start or attach a session

  @SCN-CORE-003 @failure_recovery
  Scenario: Session write failure keeps prompt recoverable
    Given an active local session exists
    When session input write fails
    Then I receive a recoverable error state
    And prompt text remains available for retry
