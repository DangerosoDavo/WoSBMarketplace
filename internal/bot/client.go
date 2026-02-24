package bot

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"wosbTrade/internal/database"
	"wosbTrade/internal/ocr"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session            *discordgo.Session
	db                 *database.DB
	claudeClient       *ocr.ClaudeClient
	imagePath          string
	adminRoleID        string
	submissionManager  *SubmissionManager
	tradeConversations *TradeConversationManager
}

type Config struct {
	Token          string
	DatabasePath   string
	ImagePath      string
	ClaudeCodePath string
	AdminRoleID    string
}

// New creates a new Discord bot instance
func New(cfg Config) (*Bot, error) {
	// Create Discord session
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create image storage directory
	if err := os.MkdirAll(cfg.ImagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create image directory: %w", err)
	}

	// Create Claude client
	claudeClient := ocr.NewClaudeClient(cfg.ClaudeCodePath)

	bot := &Bot{
		session:            session,
		db:                 db,
		claudeClient:       claudeClient,
		imagePath:          cfg.ImagePath,
		adminRoleID:        strings.TrimSpace(cfg.AdminRoleID),
		submissionManager:  NewSubmissionManager(5 * time.Minute),
		tradeConversations: NewTradeConversationManager(30 * time.Minute),
	}

	// Set intents
	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentMessageContent |
		discordgo.IntentsDirectMessages

	// Register handlers
	session.AddHandler(bot.ready)
	session.AddHandler(bot.interactionCreate)
	session.AddHandler(bot.messageCreate)

	return bot, nil
}

// Start opens the Discord connection and registers commands
func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open session: %w", err)
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")

	// Register slash commands
	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	// Start background goroutines
	go b.expiryChecker()
	go b.playerOrderExpiryChecker()
	go b.conversationTimeoutChecker()

	// Recover active conversations from DB into memory
	b.recoverActiveConversations()

	// Wait for interrupt signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return nil
}

// Close gracefully shuts down the bot
func (b *Bot) Close() error {
	log.Println("Shutting down bot...")

	if err := b.session.Close(); err != nil {
		log.Printf("Error closing Discord session: %v", err)
	}

	if err := b.db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	return nil
}

// ready handler
func (b *Bot) ready(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)

	// Set bot status
	s.UpdateGameStatus(0, "World of Sea Battle Markets")
}

// expiryChecker runs periodically to remove expired orders
func (b *Bot) expiryChecker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		count, err := b.db.DeleteExpiredOrders(ctx)
		if err != nil {
			log.Printf("Error deleting expired orders: %v", err)
			continue
		}
		if count > 0 {
			log.Printf("Deleted %d expired orders", count)
		}
	}
}

// isAdmin checks if a user has the admin role (checks both global and guild-specific)
func (b *Bot) isAdmin(guildID string, member *discordgo.Member) bool {
	ctx := context.Background()

	// First check guild-specific admin role
	if guildID != "" {
		settings, err := b.db.GetGuildSettings(ctx, guildID)
		if err != nil {
			log.Printf("Error fetching guild settings: %v", err)
		} else if settings != nil && settings.AdminRoleID != "" {
			// Check if member has the guild-specific admin role
			for _, roleID := range member.Roles {
				if roleID == settings.AdminRoleID {
					return true
				}
			}
		}
	}

	// Fall back to global admin role from config
	if b.adminRoleID == "" {
		return false
	}

	// Check if member has the global admin role
	for _, roleID := range member.Roles {
		if roleID == b.adminRoleID {
			return true
		}
	}

	return false
}

// playerOrderExpiryChecker periodically expires player orders
func (b *Bot) playerOrderExpiryChecker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		count, err := b.db.DeleteExpiredPlayerOrders(ctx)
		if err != nil {
			log.Printf("Error expiring player orders: %v", err)
			continue
		}
		if count > 0 {
			log.Printf("Expired %d player orders", count)
		}
	}
}

// conversationTimeoutChecker closes stale trade conversations and notifies both parties
func (b *Bot) conversationTimeoutChecker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		stale, err := b.db.GetStaleConversations(ctx, 30*time.Minute)
		if err != nil {
			log.Printf("Error getting stale conversations: %v", err)
			continue
		}

		for _, conv := range stale {
			// Close in DB
			if err := b.db.CloseTradeConversation(ctx, conv.ID); err != nil {
				log.Printf("Error closing stale conversation %d: %v", conv.ID, err)
				continue
			}

			// Remove from memory
			ac := &ActiveConversation{
				ConversationID:  conv.ID,
				InitiatorUserID: conv.InitiatorUserID,
				CreatorUserID:   conv.CreatorUserID,
			}
			b.tradeConversations.Remove(ac)

			// Notify both parties
			msg := "Your trade conversation has been closed due to inactivity. Use `/trade-search` to find more trades."
			if ch, err := b.session.UserChannelCreate(conv.InitiatorUserID); err == nil {
				b.session.ChannelMessageSend(ch.ID, msg)
			}
			if ch, err := b.session.UserChannelCreate(conv.CreatorUserID); err == nil {
				b.session.ChannelMessageSend(ch.ID, msg)
			}

			log.Printf("Closed stale conversation %d between %s and %s",
				conv.ID, conv.InitiatorIngameName, conv.CreatorIngameName)
		}
	}
}

// recoverActiveConversations loads active conversations from DB into memory on restart
func (b *Bot) recoverActiveConversations() {
	ctx := context.Background()
	convs, err := b.db.GetAllActiveConversations(ctx)
	if err != nil {
		log.Printf("Error recovering active conversations: %v", err)
		return
	}

	for _, conv := range convs {
		ac := &ActiveConversation{
			ConversationID:      conv.ID,
			OrderID:             conv.OrderID,
			InitiatorUserID:     conv.InitiatorUserID,
			InitiatorIngameName: conv.InitiatorIngameName,
			CreatorUserID:       conv.CreatorUserID,
			CreatorIngameName:   conv.CreatorIngameName,
		}
		b.tradeConversations.Register(ac)
	}

	if len(convs) > 0 {
		log.Printf("Recovered %d active trade conversations", len(convs))
	}
}

// hashImage creates a SHA256 hash of an image file
func hashImage(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}
