Feature: grove ls — list groves

  Scenario: no groves prints narration to stderr, stdout is silent
    When I run "grove ls"
    Then the exit code is 0
    And stdout is empty
    And stderr contains "no groves"

  Scenario: lists slug and branch for each grove
    Given a repo "backend" exists
    And a grove "myfix" exists with "backend" as write
    When I run "grove ls"
    Then the exit code is 0
    And stdout contains "myfix"
    And stdout contains "grove/myfix"

  Scenario: --json emits a valid JSON array containing the slug
    Given a repo "backend" exists
    And a grove "myfix" exists with "backend" as write
    When I run "grove ls --json"
    Then the exit code is 0
    And stdout is valid JSON
    And the JSON contains slug "myfix"
