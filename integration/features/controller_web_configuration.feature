Feature: Controller Web Configuration
  For users to work with the system easily
  They must be able to configure it from the web page

  Scenario: See Configuration
    Given controller is started
    When I navigate to /configuration
    Then the page should have an input with type text and name port and value 9400