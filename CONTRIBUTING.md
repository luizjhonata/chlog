# Contributing to chlog

## Development Setup

1. Clone the repository
2. Ensure Go 1.26+ is installed
3. Run `make build` to verify the setup

## Making Changes

1. Create a branch from `main`
2. Make your changes
3. Write tests following the conventions in CLAUDE.md
4. Run `make lint` and `make test`
5. Open a pull request

## Commit Messages

Follow conventional commits: `type(scope): description`

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

## Code Style

- Follow the patterns in existing code
- Keep functions small and focused
- Use early returns to reduce nesting
- Names should be self-explanatory
