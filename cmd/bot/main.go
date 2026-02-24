package main

import (
	"log"
	"os"

	"wosbTrade/internal/bot"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Validate required environment variables
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	claudeCodePath := os.Getenv("CLAUDE_CODE_PATH")
	if claudeCodePath == "" {
		claudeCodePath = "claude" // Default to 'claude' in PATH
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/database.db"
	}

	imagePath := os.Getenv("IMAGE_STORAGE_PATH")
	if imagePath == "" {
		imagePath = "./data/images"
	}

	adminRoleID := os.Getenv("ADMIN_ROLE_ID")

	// Create bot instance
	config := bot.Config{
		Token:          token,
		DatabasePath:   dbPath,
		ImagePath:      imagePath,
		ClaudeCodePath: claudeCodePath,
		AdminRoleID:    adminRoleID,
	}

	b, err := bot.New(config)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	defer b.Close()

	// Start bot
	if err := b.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
}
