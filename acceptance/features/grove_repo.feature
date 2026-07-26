Feature: grove repo — manage the repo collection

  # grove repo clone requires a URL with a host component (e.g. git@host:owner/repo
  # or https://host/owner/repo). Local file paths are intentionally rejected because
  # they have no canonical position under the root. Clone is therefore covered by
  # the workspace lifecycle tests rather than these acceptance tests.

  Scenario: repo ls lists repos found under the root
    Given a repo "polywit" exists
    When I run "grove repo ls --refresh"
    Then the exit code is 0
    And stdout contains "polywit"

  Scenario: repo ls --long includes the remote URL
    Given a repo "polywit" exists
    When I run "grove repo ls --refresh --long"
    Then the exit code is 0
    And stdout contains "polywit"
