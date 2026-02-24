## Complete Setup Guide - Ready to Deploy!

Your World of Sea Battle Market Bot is fully implemented and ready for testing!

## ✅ What's Implemented

### Core Features
- ✅ **Screenshot submission** with intelligent OCR using Claude AI
- ✅ **Port confirmation** - Fuzzy matching with user selection
- ✅ **Item deduplication** - Only confirm each unique item once
- ✅ **Auto-expiry** - Orders expire after 7 days
- ✅ **Tag system** - Categorize items by type, size, range, etc.
- ✅ **Admin management** - Full control over ports, items, and tags
- ✅ **Advanced filtering** - Query by region, price range, tags

### Commands (22 Total)
**User Commands:**
- `/submit [buy|sell] [screenshot]` - Submit market data
- `/price <item> [filters]` - Query prices with filters
- `/port <name>` - View all orders at a port
- `/ports [region]` - List all ports
- `/items [tags]` - Browse items by tags
- `/stats` - Bot statistics

**Admin Commands:** (16 commands for managing ports, items, tags)

## 🚀 Quick Start (5 Steps)

### 1. Get Your Credentials

**Discord Bot Token:**
1. Go to https://discord.com/developers/applications
2. Create New Application → Name it "WOSB Market Bot"
3. Go to "Bot" tab → Click "Add Bot"
4. Enable "Message Content Intent" under Privileged Gateway Intents
5. Click "Reset Token" → Copy the token

**Claude Code CLI Setup:**
1. Install Node.js (v18 or higher) from https://nodejs.org/
2. Install Claude Code globally:
   ```bash
   npm install -g @anthropic-ai/claude-code
   ```
3. Authenticate with Anthropic:
   ```bash
   claude auth login
   ```
   This will open your browser to authenticate with your Anthropic account
4. Verify installation:
   ```bash
   claude --version
   claude auth status
   ```

### 2. Configure Environment

```bash
cd wosbTrade
./scripts/init.sh
```

This will:
- Create `.env` from template
- Create data directories
- Prompt you to add credentials

Edit `.env`:
```env
DISCORD_TOKEN=your_actual_discord_token
ADMIN_ROLE_ID=your_admin_role_id
CLAUDE_CODE_PATH=claude  # Optional, defaults to 'claude' in PATH
```

**To get your Discord Role ID:**
1. Enable Developer Mode (Settings → Advanced → Developer Mode)
2. Go to Server Settings → Roles
3. Create a new role called "WOSB Admin" (or any name you prefer)
4. Right-click the role → Copy ID
5. Assign this role to users who should have admin permissions

### 3. Install Dependencies

```bash
./scripts/init.sh  # Run again after editing .env
```

This downloads all Go dependencies.

### 4. Run the Bot

**Option A: Run Locally (for testing)**
```bash
go run cmd/bot/main.go
```

**Option B: Docker (for production)**

**IMPORTANT**: Make sure Claude Code is authenticated on your host first:
```bash
claude auth login
```

Then deploy:
```bash
./scripts/build.sh   # Build image
./scripts/deploy.sh  # Start container (will copy Claude auth)
```

The deploy script will automatically copy your Claude authentication to the container volume.

### 5. Invite Bot to Server

Use the OAuth2 URL from Discord Developer Portal to invite your bot to your test server.

## 📝 Testing Checklist

### Basic Flow
1. ✅ Bot comes online in Discord
2. ✅ Type `/stats` - Should show empty statistics
3. ✅ Type `/submit buy` - Upload a screenshot
4. ✅ Bot analyzes and asks for port confirmation
5. ✅ Select port or create new one
6. ✅ Bot processes items (auto-matches or asks for confirmation)
7. ✅ Success message shown
8. ✅ Type `/stats` again - Should show 1 submission

### Admin Flow
1. ✅ `/admin-tag-create` - Create tags (weapon, heavy, etc.)
2. ✅ `/admin-item-list-untagged` - See new items
3. ✅ `/admin-item-tag` - Tag an item
4. ✅ `/admin-tag-list` - View all tags
5. ✅ `/price <item>` - Query prices

### Advanced
1. ✅ Submit duplicate items in one screenshot - Only asked once
2. ✅ Submit same port twice - Old orders replaced
3. ✅ Filter by region: `/price cannon region:Caribbean`
4. ✅ Create port aliases for OCR matching
5. ✅ Test expiry: `/admin-expire`

## 🏗️ Architecture Overview

```
User submits screenshot
        ↓
Claude OCR analyzes image
        ↓
Port fuzzy matching
  ├─ Exact match? Auto-confirm
  ├─ Close matches? Show options
  └─ No match? Create new port
        ↓
Item fuzzy matching (per unique item)
  ├─ High confidence (>85%)? Auto-match
  ├─ Medium confidence (60-85%)? Ask user
  └─ Low confidence (<60%)? Treat as new
        ↓
Database commit
  ├─ Delete old orders for (port, order_type)
  ├─ Insert new orders with 7-day expiry
  └─ Mark new items as untagged
        ↓
Success! Admins can tag items later
```

## 📊 Database Schema

```sql
items (id, name, display_name, is_tagged, ...)
item_aliases (item_id, alias) -- OCR variations
tags (id, name, category, icon, color)
item_tags (item_id, tag_id) -- Many-to-many
ports (id, name, display_name, region, ...)
port_aliases (port_id, alias)
markets (port_id, item_id, order_type, price, ...)
```

