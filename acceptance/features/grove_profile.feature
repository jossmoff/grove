Feature: grove profile — durable repo-set templates

  Background:
    Given a repo "backend" exists
    And a repo "client-sdk" exists

  Scenario: save a profile from the current grove
    Given a grove "fix" exists with "backend" as write and "client-sdk" as read
    When I run "grove profile save myprofile" inside grove "fix"
    Then the exit code is 0
    When I run "grove profile ls"
    Then stdout contains "myprofile"

  Scenario: create a grove from a saved profile
    Given a profile "fullstack" with "backend" as write and "client-sdk" as read
    When I run "grove new from-profile -p fullstack --no-fetch"
    Then the exit code is 0
    And "from-profile/backend" is on branch "grove/from-profile"
    And "from-profile/client-sdk" is in detached HEAD state

  Scenario: export and import round-trips a profile
    Given a profile "myprofile" with "backend" as write
    When I run "grove profile export myprofile" and capture as "myprofile.toml"
    And I run "grove profile import reimported myprofile.toml"
    Then the exit code is 0
    When I run "grove profile ls"
    Then stdout contains "reimported"

  Scenario: export never includes LOCAL.md content
    Given a profile "secret" with "backend" as write
    And the profile "secret" has a LOCAL.md containing "MY_SECRET_CRED"
    When I run "grove profile export secret"
    Then the exit code is 0
    And stdout does not contain "MY_SECRET_CRED"

  Scenario: rm deletes a profile
    Given a profile "tobedeleted" with "backend" as write
    When I run "grove profile rm tobedeleted"
    Then the exit code is 0
    When I run "grove profile ls"
    Then stdout does not contain "tobedeleted"
