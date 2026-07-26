Feature: grove new — create a grove

  Background:
    Given a repo "backend" exists
    And a repo "client-sdk" exists

  Scenario: write repo gets a branch, read repo is detached
    When I run "grove new fix -w backend -r client-sdk --no-fetch"
    Then the exit code is 0
    And "fix/backend" is on branch "grove/fix"
    And "fix/client-sdk" is in detached HEAD state

  Scenario: read repo leaves no branches in its clone
    When I run "grove new myfix -w backend -r client-sdk --no-fetch"
    Then the exit code is 0
    And the repo "backend" has branch "grove/myfix"
    And the repo "client-sdk" has no branches matching "grove/*"

  Scenario: stdout is the grove path and nothing else
    When I run "grove new fix -w backend --no-fetch"
    Then the exit code is 0
    And stdout is exactly the grove path for "fix"

  Scenario: two groves can share the same repo via unique branch names
    When I run "grove new one -w backend --no-fetch"
    And I run "grove new two -w backend --no-fetch"
    Then the exit code is 0
    And "one/backend" is on branch "grove/one"
    And "two/backend" is on branch "grove/two"

  Scenario: duplicate slug is rejected
    Given a grove "fix" exists with "backend" as write
    When I run "grove new fix -w backend --no-fetch"
    Then the exit code is non-zero
    And stderr contains "already exists"

  Scenario: missing slug prints an error
    When I run "grove new"
    Then the exit code is non-zero
    And stderr contains "grove:"
