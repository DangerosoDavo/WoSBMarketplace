# Project Summary - World of Sea Battle Market Bot

## Overview

This Discord bot tracks market prices for the game "World of Sea Battle" by using AI-powered OCR to extract trading data from player-submitted screenshots.

## Key Features

### Core Functionality
- ✅ Screenshot ingestion via Discord attachments
- ✅ Claude AI (Vision) for OCR processing
- ✅ SQLite database for market data storage
- ✅ Automatic 7-day data expiry
- ✅ Multi-port price tracking
- ✅ Buy/Sell order separation

### Discord Commands
- `/submit [buy|sell] [screenshot]` - Submit market data
- `/price <item>` - Query prices across all ports
- `/port <name>` - View all orders at a port
- `/stats` - View bot statistics
- `/expire` - Manual expiry trigger (admin)
- `/purge <port>` - Remove all port data (admin)

## Technical Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Language** | Go 1.21 | Main application language |
| **Database** | SQLite | Data persistence |
| **AI/OCR** | Claude 3.5 Sonnet | Image analysis |
| **Platform** | Discord Bot | User interface |
| **Container** | Docker (Alpine) | Deployment |
| **Libraries** | discordgo, go-sqlite3 | Core dependencies |

## Architecture

```
User submits screenshot
        ↓
Discord Bot receives image
        ↓
Download to temp storage
        ↓
Send to Claude API for OCR
        ↓
Parse structured JSON response
        ↓
Validate port & order type
        ↓
Atomic database update:
  - Delete old orders for port
  - Insert new orders
  - Set 7-day expiry
  - Log audit trail
        ↓
Delete temp image
        ↓
Send confirmation to Discord
```

## File Structure

```
wosbTrade/
├── cmd/bot/main.go                    # Application entry point
├── internal/
│   ├── bot/
│   │   ├── client.go                  # Bot lifecycle management
│   │   ├── commands.go                # Slash command definitions
│   │   └── handlers.go                # Command handlers
│   ├── database/
│   │   ├── schema.go                  # Database schema
│   │   ├── queries.go                 # SQL operations
│   │   └── schema_test.go             # Database tests
│   └── ocr/
│       └── claude.go                  # Claude API integration
├── scripts/
│   ├── build.sh                       # Docker build script
│   └── deploy.sh                      # Deployment script
├── deployments/
│   └── Dockerfile                     # Alpine-based container
├── .github/workflows/
│   └── build.yml                      # CI/CD pipeline
├── volumes/data/                      # Persistent storage (gitignored)
│   ├── database.db                    # SQLite database
│   └── images/                        # Temp image storage
├── go.mod                             # Go dependencies
├── go.sum                             # Dependency checksums
├── .env                               # Environment config (gitignored)
├── .env.example                       # Environment template
├── Makefile                           # Development commands
├── docker-compose.yml                 # Docker Compose config
├── README.md                          # Main documentation
├── SETUP.md                           # Setup instructions
├── QUICKSTART.md                      # Quick start guide
├── ARCHITECTURE.md                    # Technical deep dive
├── CONTRIBUTING.md                    # Contribution guidelines
├── LICENSE                            # MIT License
└── PROJECT_SUMMARY.md                 # This file
```

## Database Schema

### markets Table
```sql
- id (PK, autoincrement)
- port (indexed) - Port name
- item (indexed) - Item name
- order_type (indexed) - 'buy' or 'sell'
- price - Price per unit
- quantity - Available quantity
- submitted_by - Discord user ID
- submitted_at - Timestamp
- expires_at (indexed) - Auto-set to +7 days
- screenshot_hash - Image hash for tracking
```

### audit_log Table
```sql
- id (PK, autoincrement)
- action - Type of action
- user_id (indexed) - Discord user ID
- timestamp (indexed) - When action occurred
- details - JSON metadata
```

## Key Design Decisions

### Why SQLite?
- Simple deployment (single file)
- No separate database server needed
- Sufficient performance for expected load
- WAL mode for better concurrency
- Easy backup and migration

### Why Claude API?
- Excellent OCR capabilities
- Structured output (JSON)
- Can understand context (buy vs sell)
- Handles various image qualities
- Single API for vision + text

### Why Docker?
- Consistent environment
- Easy deployment
- Volume persistence
- Lightweight (Alpine base)
- Portable across platforms

### Why Go?
- Excellent Discord library (discordgo)
- Built-in concurrency
- Small binary size
- Fast compilation
- Strong typing for reliability

## Data Flow

### Submission Flow (Detailed)
1. User uploads screenshot via `/submit`
2. Bot validates file type is image
3. Downloads to `/data/images/<user_id>_<filename>`
4. Encodes image to base64
5. Sends to Claude API with structured prompt
6. Claude responds with JSON:
   ```json
   {
     "port": "Port Royal",
     "order_type": "buy",
     "items": [
       {"name": "Cannon", "price": 100, "quantity": 10}
     ]
   }
   ```
7. Validates order_type matches user selection
8. Begins database transaction:
   - `DELETE FROM markets WHERE port=? AND order_type=?`
   - `INSERT INTO markets ...` (for each item)
   - `INSERT INTO audit_log ...`
   - `COMMIT`
