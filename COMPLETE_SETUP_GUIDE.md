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
- ✅ **Player trading** - Create buy/sell orders, search, and contact traders
- ✅ **Cross-server DM relay** - Bot relays messages between traders using in-game names
- ✅ **Trade moderation** - Ban/unban traders, user reports, admin review workflow

### Commands (35 Total)
**User Commands (6):**
- `/submit [buy|sell] [screenshot]` - Submit market data
- `/price <item> [filters]` - Query prices with filters
- `/port <name>` - View all orders at a port
- `/ports [region]` - List all ports
- `/items [tags]` - Browse items by tags
- `/stats` - Bot statistics

**Player Trading Commands (8):**
- `/trade-set-name <name>` - Set your in-game name for trading
- `/trade-create <type> <item> <price> <quantity> <duration> [port] [notes]` - Create a buy or sell order
- `/trade-search [item] [type] [port] [min-price] [max-price]` - Search player trade orders
- `/trade-my-orders` - View your active trade orders
- `/trade-cancel <order-id>` - Cancel one of your trade orders
- `/trade-contact <order-id>` - Start a DM conversation with the order creator
- `/trade-end` - End your active trade conversation
- `/trade-report <order-id> <reason>` - Report a trader for misconduct

**Admin Commands:** (16 commands for managing ports, items, tags)

**Admin Trade Moderation Commands (5):**
- `/admin-trade-ban <user> <reason> [duration]` - Ban a user from trading (temp or permanent)
- `/admin-trade-unban <user>` - Remove a trade ban
- `/admin-trade-bans` - List all active trade bans
- `/admin-trade-reports [status]` - View trade reports (pending/reviewed/dismissed)
- `/admin-trade-report-action <report-id> <action> [reason]` - Dismiss or ban from a report

## 🚀 Quick Start (5 Steps)

### 1. Get Your Credentials

**Discord Bot Token:**
1. Go to https://discord.com/developers/applications
2. Create New Application → Name it "WOSB Market Bot"
3. Go to "Bot" tab → Click "Add Bot"
4. Under **Privileged Gateway Intents**, enable:
   - **Message Content Intent** - Required for OCR processing and DM trade relay
5. Click "Reset Token" → Copy the token

**Bot Invite URL (OAuth2 tab → URL Generator):**
1. Select scopes: `bot`, `applications.commands`
2. Select bot permissions:
   - `Send Messages` - Respond to commands and relay trade DMs
   - `Embed Links` - Rich embeds for orders and search results
   - `Add Reactions` - Checkmark reactions to confirm DM message delivery
   - `Use Slash Commands` - All bot commands
3. Copy the generated URL and use it to invite the bot to your server

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

### Player Trading Flow
1. ✅ `/trade-set-name name:TestPlayer` - Sets in-game name
2. ✅ `/trade-create type:sell item:cannon price:5000 quantity:3 duration:7d` - Creates order
3. ✅ `/trade-search item:cannon` - Shows order with Contact button
4. ✅ Second user: `/trade-contact order-id:1` - Starts conversation, both get DMs
5. ✅ Both users DM the bot - Messages relay with in-game name + checkmark reaction
6. ✅ `/trade-end` - Closes conversation, other party notified via DM
7. ✅ `/trade-my-orders` - Shows active orders
8. ✅ `/trade-cancel order-id:1` - Cancels order
9. ✅ Conversation auto-closes after 30 min inactivity with DM notification

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
-- Market Data (OCR)
items (id, name, display_name, is_tagged, ...)
item_aliases (item_id, alias) -- OCR variations
tags (id, name, category, icon, color)
item_tags (item_id, tag_id) -- Many-to-many
ports (id, name, display_name, region, ...)
port_aliases (port_id, alias)
markets (port_id, item_id, order_type, price, ...)

-- Player Trading
player_profiles (user_id, ingame_name, ...)
player_orders (id, user_id, item_id, order_type, price, quantity, port_id, notes, status, expires_at, ...)
trade_conversations (id, order_id, initiator_user_id, creator_user_id, status, last_message_at, ...)

-- Trade Moderation
trade_bans (id, user_id, reason, banned_by, banned_at, expires_at, active)
trade_reports (id, reporter_user_id, reported_user_id, order_id, reason, status, reviewed_by, ...)

