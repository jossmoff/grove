Feature: grove path — print a grove's path

  Background:
    Given a repo "backend" exists
    And a grove "fix" exists with "backend" as write

  Scenario: prints the path to stdout and nothing else
    When I run "grove path fix"
    Then the exit code is 0
    And stdout is exactly the grove path for "fix"
    And stderr is empty

  Scenario: non-existent slug is an error
    When I run "grove path no-such-grove"
    Then the exit code is non-zero
    And stderr contains "grove:"
    And stdout is empty
