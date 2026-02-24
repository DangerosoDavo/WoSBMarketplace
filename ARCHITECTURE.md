# Architecture Documentation

## System Overview

The World of Sea Battle Market Bot is a Discord-based market tracking system that uses AI-powered OCR to extract trading data from player-submitted screenshots.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Discord                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │  User 1  │  │  User 2  │  │  User N  │                  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                  │
│       │             │              │                         │
│       └─────────────┼──────────────┘                         │
│                     │                                        │
└─────────────────────┼────────────────────────────────────────┘
                      │ Slash Commands & Attachments
                      ▼
         ┌────────────────────────────┐
         │    Discord Bot (Golang)    │
         │  ┌──────────────────────┐  │
         │  │  Command Handlers    │  │
         │  ├──────────────────────┤  │
         │  │  Image Downloader    │  │
         │  ├──────────────────────┤  │
         │  │  Database Layer      │  │
         │  ├──────────────────────┤  │
         │  │  Expiry Scheduler    │  │
         │  └──────────────────────┘  │
         └─────┬────────────────┬─────┘
               │                │
               ▼                ▼
     ┌─────────────────┐  ┌──────────────┐
     │  Claude API     │  │   SQLite     │
     │  (OCR/Vision)   │  │  Database    │
     └─────────────────┘  └──────────────┘
```

## Component Details

### 1. Discord Bot Layer (`internal/bot/`)

**Responsibilities:**
- Manage Discord WebSocket connection
- Register and handle slash commands
- Download and temporarily store images
- Coordinate between OCR service and database
- Schedule and run periodic expiry checks

**Key Files:**
- `client.go` - Bot initialization, lifecycle management
- `commands.go` - Slash command definitions
- `handlers.go` - Command execution logic

### 2. OCR Layer (`internal/ocr/`)

**Responsibilities:**
- Interface with Claude API
- Send images for vision analysis
- Parse structured data from AI responses
- Validate extracted data

**Key Files:**
- `claude.go` - Claude API client and data structures

**API Flow:**
```
1. Image received (PNG/JPEG/WebP)
2. Encode to base64
3. Send to Claude API with structured prompt
4. Parse JSON response
5. Validate data (port, order_type, items)
6. Return structured MarketData
```

### 3. Database Layer (`internal/database/`)

**Responsibilities:**
- SQLite connection management
- Schema management and migrations
- CRUD operations for market data
- Audit logging
- Statistics aggregation

**Key Files:**
- `schema.go` - Database schema and connection
- `queries.go` - SQL operations

**Database Schema:**

```sql
markets
├── id (PK)
├── port (indexed)
├── item (indexed)
├── order_type (indexed: 'buy'|'sell')
├── price
├── quantity
├── submitted_by
├── submitted_at
├── expires_at (indexed)
└── screenshot_hash

audit_log
├── id (PK)
├── action
├── user_id (indexed)
├── timestamp (indexed)
└── details (JSON)
```

## Data Flow

### Screenshot Submission Flow

```
1. User runs /submit buy [screenshot.png]
   │
   ▼
2. Bot validates attachment is an image
   │
   ▼
3. Download image to temp storage
   │
   ▼
4. Send to Claude API for analysis
   │
   ├─► Claude returns JSON:
   │   {
   │     "port": "Port Royal",
   │     "order_type": "buy",
   │     "items": [...]
   │   }
   │
   ▼
5. Validate order_type matches user selection
   │
   ▼
6. Begin database transaction:
   ├─► DELETE all orders for (port, order_type)
   ├─► INSERT new orders with 7-day expiry
   └─► INSERT audit log entry
   │
   ▼
7. Delete temp image file
   │
   ▼
8. Send success embed to Discord
```

### Price Query Flow

```
1. User runs /price cannon
   │
   ▼
2. Query database:
   SELECT * FROM markets
   WHERE item LIKE '%cannon%'
   AND expires_at > NOW()
   ORDER BY order_type, price
   │
   ▼
3. Group results by buy/sell
   │
   ▼
4. Format as Discord embed:
   ├─► Buy Orders (lowest first)
   └─► Sell Orders (lowest first)
   │
   ▼
5. Send embed to Discord
```

### Expiry Flow

```
┌─────────────────────┐
│  Hourly Timer       │
│  (go routine)       │
└──────────┬──────────┘
           │
           ▼
  DELETE FROM markets
  WHERE expires_at <= NOW()
           │
           ▼
   INSERT audit_log
   (expired count)
```

## Deployment Architecture

### Docker Container

```
Alpine Linux Container
├── /app/wosbTrade (binary)
└── /data/ (mounted volume)
    ├── database.db (SQLite)
    ├── database.db-wal (Write-Ahead Log)
    ├── database.db-shm (Shared Memory)
    └── images/ (temp storage)