## 🔧 Configuration Options

### Environment Variables

```env
# Required
DISCORD_TOKEN=                # Your Discord bot token
ADMIN_ROLE_ID=                # Discord Role ID for admin permissions

# Optional
DATABASE_PATH=/data/database.db
IMAGE_STORAGE_PATH=/data/images
LOG_LEVEL=info
CLAUDE_CODE_PATH=claude      # Path to claude CLI (defaults to 'claude')
```

### Admin Setup

1. Create a Discord role for admins (e.g., "WOSB Admin")
2. Copy the role ID (Developer Mode → Right-click role → Copy ID)
3. Add the role ID to `ADMIN_ROLE_ID` in .env
4. Assign this role to users who should have admin access

**Benefits of role-based permissions:**
- Easy to add/remove admins by assigning/removing the role
- No need to edit .env or restart the bot when changing admins
- Visible in Discord who has admin permissions
- Can use Discord's built-in role hierarchy

## 📁 Project Structure

```
wosbTrade/
├── cmd/bot/main.go                     # Entry point
├── internal/
│   ├── bot/
│   │   ├── client.go                   # Bot core
│   │   ├── commands.go                 # 22 command definitions
│   │   ├── submissions.go              # Pending submission manager
│   │   ├── handlers_submit.go          # Submit flow (port)
│   │   ├── handlers_submit_items.go    # Submit flow (items)
│   │   ├── handlers_admin.go           # Admin commands
│   │   └── handlers_queries.go         # User queries
│   ├── database/
│   │   ├── schema.go                   # Database schema
│   │   ├── queries.go                  # SQL operations
│   │   └── matching.go                 # Fuzzy matching
│   └── ocr/
│       └── claude.go                   # Claude API
├── scripts/
│   ├── init.sh                         # Initialization
│   ├── build.sh                        # Docker build
│   └── deploy.sh                       # Docker deploy
├── .env                                # Your config (gitignored)
└── data/                               # Local storage (gitignored)
```

## 🐛 Troubleshooting

### Bot not responding?
```bash
# Check logs
docker logs -f wosb-market-bot

# Or if running locally
# Check terminal output
```

### Database errors?
```bash
# Check database exists
ls -la data/database.db

# Check permissions
chmod 644 data/database.db
```

### OCR failures?
- Verify Claude Code is installed: `claude --version`
- Check authentication: `claude auth status`
- Re-authenticate if needed: `claude auth login`
- Check API credits/billing at console.anthropic.com
- Ensure image is clear and readable
- Try a different screenshot

### Commands not showing?
- Wait 5-10 minutes after first start (Discord caches commands)
- Kick and re-invite the bot
- Check bot has proper permissions

## 💰 Cost Estimates

**Claude API (via Claude Code):**
- ~$0.01-0.02 per screenshot (cost-optimized)
- 100 submissions/month = ~$1-2/month

**Server:**
- Local: Free
- VPS: $5-10/month (DigitalOcean, Linode)

**Total: ~$6-12/month for moderate usage**

## 🎯 Usage Tips

### For Users
1. Take clear, full-screen screenshots
2. Make sure correct tab (Buy/Sell) is visible
3. PNG format works best
4. Submit fresh data (< 7 days old)

### For Admins
1. Tag new items promptly: `/admin-item-list-untagged`
2. Create comprehensive tags: weapon, ammunition, food, material, etc.
3. Add port aliases for OCR variations
4. Monitor stats: `/stats`
5. Create regional tags for ports

### Tag Suggestions

**Type:** weapon, ammunition, material, food, tool, ship-part
**Size:** small, medium, large, huge
**Range:** short-range, medium-range, long-range
**Quality:** common, uncommon, rare, legendary
**Special:** explosive, heavy, fragile, perishable

## 📚 Additional Documentation

- [README.md](README.md) - Full project documentation
- [SETUP.md](SETUP.md) - Detailed setup guide
- [ARCHITECTURE.md](ARCHITECTURE.md) - Technical architecture
- [REFACTORING_PLAN.md](REFACTORING_PLAN.md) - Implementation details
- [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md) - Current status

## ✨ Key Features Explained

### Item Deduplication
If your screenshot shows:
- Cannon (3x)
- Wood (1x)
- Iron (2x)

You'll only be asked to confirm **3 unique items**, not all 6 entries.

### Fuzzy Matching
OCR detects "Port Royale" but database has "Port Royal":
- Bot shows similarity score (91% match)
- You can select or create new
- Aliases prevent future confusion

### Smart Auto-Matching
- **Exact match** (100%) → Auto-confirmed
- **High confidence** (>85%) → Auto-matched
- **Medium** (60-85%) → You choose
- **Low** (<60%) → Treated as new

### Atomic Updates
When you submit new data for a port:
1. Old orders DELETED
2. New orders INSERTED
3. Both happen in a transaction (all-or-nothing)
4. No partial/corrupted data

## 🚀 You're Ready!

Everything is implemented and ready to test. Follow the Quick Start above and let me know if you hit any issues!

Key commands to test first:
1. `/stats` - Verify bot is working
2. `/submit buy [screenshot]` - Full submission flow
3. `/admin-tag-create` - Create some tags
4. `/admin-item-list-untagged` - See untagged items
5. `/price <item>` - Query prices

Happy trading! ⚓
