You are tasked with implementing a new feature in an existing software project using Test-Driven Development (TDD), SOLID principles, and a functional approach. Your goal is to create high-quality, maintainable code that integrates seamlessly with the current codebase.

Now, let's implement the following feature:

<feature_description>
$ARGUMENTS.
</feature_description>

Follow these steps to implement the feature:

1. Analyze the current codebase:
    - Identify existing patterns and architectural decisions
    - Note any areas where SOLID principles or functional programming concepts are already in use

2. Implement the feature using Test-Driven Development (TDD):
    - Write a failing test that describes the expected behavior of the new feature
    - Implement the minimum code necessary to make the test pass
    - Refactor the code while keeping the tests passing

3. Apply SOLID principles:
    - Single Responsibility Principle: Ensure each class or function has only one reason to change
    - Open-Closed Principle: Design classes to be open for extension but closed for modification
    - Liskov Substitution Principle: Ensure derived classes can be substituted for their base classes
    - Interface Segregation Principle: Create specific interfaces instead of general-purpose ones
    - Dependency Inversion Principle: Depend on abstractions, not concretions

4. Incorporate functional programming concepts:
    - Use pure functions where possible
    - Avoid mutable state and side effects
    - Leverage higher-order functions and composition

5. Follow a small batch, iterative approach:
    - Break down the feature into small, manageable tasks
    - Implement each task using the red-green-blue cycle:
      a. Red: Write a failing test
      b. Green: Write the minimum code to make the test pass
      c. Blue: Refactor the code while keeping tests passing
    - Integrate and test each small batch before moving on to the next

6. Continuously refactor and improve the code:
    - Look for opportunities to apply design patterns
    - Ensure code readability and maintainability
    - Update documentation as needed

Your final output should include:

<output>
1. A brief summary of the current codebase analysis
2. A list of tests implemented for the new feature
3. The code for the new feature, following TDD, SOLID principles, and functional approach
4. A brief explanation of how SOLID principles and functional concepts were applied
5. Any refactoring or improvements made to the existing codebase
</output>

Ensure that your code is well-commented and follows the existing coding style of the project. Your final answer should only include the content specified in the <output> tags, without repeating any of the instructions or intermediate steps.