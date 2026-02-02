# Contributing to Jimmy.ai

Thank you for your interest in contributing to Jimmy.ai! This document provides guidelines and instructions for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/jimmy.ai.git`
3. Install Go 1.23 or later
4. Run `make deps` to download dependencies
5. Run `make build` to build the project

## Development Workflow

1. Create a new branch for your feature: `git checkout -b feature/my-feature`
2. Make your changes
3. Run tests: `make test`
4. Format code: `make fmt`
5. Commit your changes with a descriptive message
6. Push to your fork
7. Create a Pull Request

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions focused and small

## Testing

- Write tests for new features
- Ensure all tests pass before submitting PR
- Aim for good test coverage

## Commit Messages

Use conventional commit format:
- `feat: add new feature`
- `fix: fix bug`
- `docs: update documentation`
- `refactor: refactor code`
- `test: add tests`

## Pull Request Process

1. Update documentation if needed
2. Ensure CI passes
3. Request review from maintainers
4. Address review feedback
5. Squash commits if requested

## Questions?

Feel free to open an issue for:
- Bug reports
- Feature requests
- Questions about the codebase

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
