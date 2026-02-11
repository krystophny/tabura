Feature: Advisory guardrails
  As a desktop user
  I want safety warnings without hard blocking in v1
  So that iteration speed stays high with explicit risk visibility

  @SCN-WARN-001 @happy_path
  Scenario: Risky global prompt emits advisory warning
    Given global context mode is active
    When prompt indicates potentially risky write intent
    Then warning is emitted and prompt still executes

  @SCN-WARN-002 @safety_guardrail
  Scenario: Execution mode without plan lock emits warning
    Given project execution mode is active
    And no plan lock hash is present
    When prompt is submitted
    Then plan lock missing warning is emitted

  @SCN-WARN-003 @failure_recovery
  Scenario: Warning persistence failure does not block turn
    Given a warning should be emitted
    When warning persistence fails
    Then turn still completes with runtime error guidance
