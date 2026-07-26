Feature: grove finish — tear a grove down

  Background:
    Given a repo "polywit" exists
    And a grove "work" exists with "polywit" as write

  Scenario: refuses uncommitted changes
    Given the worktree "work/polywit" has an uncommitted file "scratch.txt"
    When I run "grove finish work"
    Then the exit code is non-zero
    And stderr contains "uncommitted"
    And the grove "work" still exists

  Scenario: --force proceeds despite uncommitted changes
    Given the worktree "work/polywit" has an uncommitted file "scratch.txt"
    When I run "grove finish --force work"
    Then the exit code is 0
    And the grove "work" does not exist

  Scenario: finish never deletes write branches
    When I run "grove finish --force work"
    Then the exit code is 0
    And the repo "polywit" still has branch "grove/work"

  Scenario: finish prunes the worktree from the source repo
    When I run "grove finish --force work"
    Then the exit code is 0
    And the repo "polywit" has exactly 1 worktree entry

  Scenario: stdout is the grove root directory
    When I run "grove finish --force work"
    Then the exit code is 0
    And stdout is the grove root
