#!/bin/bash
set -e

echo "🚀 Initializing World of Sea Battle Market Bot..."

# Check for Claude Code CLI
echo "🔍 Checking for Claude Code CLI..."
if ! command -v claude &> /dev/null; then
    echo "⚠️  Claude Code CLI not found!"
    echo "Please install it first:"
    echo "  npm install -g @anthropic-ai/claude-code"
    echo "  claude auth login"
    echo ""
    echo "Then run this script again."
    exit 1
fi

# Check Claude authentication
if ! claude auth status &> /dev/null; then
    echo "⚠️  Claude Code not authenticated!"
    echo "Please authenticate:"
    echo "  claude auth login"
    echo ""
    echo "Then run this script again."
    exit 1
fi

echo "✅ Claude Code CLI found and authenticated"

# Create data directories
echo "📁 Creating data directories..."
mkdir -p data/images
mkdir -p volumes/data

# Check for .env file
if [ ! -f ".env" ]; then
    echo "⚙️  Creating .env from template..."
    cp .env.example .env
    echo ""
    echo "⚠️  IMPORTANT: Edit .env and add your credentials:"
    echo "   - DISCORD_TOKEN (from Discord Developer Portal)"
    echo "   - ADMIN_ROLE_ID (Create a role in Discord, right-click it, Copy ID)"
    echo ""
    echo "Note: Claude authentication is handled by 'claude auth login'"
    echo ""
    echo "How to get Admin Role ID:"
    echo "  1. Enable Developer Mode in Discord (Settings → Advanced)"
    echo "  2. Go to Server Settings → Roles"
    echo "  3. Create a role named 'WOSB Admin'"
    echo "  4. Right-click the role → Copy ID"
    echo "  5. Paste the ID into ADMIN_ROLE_ID in .env"
    echo ""
    echo "Run this script again after configuring .env"
    exit 0
fi

# Check if credentials are set
if grep -q "your_discord_bot_token_here" .env; then
    echo "❌ Error: .env file still contains placeholder values"
    echo "Please edit .env and add your actual Discord token"
    exit 1
fi

# Download Go dependencies
echo "📦 Downloading Go dependencies..."
go mod download
go mod tidy

echo ""
echo "✅ Initialization complete!"
echo ""
echo "Next steps:"
echo "  1. Review your .env configuration"
echo "  2. Run locally: go run cmd/bot/main.go"
echo "  3. Or build Docker: ./scripts/build.sh"
echo "  4. Or deploy: ./scripts/deploy.sh"
echo ""
