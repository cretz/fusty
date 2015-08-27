Feature: Controller CLI Help
  In order for users to see what options are supported
  They need to be able to invoke "help"

  Scenario: Run Help
    Given I am at a terminal
    When I run fusty with parameters "help"
    Then the output should contain "Usage"