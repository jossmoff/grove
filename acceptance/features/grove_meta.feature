Feature: grove init and config — meta commands

  Scenario: grove init zsh prints gcd and gfin shell functions
    When I run "grove init zsh"
    Then the exit code is 0
    And stdout contains "gcd"
    And stdout contains "gfin"
    And stderr is empty

  Scenario: grove init bash prints the same functions
    When I run "grove init bash"
    Then the exit code is 0
    And stdout contains "gcd"
    And stdout contains "gfin"

  Scenario: grove init fish prints fish function syntax
    When I run "grove init fish"
    Then the exit code is 0
    And stdout contains "function gcd"
    And stdout contains "function gfin"

  Scenario: grove init with unknown shell is an error
    When I run "grove init powershell"
    Then the exit code is non-zero
    And stderr contains "grove:"

  Scenario: grove config prints TOML with known keys
    When I run "grove config"
    Then the exit code is 0
    And stdout contains "root"
    And stdout contains "grove_root"
    And stdout contains "branch_prefix"
    And stderr is empty

  Scenario: grove config --path prints the config file location
    When I run "grove config --path"
    Then the exit code is 0
    And stdout is a valid absolute path
    And stderr is empty
