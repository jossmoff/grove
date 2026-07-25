Feature: grove sync — regenerate agent context

  Background:
    Given a repo "backend" exists
    And a grove "fix" exists with "backend" as write

  Scenario: regenerates a deleted AGENTS.md
    Given the file "fix/AGENTS.md" has been deleted
    When I run "grove sync fix"
    Then the exit code is 0
    And the file "fix/AGENTS.md" exists
    And stderr contains "regenerated"

  Scenario: AGENTS.md routes to per-repo CLAUDE.md and does not inline it
    When I run "grove sync fix"
    Then the exit code is 0
    And the file "fix/AGENTS.md" contains "backend/CLAUDE.md"
    And the file "fix/AGENTS.md" does not contain "instructions for"