```

### Build Process

**Multi-stage Docker build:**

1. **Build Stage** (golang:1.21-alpine)
   - Install build dependencies (gcc, musl-dev, sqlite-dev)
   - Download Go dependencies
   - Compile binary with CGO enabled (for SQLite)

2. **Runtime Stage** (alpine:latest)
   - Minimal runtime dependencies
   - Non-root user (botuser)
   - Binary only, no source code

### Volume Persistence

```
Host Machine                 Container
─────────────                ─────────
./volumes/data/     ────────────► /data/
  ├── database.db   ────────────► /data/database.db
  └── images/       ────────────► /data/images/
```

## Security Considerations

### 1. Secrets Management
- Environment variables (never in code)
- `.env` file excluded from git
- No hardcoded tokens

### 2. Database Security
- Non-root container user
- SQLite WAL mode (better concurrency)
- Foreign key constraints enabled
- Prepared statements (SQL injection prevention)

### 3. Image Handling
- Temporary storage only
- Immediate deletion after processing
- Size limits enforced by Discord
- Hash verification

### 4. API Security
- HTTPS only (Claude API)
- API key in headers, not URLs
- Rate limiting handled by SDK

## Performance Characteristics

### Bottlenecks

1. **Claude API calls** (~2-5s per image)
   - Mitigated: Deferred Discord responses
   - User sees "processing..." immediately

2. **SQLite writes** (concurrent submissions)
   - Mitigated: WAL mode enabled
   - Transactions for atomic updates

3. **Discord API rate limits**
   - Handled by discordgo library
   - Exponential backoff built-in

### Scalability

**Current Limits:**
- SQLite: ~100K concurrent reads, ~1K concurrent writes
- Single container: Handles ~100 simultaneous users
- Claude API: Rate limited by Anthropic tier

**Scaling Options:**
1. Horizontal: Multiple bot instances (different Discord tokens)
2. Vertical: More CPU for concurrent OCR processing
3. Database: Migrate to PostgreSQL for higher concurrency

## Error Handling

### Error Levels

1. **User Errors** (shown in Discord)
   - Invalid image format
   - Order type mismatch
   - No data detected

2. **System Errors** (logged only)
   - Database connection failures
   - API timeouts
   - Disk space issues

3. **Recoverable Errors**
   - Temporary API failures → retry
   - Discord disconnects → auto-reconnect
   - Transient DB locks → retry transaction

### Logging Strategy

```
INFO  - Normal operations (command usage, submissions)
WARN  - Recoverable errors (retries, degraded service)
ERROR - Critical failures (startup issues, persistent errors)
```

## Future Enhancements

### Potential Features

1. **Price History**
   - Don't delete old orders, mark as expired
   - Chart price trends over time
   - Predict market movements

2. **Price Alerts**
   - Users subscribe to items
   - DM when price threshold met
   - Webhook notifications

3. **Multi-Server Support**
   - Separate databases per Discord server
   - Server-specific configurations
   - Cross-server price comparison

4. **Web Dashboard**
   - Public market browser
   - REST API for external tools
   - Real-time WebSocket updates

5. **OCR Improvements**
   - Cache common item names
   - Fuzzy matching for typos
   - Multi-language support

6. **Analytics**
   - Most traded items
   - Busiest ports
   - User contribution leaderboard

## Monitoring & Observability

### Key Metrics to Track

1. **Operational**
   - Uptime/Downtime
   - Command response time
   - OCR success rate

2. **Usage**
   - Submissions per day
   - Unique users
   - Most queried items

3. **Costs**
   - Claude API tokens used
   - Cost per submission
   - Monthly totals

4. **Database**
   - Total orders stored
   - Database file size
   - Query performance

### Health Checks

```yaml
Docker Health Check:
  Command: pgrep -f wosbTrade
  Interval: 30s
  Timeout: 10s
  Retries: 3
```

## Development Guidelines

### Code Organization

```
cmd/          - Application entry points
internal/     - Private application code
  bot/        - Discord bot logic
  database/   - Data layer
  ocr/        - External service integration
scripts/      - Build and deployment scripts
deployments/  - Docker and infrastructure
```

### Adding New Commands

1. Add command definition to `commands.go`
2. Add handler function to `handlers.go`
3. Add database queries if needed to `queries.go`
4. Update README.md with command documentation
5. Test locally before deploying

### Database Changes

1. Update schema in `schema.go`
2. Add migration logic if needed
3. Test with existing data
4. Document in commit message
5. Consider backward compatibility

## Troubleshooting Guide

### Common Issues

**Bot not responding:**
- Check Docker container is running
- Verify Discord token is valid
- Check bot has channel permissions
- Review logs for errors

**OCR failures:**
- Verify Anthropic API key
- Check API quota/credits
- Ensure image quality is good
- Review Claude API status

**Database errors:**
- Check file permissions on volume
- Verify disk space available
- Check for database corruption
- Review SQLite logs

**Memory issues:**
- Monitor container memory usage
- Check for image cleanup
- Review goroutine leaks
- Profile with pprof if needed
