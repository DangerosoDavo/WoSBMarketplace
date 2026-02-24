# Setup Guide - World of Sea Battle Market Bot

This guide will walk you through setting up the Discord bot from scratch.

## Prerequisites

- Docker installed on your system
- A Discord account
- An Anthropic API account (for Claude)
- Git installed

## Step 1: Create Discord Bot

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)

2. Click "New Application" in the top right
   - Name it something like "WOSB Market Bot"
   - Click "Create"

3. Navigate to the "Bot" section in the left sidebar
   - Click "Add Bot" and confirm
   - Under "Privileged Gateway Intents", enable:
     - ✅ Message Content Intent
     - ✅ Server Members Intent (optional, for better user tracking)

4. Copy your bot token
   - Click "Reset Token" if needed
   - Click "Copy" to copy the token
   - **SAVE THIS TOKEN SECURELY** - you'll need it for the `.env` file

5. Navigate to "OAuth2" → "URL Generator"
   - Select scopes:
     - ✅ `bot`
     - ✅ `applications.commands`
   - Select bot permissions:
     - ✅ Send Messages
     - ✅ Embed Links
     - ✅ Attach Files
     - ✅ Read Message History
     - ✅ Use Slash Commands

6. Copy the generated URL at the bottom
   - Open it in your browser
   - Select the server you want to add the bot to
   - Click "Authorize"

## Step 2: Get Claude API Key

