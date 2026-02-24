.PHONY: help build deploy restart logs test clean fmt vet db-backup

# Default target
help:
	@echo "World of Sea Battle Market Bot - Available Commands:"
	@echo ""
	@echo "  make build        - Build Docker image"
	@echo "  make deploy       - Deploy/start the bot container"
	@echo "  make restart      - Rebuild and restart the bot"
	@echo "  make logs         - Follow bot logs"
	@echo "  make stop         - Stop the bot container"
	@echo "  make test         - Run Go tests"
	@echo "  make clean        - Remove containers and volumes"
	@echo "  make fmt          - Format Go code"
	@echo "  make vet          - Run go vet"
	@echo "  make db-backup    - Backup the database"
	@echo "  make db-shell     - Open SQLite shell"
	@echo ""

# Build Docker image
build:
	@echo "Building Docker image..."
	@./scripts/build.sh

# Deploy the bot
deploy:
	@echo "Deploying bot..."
	@./scripts/deploy.sh

# Rebuild and restart
restart: build deploy
	@echo "Bot restarted successfully!"

# View logs
logs:
	@docker logs -f wosb-market-bot

# Stop the bot
stop:
	@echo "Stopping bot..."
	@docker stop wosb-market-bot || true

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean up containers and build artifacts
clean:
	@echo "Cleaning up..."
	@docker stop wosb-market-bot || true
	@docker rm wosb-market-bot || true
	@docker rmi wosb-market-bot:latest || true
	@rm -f coverage.out coverage.html

# Format Go code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Backup database
db-backup:
	@echo "Backing up database..."
	@mkdir -p backups
	@docker exec wosb-market-bot sh -c "cp /data/database.db /data/database-backup-$$(date +%Y%m%d-%H%M%S).db"
	@docker cp wosb-market-bot:/data/database-backup-*.db ./backups/ || true
	@echo "Backup complete: ./backups/"

# Open database shell
db-shell:
	@docker exec -it wosb-market-bot sqlite3 /data/database.db

# View database stats
db-stats:
	@docker exec wosb-market-bot sqlite3 /data/database.db "SELECT 'Total Orders:', COUNT(*) FROM markets; SELECT 'Unique Ports:', COUNT(DISTINCT port) FROM markets; SELECT 'Buy Orders:', COUNT(*) FROM markets WHERE order_type='buy'; SELECT 'Sell Orders:', COUNT(*) FROM markets WHERE order_type='sell';"

# Shell into container
shell:
	@docker exec -it wosb-market-bot sh

# Check container status
status:
	@docker ps -a | grep wosb-market-bot || echo "Container not running"

# Run locally (without Docker)
run-local:
	@echo "Running bot locally..."
	@go run cmd/bot/main.go

# Docker compose commands
compose-up:
	@docker-compose up -d

compose-down:
	@docker-compose down

compose-logs:
	@docker-compose logs -f
