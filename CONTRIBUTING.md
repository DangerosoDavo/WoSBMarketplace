# Contributing to World of Sea Battle Market Bot

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Git
- A Discord bot token (for testing)
- An Anthropic API key (for testing OCR features)

### Development Setup

1. Fork the repository on GitHub

2. Clone your fork:
```bash
git clone https://github.com/YOUR_USERNAME/wosbTrade.git
cd wosbTrade
```

3. Add upstream remote:
```bash
git remote add upstream https://github.com/ORIGINAL_OWNER/wosbTrade.git
```

4. Create a development branch:
```bash
git checkout -b feature/your-feature-name
```

5. Set up your environment:
```bash
cp .env.example .env
# Edit .env with your test credentials
```

6. Install dependencies:
```bash
make deps
```

## Development Workflow

### Making Changes

1. **Create a feature branch:**
```bash
git checkout -b feature/add-new-command
```

2. **Make your changes:**
   - Write clean, documented code
   - Follow Go conventions
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes:**
```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Format code
make fmt

# Run linter
make vet
```

4. **Test with Docker:**
```bash
make restart
make logs
```

5. **Commit your changes:**
```bash
git add .
git commit -m "feat: add new price alert command"
```

### Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

Examples:
```
feat: add price history tracking
fix: resolve database deadlock on concurrent submissions
docs: update setup instructions for Windows
refactor: simplify Claude API error handling
test: add integration tests for /port command
```

### Pull Request Process

1. **Update from upstream:**
```bash
git fetch upstream
git rebase upstream/main
```

2. **Push to your fork:**
```bash
git push origin feature/your-feature-name
```

3. **Create Pull Request:**
   - Go to GitHub and create a PR from your fork
   - Fill out the PR template
   - Link any related issues
   - Request review

4. **Address feedback:**
   - Make requested changes
   - Push new commits to the same branch
   - PR will automatically update

5. **Merge:**
   - Maintainer will merge once approved
   - Delete your feature branch after merge

## Code Guidelines

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Run `go vet` before committing
- Keep functions small and focused
- Document exported functions and types

### Project Structure

```
cmd/          - Application entry points
internal/     - Private application code
  bot/        - Discord bot logic
  database/   - Data layer
  ocr/        - External service integration
scripts/      - Build and deployment
deployments/  - Docker and infrastructure
```

### Adding New Commands

1. **Define the command** in `internal/bot/commands.go`:
```go
{
    Name:        "newcommand",
    Description: "Description of what it does",
    Options: []*discordgo.ApplicationCommandOption{
        // Define options here
    },
}
```

2. **Add handler** in `internal/bot/handlers.go`:
```go
func (b *Bot) handleNewCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Implementation
}
```

3. **Add database queries** if needed in `internal/database/queries.go`

4. **Write tests** in corresponding `_test.go` files

5. **Update documentation** in README.md

### Adding Database Changes

1. **Update schema** in `internal/database/schema.go`
2. **Add migration logic** if modifying existing tables
3. **Update queries** in `internal/database/queries.go`
4. **Write tests** in `internal/database/schema_test.go`
5. **Document changes** in commit message

### Testing

- Write tests for new functionality
- Maintain or improve test coverage
- Test edge cases and error conditions
- Use table-driven tests where appropriate

Example test:
```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFeature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Documentation

### Code Documentation

- Document all exported functions, types, and constants
- Use GoDoc format
- Include examples where helpful

```go
// AnalyzeScreenshot processes a game screenshot and extracts market data.
// It sends the image to Claude API and parses the structured response.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - imagePath: Path to the screenshot file
//
// Returns:
//   - *MarketData: Parsed market information
//   - error: Any error encountered during processing
func (c *ClaudeClient) AnalyzeScreenshot(ctx context.Context, imagePath string) (*MarketData, error) {
    // Implementation
}
```

### User Documentation

Update these files as needed:
- `README.md` - Main documentation
- `SETUP.md` - Setup instructions
- `QUICKSTART.md` - Quick start guide
- `ARCHITECTURE.md` - Technical details

## Issue Reporting

### Bug Reports

Include:
- Clear description of the issue
- Steps to reproduce
- Expected behavior
- Actual behavior
- Environment details (OS, Docker version, etc.)
- Logs and error messages

### Feature Requests

Include:
- Clear description of the feature
- Use case and motivation
- Proposed implementation (if you have ideas)
- Alternative approaches considered

## Code Review

### What Reviewers Look For

- **Correctness:** Does it work as intended?
- **Tests:** Are there adequate tests?
- **Documentation:** Is it well documented?
- **Style:** Does it follow project conventions?
- **Performance:** Are there any bottlenecks?
- **Security:** Are there any vulnerabilities?

### Responding to Reviews

- Be receptive to feedback
- Ask questions if something is unclear
- Make requested changes promptly
- Explain your reasoning when disagreeing

## Community

### Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume good intentions

### Getting Help

- Check existing issues and PRs
- Read documentation thoroughly
- Ask in GitHub Discussions
- Join our Discord server (if applicable)

## Development Tips

### Useful Make Commands

```bash
make help           # Show all commands
make test           # Run tests
make restart        # Rebuild and restart
make logs           # View logs
make db-shell       # Access database
make fmt            # Format code
```

### Debugging

**View logs:**
```bash
make logs
```

**Access container:**
```bash
make shell
```

**Check database:**
```bash
make db-stats
```

**Test locally without Docker:**
```bash
make run-local
```

### Common Issues

**Tests failing locally:**
- Ensure dependencies are updated: `make deps`
- Check Go version: `go version`
- Clear test cache: `go clean -testcache`

**Docker build failing:**
- Check Docker is running
- Clear Docker cache: `docker system prune`
- Rebuild from scratch: `make clean && make build`

## Release Process

(For maintainers)

1. Update version in code
2. Update CHANGELOG.md
3. Create release branch
4. Run full test suite
5. Create GitHub release
6. Tag version
7. Deploy to production

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Questions?

Feel free to:
- Open an issue for questions
- Start a discussion
- Reach out to maintainers

Thank you for contributing to make this project better!
