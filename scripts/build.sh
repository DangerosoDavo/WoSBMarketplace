#!/bin/bash
set -e

echo "Building World of Sea Battle Market Bot Docker image..."

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

# Build the Docker image
docker build -t wosb-market-bot:latest -f deployments/Dockerfile .

echo "Build complete! Image: wosb-market-bot:latest"
