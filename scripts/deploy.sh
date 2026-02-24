#!/bin/bash
set -e

echo "Deploying World of Sea Battle Market Bot..."

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

# Container name
CONTAINER_NAME="wosb-market-bot"

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Error: .env file not found!"
    echo "Please create .env from .env.example and configure your credentials"
    exit 1
fi

# Stop and remove existing container if it exists
if [ "$(docker ps -aq -f name=$CONTAINER_NAME)" ]; then
    echo "Stopping existing container..."
    docker stop $CONTAINER_NAME || true
    docker rm $CONTAINER_NAME || true
fi

# Create volumes directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/volumes/data"
mkdir -p "$PROJECT_ROOT/volumes/claude-config"

# Check if Claude is authenticated on host
echo "⚠️  IMPORTANT: Claude Code Authentication"
echo "The container needs access to your Claude authentication."
echo ""
if [ -d "$HOME/.config/claude" ]; then
    echo "Found Claude config at ~/.config/claude"
    echo "Copying Claude authentication to container volume..."
    cp -r "$HOME/.config/claude" "$PROJECT_ROOT/volumes/claude-config/"
else
    echo "❌ Claude authentication not found!"
    echo "Please authenticate Claude first:"
    echo "  claude auth login"
    echo ""
    read -p "Press Enter after authenticating, or Ctrl+C to cancel..."
    if [ -d "$HOME/.config/claude" ]; then
        cp -r "$HOME/.config/claude" "$PROJECT_ROOT/volumes/claude-config/"
    else
        echo "Still not found. Deployment will fail without authentication."
        exit 1
    fi
fi

# Run the container
echo "Starting container..."
docker run -d \
    --name $CONTAINER_NAME \
    --restart unless-stopped \
    --env-file .env \
    -v "$PROJECT_ROOT/volumes/data:/data" \
    -v "$PROJECT_ROOT/volumes/claude-config/claude:/home/botuser/.config/claude" \
    wosb-market-bot:latest

echo "Container started successfully!"
echo ""
echo "Useful commands:"
echo "  View logs:        docker logs -f $CONTAINER_NAME"
echo "  Stop container:   docker stop $CONTAINER_NAME"
echo "  Restart:          docker restart $CONTAINER_NAME"
echo "  Shell access:     docker exec -it $CONTAINER_NAME sh"
