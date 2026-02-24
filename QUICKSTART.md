# Quick Start Guide

Get the World of Sea Battle Market Bot running in under 10 minutes.

## Prerequisites Checklist

- [ ] Docker installed
- [ ] Node.js installed (v18+) for Claude Code CLI
- [ ] Discord bot token (see [COMPLETE_SETUP_GUIDE.md](COMPLETE_SETUP_GUIDE.md) for details)
- [ ] Anthropic account (free tier works)
- [ ] Bot invited to your Discord server with required permissions and intents (see below)

## 5-Minute Setup

### 1. Discord Bot Setup (Developer Portal)

Before cloning, ensure your bot has the correct intents and permissions in the [Discord Developer Portal](https://discord.com/developers/applications):

**Privileged Gateway Intents** (Bot tab → Privileged Gateway Intents):
- [x] **Message Content Intent** - Required for OCR screenshot processing and DM trade relay

**Bot Permissions** (OAuth2 → URL Generator):
When generating your bot invite URL, select these permissions:
- `Send Messages` - Respond to commands and relay trade DMs
- `Embed Links` - Rich embeds for search results and order confirmations
- `Add Reactions` - Checkmark reactions to confirm DM delivery
- `Use Slash Commands` - All bot commands
- `Send Messages in Threads` (optional)

**OAuth2 Scopes**: `bot`, `applications.commands`

**Note on DM Relay**: The bot sends and receives Direct Messages to facilitate player-to-player trading. Users must have "Allow direct messages from server members" enabled in their Discord privacy settings for a server the bot shares with them.

### 2. Clone the Repository

```bash
git clone https://github.com/yourusername/wosbTrade.git
cd wosbTrade
```

### 3. Install and Authenticate Claude Code

```bash
# Install Claude Code CLI globally
npm install -g @anthropic-ai/claude-code

# Authenticate with Anthropic (opens browser)
claude auth login

# Verify installation
claude --version
claude auth status
```

### 4. Configure Environment

```bash
cp .env.example .env
nano .env  # Edit with your tokens
```

**Minimum required value:**
```env
DISCORD_TOKEN=your_bot_token_here
```

**Optional global admin role (applies to all servers):**
```env
ADMIN_ROLE_ID=your_global_admin_role_id
```

You can configure admin roles per-server using Discord commands (recommended) - see step 8 below!

### 5. Build and Deploy

```bash
./scripts/build.sh   # Builds Docker image (~2 min)
./scripts/deploy.sh  # Starts the bot (copies Claude auth to container)
```

The deploy script will automatically copy your Claude authentication to the container.

### 6. Verify It's Running

```bash
docker logs -f wosb-market-bot
```

Look for:
```
✓ Logged in as: WOSB Market Bot#1234
✓ Registered command: submit
✓ Bot is now running
```

### 7. Test in Discord

Type `/stats` in your Discord server. If you see a response, you're done!

### 8. Configure Server Admin Role (In Discord)

**Option 1: Use Discord command (Recommended)**
```
1. Create a role in your server (e.g., "WOSB Admin")
2. Assign the role to yourself
3. In Discord, type: /config-set-admin-role role:@WOSB Admin
4. Verify with: /config-show
```

This command requires "Manage Server" permission and is much easier than editing .env files!

**Option 2: Use global config (applies to all servers)**
- Set `ADMIN_ROLE_ID` in .env as shown in step 3
- Restart the bot

**Priority:** Server-specific settings (Option 1) take priority over global config (Option 2)

## Using the Bot

### Submit Market Data

1. Take a screenshot of the market interface in World of Sea Battle
2. In Discord, type `/submit`
3. Select `buy` or `sell` depending on which tab is showing
4. Attach your screenshot
5. Submit and wait ~5 seconds for processing

### Query Prices

**Find item prices:**
```
/price cannon
/price wood
/price iron
```

**View all orders at a port:**
```
/port Port Royal
/port Tortuga
```

**View statistics:**
```
/stats
```

### Player Trading (Cross-Server)

The bot includes a full player-to-player trading system with DM relay, allowing players across different servers to trade without joining each other's Discord servers.

**Set up your trader profile:**
```
/trade-set-name name:CaptainHook
```

**Create a trade order:**
```
/trade-create type:sell item:Heavy Cannon price:5000 quantity:3 duration:7d
/trade-create type:buy item:Iron price:100 quantity:50 duration:3d port:Port Royal
```

**Browse and search orders:**
```
/trade-search item:cannon
/trade-search type:sell min-price:100 max-price:1000
/trade-my-orders
```

**Contact a trader:**
```
/trade-contact order-id:42
```
This opens a DM relay through the bot. Both traders chat via DMs with the bot forwarding messages using in-game names. Conversations auto-close after 30 minutes of inactivity.

**End a conversation:**
```
/trade-end
```

**Cancel your order:**
```
/trade-cancel order-id:42
```

**Report a trader:**
```
/trade-report order-id:42 reason:"Posting fake prices"
```

### Admin Commands (Requires Admin Role)

**Create tags:**
```
/admin-tag-create weapon type
/admin-tag-create heavy size
```

**Tag items:**
```
/admin-item-list-untagged
/admin-item-tag "Heavy Cannon" weapon,heavy
```

**Trade moderation:**
```
/admin-trade-ban user:@someone reason:"scamming" duration:7d
/admin-trade-unban user:@someone
/admin-trade-bans
/admin-trade-reports
/admin-trade-report-action report-id:1 action:ban
/admin-trade-report-action report-id:2 action:dismiss
```

## Troubleshooting

### Bot not appearing in slash commands?

Wait 5-10 minutes after first start, Discord caches commands.

### "Failed to analyze screenshot" error?

```bash
# Enter the container
docker exec -it wosb-market-bot sh

# Check Claude Code is working
claude --version
claude auth status

# If auth expired, re-authenticate on host and redeploy
# Exit container first
exit

# On host machine
claude auth login
./scripts/deploy.sh  # Copies new auth to container
```

### Claude authentication issues?

**Option 1: Re-copy authentication**
```bash
# Make sure you're authenticated on host
claude auth status

# Redeploy to copy fresh auth
./scripts/deploy.sh
```

**Option 2: Manually authenticate in container**
```bash
# Enter container
docker exec -it wosb-market-bot sh

# Authenticate inside container (requires interactive terminal)
claude auth login
```

**Option 3: Check logs for details**
```bash
docker logs wosb-market-bot | grep -i claude
docker logs wosb-market-bot | grep -i error
```

### Bot offline?

```bash
docker restart wosb-market-bot
docker logs wosb-market-bot
```

## What's Next?

- Read [COMPLETE_SETUP_GUIDE.md](COMPLETE_SETUP_GUIDE.md) for detailed configuration
- See [ARCHITECTURE.md](ARCHITECTURE.md) for how it works
- Check [IMPLEMENTATION_COMPLETE.md](IMPLEMENTATION_COMPLETE.md) for features
- Review [QUICK_REFERENCE.md](QUICK_REFERENCE.md) for command reference

## Common Commands

```bash
# View logs
docker logs -f wosb-market-bot

# Restart bot
docker restart wosb-market-bot

# Rebuild after code changes
./scripts/build.sh && ./scripts/deploy.sh

# Stop bot
docker stop wosb-market-bot

# Access container shell
docker exec -it wosb-market-bot sh

# Access database
docker exec -it wosb-market-bot sh
cd /data && sqlite3 database.db
```

## Screenshot Tips

For best OCR results:
1. Take full-screen or high-resolution screenshots
2. Ensure text is clear and readable
3. Make sure the correct tab (Buy/Sell) is selected
4. Capture the entire market table
5. PNG format recommended over JPEG

## Cost Estimates

- **Claude API**: ~$0.01-0.02 per screenshot (via Claude Code)
- **Server**: Free (local) or ~$5-10/month (VPS)
- **Total**: ~$1-2/month for 100 submissions

The first few hundred API calls per month are often covered by Anthropic's free tier.

## Docker Deployment Notes

### How Claude Authentication Works

1. **Host Setup**: You authenticate Claude Code on your host machine (`claude auth login`)
2. **Deploy Script**: `./scripts/deploy.sh` copies `~/.config/claude` to `volumes/claude-config/`
3. **Container Mount**: Docker mounts this as `/home/botuser/.config/claude` inside the container
4. **Persistence**: Authentication persists across container restarts via volume mount

### Re-authentication Process

If Claude auth expires or you need to refresh:

```bash
# 1. Re-authenticate on host
claude auth login

# 2. Redeploy to copy new credentials
./scripts/deploy.sh
```

The bot will automatically use the new credentials on restart.

## Getting Help

1. Check logs: `docker logs wosb-market-bot`
2. Review [COMPLETE_SETUP_GUIDE.md](COMPLETE_SETUP_GUIDE.md) troubleshooting section
3. Verify Claude Code: `docker exec -it wosb-market-bot claude auth status`
4. Open a GitHub issue with logs and error messages

---

**Need detailed setup instructions?** → [COMPLETE_SETUP_GUIDE.md](COMPLETE_SETUP_GUIDE.md)
**Want to understand the architecture?** → [ARCHITECTURE.md](ARCHITECTURE.md)
**Looking for all features?** → [IMPLEMENTATION_COMPLETE.md](IMPLEMENTATION_COMPLETE.md)
**Quick command reference?** → [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
