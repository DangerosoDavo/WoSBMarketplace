# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Price history tracking
- Price change alerts via DM
- Web dashboard for public viewing
- Export data to CSV functionality

## [1.0.0] - 2026-02-22

### Added
- Initial release of World of Sea Battle Market Bot
- Discord bot with slash command interface
- Claude AI integration for screenshot OCR
- SQLite database for market data storage
- Automatic 7-day order expiry
- `/submit` command for screenshot submission
- `/price` command for item price queries
- `/port` command for port-specific order viewing
- `/stats` command for bot statistics
- `/expire` admin command for manual expiry
- `/purge` admin command for port data removal
- Docker containerization with Alpine Linux
- Volume-based persistence for database
- Automated build and deployment scripts
- Comprehensive documentation (README, SETUP, QUICKSTART, ARCHITECTURE)
- Unit tests for database layer
- GitHub Actions CI/CD pipeline
- Docker Compose configuration
- Makefile for development convenience

### Security
- Environment-based secrets management
- Non-root container user
- SQL injection prevention via prepared statements
- Temporary image storage with immediate cleanup

---

## Release Notes

### Version 1.0.0 - Initial Release

This is the first production-ready release of the World of Sea Battle Market Bot.

**Key Features:**
- AI-powered screenshot analysis
- Real-time market tracking
- Multi-port price comparison
- Automatic data expiry
- Admin controls

**Technical Highlights:**
- Go 1.21 with discordgo
- SQLite with WAL mode
- Claude 3.5 Sonnet for OCR
- Docker containerization
- Comprehensive test suite

**Getting Started:**
See [QUICKSTART.md](QUICKSTART.md) for a 5-minute setup guide.

**Requirements:**
- Discord bot token
- Anthropic API key
- Docker installed

**Known Issues:**
- None at this time

**Breaking Changes:**
- N/A (initial release)

---

## Version History

| Version | Date | Notes |
|---------|------|-------|
| 1.0.0 | 2026-02-22 | Initial release |

---

## Migration Guides

### Upgrading to 1.0.0
This is the initial release, no migration needed.

---

## Contributors

Thanks to all contributors who helped build this project!

- Initial development and architecture

---

## Links

- [GitHub Repository](https://github.com/yourusername/wosbTrade)
- [Issue Tracker](https://github.com/yourusername/wosbTrade/issues)
- [Documentation](https://github.com/yourusername/wosbTrade#readme)
