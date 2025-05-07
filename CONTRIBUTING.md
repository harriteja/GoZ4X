# Contributing to GoZ4X

Thank you for your interest in contributing to GoZ4X! This document outlines the process for contributing to the project and how to get started.

## Code of Conduct

This project and everyone participating in it is governed by the GoZ4X Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally (`git clone https://github.com/yourusername/GoZ4X.git`)
3. Add the original repository as an upstream remote (`git remote add upstream https://github.com/harriteja/GoZ4X.git`)
4. Create a new branch for your changes (`git checkout -b feature/your-feature-name`)

## Development Process

1. Make your changes, following the coding style and guidelines
2. Add tests for your changes
3. Run the test suite to ensure all tests pass
4. Run benchmarks to check performance impact
5. Commit your changes with descriptive commit messages
6. Push to your fork (`git push origin feature/your-feature-name`)
7. Submit a pull request to the main repository

## Pull Request Process

1. Update the README.md or documentation with details of changes if applicable
2. Include benchmark results before/after your changes if performance-related
3. The PR should work on Go 1.21 and later
4. The PR will be merged once it receives approval from maintainers

## Coding Standards

- Follow standard Go code formatting (run `go fmt ./...` before committing)
- Add comments for public API functions and any complex logic
- Write tests for all new functionality
- Include benchmarks for performance-critical code
- Maintain backward compatibility unless explicitly breaking in a major version

## Benchmarking

For any performance-related changes, please include benchmark results:

```
# Before
BenchmarkMyFunction-8     10000000     150 ns/op      64 B/op      1 allocs/op

# After
BenchmarkMyFunction-8     20000000      75 ns/op      64 B/op      1 allocs/op
```

## Issue Reporting

- Use the GitHub issue tracker to report bugs
- Describe the bug clearly, including steps to reproduce
- Include Go version, OS, and other relevant environment details
- If possible, provide a minimal test case that reproduces the issue

## Feature Requests

- Use the GitHub issue tracker with the "enhancement" label
- Describe the feature and the problem it solves
- Explain why this feature would be useful to the project

## Review Process

All submissions require review. We use GitHub pull requests for this purpose. 
Consult [GitHub Help](https://help.github.com/articles/about-pull-requests/) for more information on using pull requests.

## License

By contributing to GoZ4X, you agree that your contributions will be licensed under the project's MIT License. 