9. Deletes temporary image file
10. Sends success embed to Discord

### Query Flow (Detailed)
1. User runs `/price cannon`
2. Query: `SELECT * FROM markets WHERE item LIKE '%cannon%' AND expires_at > NOW()`
3. Results sorted by order_type, then price
4. Grouped into buy orders and sell orders
5. Formatted as Discord embed with:
   - Port name
   - Price
   - Quantity
   - Age (how long ago submitted)
6. Limited to top 5 of each type
7. Sent to Discord

### Expiry Flow (Detailed)
1. Background goroutine runs every hour
2. Executes: `DELETE FROM markets WHERE expires_at <= NOW()`
3. If rows deleted > 0:
   - Logs to audit_log
   - Prints count to console
4. Continues until bot shutdown

## Security Considerations

### Secrets Management
- All tokens in `.env` (gitignored)
- Environment variables only
- No hardcoded credentials

### Database Security
- Non-root container user
- Prepared statements (SQL injection prevention)
- Foreign key constraints
- WAL mode (atomic writes)

### Image Handling
- Temporary storage only
- Immediate deletion after processing
- No permanent image storage
- Hash verification

### API Security
- HTTPS only
- API keys in headers
- No sensitive data in URLs
- Rate limiting respected

## Performance Characteristics

### Expected Load
- 100-1000 users
- 10-100 submissions per day
- 100-500 queries per day

### Bottlenecks
1. **Claude API** (~2-5s per image)
   - Mitigated: Deferred responses
2. **SQLite Writes** (concurrent submissions)
   - Mitigated: WAL mode
3. **Discord Rate Limits**
   - Mitigated: Library handles backoff

### Costs
- **Claude API**: ~$0.01-0.03 per screenshot
- **Server**: $5-10/month (VPS)
- **Discord**: Free
- **Total**: ~$10-20/month for moderate usage

## Development Workflow

### Local Development
```bash
# Initial setup
cp .env.example .env
# Edit .env with test credentials

# Run locally
make run-local

# Or with Docker
make restart
make logs
```

### Testing
```bash
make test              # Run all tests
make test-coverage     # Generate coverage report
make fmt               # Format code
make vet               # Run linter
```

### Deployment
```bash
make build    # Build Docker image
make deploy   # Start container
make logs     # View logs
```

## Deployment Options

### Option 1: Shell Scripts
```bash
./scripts/build.sh
./scripts/deploy.sh
```

### Option 2: Makefile
```bash
make restart
```

### Option 3: Docker Compose
```bash
docker-compose up -d
```

## Future Enhancement Ideas

### High Priority
- [ ] Price history tracking
- [ ] Price change alerts
- [ ] Web dashboard for public viewing

### Medium Priority
- [ ] Export data to CSV
- [ ] Price trend charts
- [ ] Top traders leaderboard

### Low Priority
- [ ] Multi-language support
- [ ] Custom expiry periods
- [ ] Bulk data import

### Technical Improvements
- [ ] Redis caching layer
- [ ] PostgreSQL migration (for scale)
- [ ] Horizontal scaling support
- [ ] Prometheus metrics
- [ ] Grafana dashboards

## Known Limitations

### Current Constraints
- SQLite (single-server only)
- No price history
- 7-day fixed expiry
- Manual screenshot submission
- English language only

### Design Trade-offs
- **Simplicity vs Features**: Chose simplicity for v1
- **Cost vs Speed**: Claude API is slower but more accurate than dedicated OCR
- **Storage vs History**: Delete old data to save space
- **Complexity vs Reliability**: Simple architecture is easier to maintain

## Success Metrics

### Technical Metrics
- Uptime: >99%
- OCR accuracy: >95%
- Query response time: <1s
- Submission processing: <10s

### Usage Metrics
- Active users
- Submissions per day
- Queries per day
- Data freshness

## Documentation Map

| Document | Audience | Purpose |
|----------|----------|---------|
| README.md | Everyone | Overview and features |
| QUICKSTART.md | New users | Get started fast |
| SETUP.md | Administrators | Detailed setup |
| ARCHITECTURE.md | Developers | Technical details |
| CONTRIBUTING.md | Contributors | How to contribute |
| PROJECT_SUMMARY.md | Stakeholders | High-level overview |

## Getting Help

### For Users
1. Check QUICKSTART.md
2. Review SETUP.md troubleshooting
3. Open GitHub issue

### For Developers
1. Read ARCHITECTURE.md
2. Check CONTRIBUTING.md
3. Review existing code
4. Ask in GitHub Discussions

## Credits

- **Discord Bot Library**: [discordgo](https://github.com/bwmarrin/discordgo)
- **SQLite Driver**: [go-sqlite3](https://github.com/mattn/go-sqlite3)
- **AI Service**: [Anthropic Claude](https://www.anthropic.com/)
- **Deployment**: Docker

## License

MIT License - See LICENSE file for details

---

**Project Status**: ✅ Ready for deployment

**Last Updated**: 2026-02-22

**Version**: 1.0.0