-- Configuration
guild_settings (guild_id, admin_role_id, ...)
audit_log (id, action, user_id, timestamp, details)
```

## 🤝 Player Trading & DM Relay

### How It Works

Players create buy/sell orders visible across all servers the bot is in. When someone wants to trade, the bot acts as a DM relay so traders never need to join each other's servers.

```
Player A (Server 1)                Bot                Player B (Server 2)
     │                              │                        │
     ├── /trade-set-name ──────────>│                        │
     ├── /trade-create sell ───────>│                        │
     │                              │<──── /trade-search ────┤
     │                              │<──── /trade-contact ───┤
     │<──── DM: "B wants to trade" ─┤──── DM: "Connected" ──>│
     │                              │                        │
     ├── DM: "I have 3 cannons" ───>│── "[A]: I have 3..." ─>│
     │<── "[B]: What price?" ───────┤<── DM: "What price?" ──┤
     │                              │                        │
     ├── /trade-end ───────────────>│── DM: "A ended chat" ─>│
```

### Requirements for DM Relay
- **Both users** must have set their in-game name via `/trade-set-name`
- **DMs must be open**: Users need "Allow direct messages from server members" enabled in Discord privacy settings for at least one shared server with the bot
- **One conversation at a time**: Each user can only have one active trade conversation
- **30-minute timeout**: Conversations auto-close after 30 minutes of inactivity (both parties are notified)
- **Message delivery**: The bot adds a checkmark reaction to each message to confirm delivery

### Order Expiry
Players choose how long their orders stay active: 1 day, 3 days, 7 days, or 14 days. Expired orders are automatically cleaned up hourly.

### Background Processes
- **Player order expiry** - Runs hourly, cancels expired player orders
- **Conversation timeout** - Runs every 5 minutes, closes stale conversations and notifies both parties
- **Conversation recovery** - On bot restart, active conversations are loaded from the database back into memory

## 🛡️ Trade Moderation

### Banning Traders
Admins can ban users from the trading system. Banned users cannot create orders or contact other traders.

- **Temporary bans**: Choose a duration (1d, 3d, 7d, 14d, 30d) — ban auto-expires
- **Permanent bans**: Omit the duration or select "Permanent"
- **On ban**: All active orders from the banned user are automatically cancelled
- **Unban**: Restores trading privileges immediately

### User Reports
Any user can report a trader by referencing an active order. Reports include:
- The reported user (auto-detected from the order)
- A reason (5-500 characters)
- Reports are submitted anonymously (only admins see reporter identity)

### Admin Report Workflow
1. `/admin-trade-reports` — View pending reports
2. Review the report details (reporter, reported user, order, reason)
3. `/admin-trade-report-action report-id:X action:ban` — Ban the reported user (permanent, cancels their orders)
4. `/admin-trade-report-action report-id:X action:dismiss` — Dismiss the report
5. Optionally provide a custom `reason` when banning

All moderation actions are logged to the audit log.

## 🔧 Configuration Options

### Environment Variables

```env
# Required
DISCORD_TOKEN=                # Your Discord bot token

# Optional
ADMIN_ROLE_ID=                # Global admin role (can configure per-server with /config-set-admin-role)
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
│   │   ├── client.go                   # Bot core, background goroutines
│   │   ├── commands.go                 # 35 command definitions
│   │   ├── handlers.go                 # Command routing + helpers
│   │   ├── submissions.go              # Pending submission manager
│   │   ├── trade_conversations.go      # In-memory trade conversation manager
│   │   ├── handlers_submit.go          # Submit flow (port)
│   │   ├── handlers_submit_items.go    # Submit flow (items)
│   │   ├── handlers_admin.go           # Admin commands
│   │   ├── handlers_queries.go         # User queries (price, port, items, stats)
│   │   ├── handlers_config.go          # Server configuration commands
│   │   ├── handlers_trading.go         # Player trading commands
│   │   ├── handlers_dm_relay.go        # DM message relay for trades
│   │   └── handlers_moderation.go     # Trade ban/report moderation
│   ├── database/
│   │   ├── schema.go                   # Database schema + models
│   │   ├── queries.go                  # Market/admin SQL operations
│   │   ├── queries_trading.go          # Trading SQL operations
│   │   ├── queries_moderation.go      # Ban/report SQL operations
│   │   └── matching.go                 # Fuzzy matching
│   └── ocr/
│       └── claude.go                   # Claude Code CLI integration
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
6. `/trade-set-name name:YourName` - Set up trading profile
7. `/trade-create type:sell item:cannon price:5000 quantity:3 duration:7d` - Create a trade order
8. `/trade-search item:cannon` - Search and contact traders

Happy trading! ⚓
