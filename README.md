# World of Sea Battle Market Bot

A Discord bot that tracks market prices for World of Sea Battle by ingesting player-submitted screenshots using Claude AI for OCR.

## Features

- 📸 Screenshot ingestion via Discord
- 🤖 AI-powered OCR using Claude API
- 💾 SQLite database for market data
- 🔍 Query buy/sell orders by port and item
- ⏰ Auto-expiry of orders after 7 days
- 🐳 Docker containerized deployment

## Architecture

```
┌─────────────┐
│   Discord   │
│    Users    │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│   Discord Bot   │
│   (Golang)      │
└────┬───────┬────┘
     │       │
     │       ▼
     │  ┌─────────────┐
     │  │   Claude    │
     │  │  API (OCR)  │
     │  └─────────────┘
     │
     ▼
┌─────────────┐
│   SQLite    │
│  Database   │
└─────────────┘
```

## Tech Stack

- **Language**: Go
- **Database**: SQLite
- **AI/OCR**: Claude API (Anthropic)
- **Platform**: Discord Bot
- **Deployment**: Docker (Alpine Linux)

## Project Structure

```
wosbTrade/
├── cmd/
│   └── bot/
│       └── main.go              # Entry point
├── internal/
│   ├── bot/
│   │   ├── commands.go          # Discord slash commands
│   │   ├── handlers.go          # Message/interaction handlers
│   │   └── client.go            # Discord client setup
│   ├── database/
│   │   ├── schema.go            # Database schema
│   │   ├── queries.go           # SQL queries
│   │   └── migrations.go        # Database migrations
│   └── ocr/
│       ├── claude.go            # Claude API integration
│       └── parser.go            # Parse OCR results
├── scripts/
│   ├── build.sh                 # Build Docker image
│   └── deploy.sh                # Deploy/restart container
├── deployments/
│   └── Dockerfile               # Alpine-based container
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Database Schema

### Tables

**markets**
- `id` (INTEGER PRIMARY KEY)
- `port` (TEXT) - Port name
- `item` (TEXT) - Item name
- `order_type` (TEXT) - 'buy' or 'sell'
- `price` (INTEGER) - Price per unit
- `quantity` (INTEGER) - Available quantity
- `submitted_by` (TEXT) - Discord user ID
- `submitted_at` (TIMESTAMP)
- `expires_at` (TIMESTAMP) - Auto-calculated as submitted_at + 7 days
- `screenshot_hash` (TEXT) - Hash of original screenshot

**audit_log**
- `id` (INTEGER PRIMARY KEY)
- `action` (TEXT) - Type of action
- `user_id` (TEXT) - Discord user ID
- `timestamp` (TIMESTAMP)
- `details` (TEXT) - JSON details

## Discord Commands

### User Commands

**`/submit [buy|sell]`**
- Attach a screenshot of market orders
- Bot processes image with Claude AI
- Replaces all items for that port/order_type

**`/price <item>`**
- Query best buy/sell prices across all ports
- Shows port, price, quantity, age

**`/port <port_name>`**
- List all active orders at a specific port
- Grouped by buy/sell

**`/stats`**
- Show bot statistics
- Total orders, ports tracked, last update

### Admin Commands

**`/purge <port>`**
- Manually clear all orders for a port

**`/expire`**
- Manually trigger expiry check

## Setup Instructions

### Prerequisites

1. **Discord Bot Token**
   - Go to [Discord Developer Portal](https://discord.com/developers/applications)
   - Create New Application
   - Go to "Bot" section → "Add Bot"
   - Enable "Message Content Intent" under Privileged Gateway Intents
   - Copy the Bot Token

2. **Claude API Key**
   - Sign up at [Anthropic Console](https://console.anthropic.com/)
   - Generate an API key
   - Ensure you have credits/billing set up

3. **Docker** (for deployment)
   - Install Docker on your host system

### Local Development Setup

1. **Clone the repository**
```bash
git clone https://github.com/yourusername/wosbTrade.git
cd wosbTrade
```

2. **Configure environment**
```bash
cp .env.example .env
# Edit .env with your credentials
```

3. **Initialize Go module**
```bash
go mod download
```

4. **Run locally**
```bash
go run cmd/bot/main.go
```

### Docker Deployment

1. **Build the image**
```bash
./scripts/build.sh
```

2. **Deploy the container**
```bash
./scripts/deploy.sh
```

The deployment uses volume mounts for:
- `/data/database.db` - SQLite database (persistent)
- `/data/images` - Temporary image storage
- `/root/.claude` - Claude Code auth (persistent)

### Environment Variables

Create a `.env` file with:

```env
DISCORD_TOKEN=your_discord_bot_token
ANTHROPIC_API_KEY=your_claude_api_key
DATABASE_PATH=/data/database.db
IMAGE_STORAGE_PATH=/data/images
LOG_LEVEL=info
```

## How It Works

1. **User submits screenshot** via `/submit buy` or `/submit sell`
2. **Bot downloads and saves** the image temporarily
3. **Claude API analyzes** the screenshot:
   - Detects port name
   - Identifies buy/sell button state
   - Extracts table data (item, price, quantity)
4. **Database update**:
   - Delete all existing orders for that port + order_type
   - Insert new orders with 7-day expiry
5. **Cleanup**: Delete the processed image
6. **Background task**: Hourly check for expired orders

## Data Expiry

- Orders automatically expire 7 days after submission
- Background goroutine runs every hour to purge expired entries
- Manual `/expire` command available for admins

## Development Workflow

1. Make code changes
2. Run `./scripts/build.sh` to rebuild Docker image
3. Run `./scripts/deploy.sh` to restart container
4. Database and auth persist across restarts

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License

## Support

For issues or questions, please open a GitHub issue.
