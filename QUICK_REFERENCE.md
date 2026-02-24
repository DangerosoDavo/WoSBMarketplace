# Quick Reference Card

## Setup (One-Time)

```bash
./scripts/init.sh              # Initialize project
# Edit .env with your tokens
go run cmd/bot/main.go         # Test locally
# OR
./scripts/build.sh && ./scripts/deploy.sh  # Deploy Docker
```

## Essential Commands

### Users
```
/submit buy [screenshot]       Submit buy orders
/submit sell [screenshot]      Submit sell orders
/price <item>                  Find best prices
/port <name>                   View port orders
/stats                         Bot statistics
```

### Server Setup (Requires "Manage Server" Permission)
```
/config-set-admin-role role:@RoleName  Set admin role for server
/config-show                           Show server configuration
```

### Admins (Setup - Requires Admin Role)
```
/admin-tag-create <name> <category>    Create tag
/admin-port-add <name> <region>        Create port
```

### Admins (Maintenance)
```
/admin-item-list-untagged             View untagged items
/admin-item-tag <item> <tags>         Tag an item
/admin-tag-list                       View all tags
```

## File Locations

```
.env                           Your credentials (DO NOT COMMIT)
data/database.db               SQLite database
data/images/                   Temp screenshot storage
volumes/data/                  Docker persistent storage
```

## Common Tasks

### First Time Setup
1. Install Claude Code CLI: `npm install -g @anthropic-ai/claude-code`
2. Authenticate Claude: `claude auth login`
3. Run `./scripts/init.sh`
4. Edit `.env` (add Discord token - `ADMIN_ROLE_ID` is optional)
5. Run bot: `go run cmd/bot/main.go`
6. In Discord, create an "Admin" role and assign it to yourself
7. Configure the bot: `/config-set-admin-role role:@Admin`
8. Verify configuration: `/config-show`

### Create Tags (Do This First!)
```
/admin-tag-create weapon type ⚔️
/admin-tag-create ammunition type 💣
/admin-tag-create material type 🔩
/admin-tag-create food type 🍖

/admin-tag-create small size
/admin-tag-create medium size
/admin-tag-create large size
/admin-tag-create heavy size

/admin-tag-create short-range range
/admin-tag-create long-range range
```

### Tag New Items (Daily)
```
/admin-item-list-untagged
/admin-item-tag "Heavy Cannon" weapon,heavy,long-range
/admin-item-tag "Cannonball" ammunition,iron
/admin-item-tag "Rope" material,medium
```

### Query Examples
```
/price cannon                          All prices
/price cannon region:Caribbean         Caribbean only
/price cannon min-price:50 max-price:200   Price range
/port Port Royal                       All orders at port
/ports region:Caribbean                List Caribbean ports
/items tags:weapon,heavy               Heavy weapons
```

## Troubleshooting

### Bot not starting?
```bash
# Check .env has valid tokens
cat .env | grep TOKEN

# Check dependencies
go mod download

# Check logs
docker logs wosb-market-bot
```

### Commands not showing?
- Wait 5-10 minutes (Discord caches)
- Kick and re-invite bot
- Check bot permissions

### OCR failing?
- Verify Claude Code is installed: `claude --version`
- Check Claude authentication: `claude auth status`
- Check billing/credits at console.anthropic.com
- Ensure clear screenshot

## File Structure Quick View

```
wosbTrade/
├── .env                    ← YOUR TOKENS HERE
├── cmd/bot/main.go         ← Entry point
├── internal/
│   ├── bot/                ← Discord logic
│   ├── database/           ← Data layer
│   └── ocr/                ← Claude AI
├── scripts/
│   ├── init.sh             ← Run this first
│   ├── build.sh            ← Build Docker
│   └── deploy.sh           ← Deploy Docker
└── data/                   ← Local storage
```

## Environment Variables

```env
# Required
DISCORD_TOKEN=...

# Optional
ADMIN_ROLE_ID=...  # Global admin role (optional - can configure per-server with /config-set-admin-role)
DATABASE_PATH=/data/database.db
IMAGE_STORAGE_PATH=/data/images
CLAUDE_CODE_PATH=claude  # Path to claude CLI (defaults to 'claude' in PATH)
```

**Note:** Server-specific admin roles (set via `/config-set-admin-role`) take priority over the global `ADMIN_ROLE_ID`.

## Getting Help

1. Check logs first
2. Read COMPLETE_SETUP_GUIDE.md
3. Review error messages
4. Test with `/stats` (simplest command)

## Costs

- Claude API (via Claude Code): ~$0.01-0.02 per screenshot (more cost-effective)
- VPS: ~$5-10/month
- Total: ~$8-12/month for moderate use

## Quick Links

- Setup Guide: [COMPLETE_SETUP_GUIDE.md](COMPLETE_SETUP_GUIDE.md)
- Implementation: [IMPLEMENTATION_COMPLETE.md](IMPLEMENTATION_COMPLETE.md)
- Architecture: [ARCHITECTURE.md](ARCHITECTURE.md)