1. Go to [Anthropic Console](https://console.anthropic.com/)

2. Sign up or log in

3. Navigate to "API Keys"

4. Click "Create Key"
   - Give it a name like "WOSB Market Bot"
   - Copy the API key
   - **SAVE THIS KEY SECURELY**

5. Ensure you have credits or billing set up
   - The bot uses Claude 3.5 Sonnet for image analysis
   - Each image analysis costs approximately $0.01-0.03 depending on size

## Step 3: Clone and Configure

1. Clone the repository:
```bash
git clone https://github.com/yourusername/wosbTrade.git
cd wosbTrade
```

2. Create your `.env` file:
```bash
cp .env.example .env
```

3. Edit `.env` with your credentials:
```bash
nano .env  # or use your preferred editor
```

Add your tokens:
```env
DISCORD_TOKEN=your_discord_bot_token_here
ANTHROPIC_API_KEY=your_anthropic_api_key_here
DATABASE_PATH=/data/database.db
IMAGE_STORAGE_PATH=/data/images
LOG_LEVEL=info
ADMIN_USER_IDS=your_discord_user_id,another_admin_id
```

**To find your Discord User ID:**
- Enable Developer Mode in Discord (Settings → Advanced → Developer Mode)
- Right-click your username → Copy ID

## Step 4: Build and Deploy

1. Build the Docker image:
```bash
./scripts/build.sh
```

This will:
- Create a multi-stage Docker build
- Compile the Go application with SQLite support
- Create an Alpine-based runtime container

2. Deploy the container:
```bash
./scripts/deploy.sh
```

This will:
- Create necessary volume directories
- Start the container with persistent storage
- Mount the database and image storage

3. Verify it's running:
```bash
docker logs -f wosb-market-bot
```

You should see:
```
Logged in as: WOSB Market Bot#1234
Registering slash commands...
Registered command: submit
Registered command: price
...
Bot is now running. Press CTRL-C to exit.
```

## Step 5: Test the Bot

1. In Discord, type `/` in any channel where the bot has permissions

2. You should see the bot's commands:
   - `/submit` - Submit a market screenshot
   - `/price` - Query item prices
   - `/port` - View port orders
   - `/stats` - View bot statistics

3. Test with `/stats` to verify the bot is responding

## Development Workflow

### Making Changes

1. Edit the source code

2. Rebuild and redeploy:
```bash
./scripts/build.sh && ./scripts/deploy.sh
```

The database and any stored data will persist across restarts.

### Viewing Logs

```bash
docker logs -f wosb-market-bot
```

### Accessing the Database

```bash
docker exec -it wosb-market-bot sh
cd /data
sqlite3 database.db
```

Example queries:
```sql
-- View all markets
SELECT * FROM markets LIMIT 10;

-- View recent submissions
SELECT * FROM audit_log ORDER BY timestamp DESC LIMIT 10;

-- Count orders by port
SELECT port, COUNT(*) FROM markets GROUP BY port;
```

### Stopping the Bot

```bash
docker stop wosb-market-bot
```

### Restarting the Bot

```bash
docker restart wosb-market-bot
```

## Data Persistence

The bot uses Docker volumes to persist data:

```
wosbTrade/
└── volumes/
    └── data/
        ├── database.db        # SQLite database
        ├── database.db-shm    # SQLite shared memory
        ├── database.db-wal    # SQLite write-ahead log
        └── images/            # Temporary image storage
```

This data persists even when you rebuild/restart the container.

## Troubleshooting

### Bot doesn't respond to commands

1. Check the bot is online in Discord (green status)
2. Verify bot has permissions in the channel
3. Check logs: `docker logs wosb-market-bot`
4. Ensure slash commands are registered (wait 5 minutes after first start)

### "Failed to analyze screenshot" errors

1. Verify your Anthropic API key is valid
2. Check you have API credits available
3. Ensure the image is a valid format (PNG, JPEG, WebP)
4. Check logs for detailed error messages

### Database errors

1. Ensure the volumes directory has proper permissions
2. Try stopping and removing the container, then redeploying:
```bash
docker stop wosb-market-bot
docker rm wosb-market-bot
./scripts/deploy.sh
```

### Container won't start

1. Check Docker is running
2. Verify .env file exists and has correct values
3. Check logs: `docker logs wosb-market-bot`
4. Ensure ports aren't in use by another container

## Security Best Practices

1. **Never commit `.env` to git**
   - It's in `.gitignore` by default
   - Keep tokens secure

2. **Limit admin access**
   - Only add trusted user IDs to `ADMIN_USER_IDS`
   - Admins can purge data and run manual expiry

3. **Regular backups**
   - Backup `volumes/data/database.db` regularly
   - Consider automated backups for production

4. **Monitor API usage**
   - Check Anthropic Console for usage
   - Each screenshot costs ~$0.01-0.03
   - Set billing alerts if desired

## Production Deployment

For production use:

1. **Use a proper server** (not your local machine)
   - VPS (DigitalOcean, Linode, etc.)
   - Cloud (AWS, GCP, Azure)

2. **Set up automated backups**
```bash
# Example backup script
#!/bin/bash
BACKUP_DIR="/backups/wosb"
DATE=$(date +%Y%m%d_%H%M%S)
cp volumes/data/database.db "$BACKUP_DIR/database_$DATE.db"
# Keep only last 30 days
find $BACKUP_DIR -name "database_*.db" -mtime +30 -delete
```

3. **Configure Docker to start on boot**
```bash
docker update --restart unless-stopped wosb-market-bot
```

4. **Set up monitoring**
   - Use Docker health checks
   - Monitor logs for errors
   - Track API costs

5. **Use environment-specific configs**
   - Separate `.env.production` and `.env.development`
   - Never use the same tokens for dev and prod

## Updating the Bot

When there's a new version:

```bash
git pull origin main
./scripts/build.sh
./scripts/deploy.sh
```

Your data will persist through updates.

## Getting Help

- Check logs first: `docker logs -f wosb-market-bot`
- Review error messages in Discord (bot replies with errors)
- Open an issue on GitHub with:
  - What you were trying to do
  - Error messages from logs
  - Steps to reproduce

## Next Steps

- Invite friends to submit market data
- Build a community around tracking prices
- Consider adding price history charts (future feature)
- Set up alerts for price thresholds (future feature)
