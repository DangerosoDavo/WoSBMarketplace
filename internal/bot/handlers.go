package bot

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// interactionCreate handles all slash command and component interactions
func (b *Bot) interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleCommand(s, i)
	case discordgo.InteractionMessageComponent:
		b.handleComponentInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		b.handleModalSubmit(s, i)
	}
}

// handleComponentInteraction routes button and select menu interactions
func (b *Bot) handleComponentInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	customID := data.CustomID

	// Route based on custom ID prefix
	parts := strings.Split(customID, "_")
	switch {
	case strings.HasPrefix(customID, "port_select_"):
		b.handlePortSelect(s, i, parts)
	case strings.HasPrefix(customID, "port_create"):
		b.handlePortCreate(s, i)
	case strings.HasPrefix(customID, "item_select_"):
		b.handleItemConfirm(s, i, parts)
	case strings.HasPrefix(customID, "trade_contact_"):
		b.handleTradeContactButton(s, i, parts)
	default:
		log.Printf("Unknown component interaction: %s", customID)
	}
}

// handleModalSubmit routes modal submissions
func (b *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	customID := data.CustomID

	// Route based on custom ID prefix
	switch {
	case strings.HasPrefix(customID, "new_port_"):
		b.handleCreatePortModal(s, i)
	default:
		log.Printf("Unknown modal submit: %s", customID)
	}
}

// handleCommand routes slash commands to their handlers
func (b *Bot) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	switch data.Name {
	// User commands
	case "submit":
		b.handleSubmit(s, i)
	case "price":
		b.handlePrice(s, i)
	case "port":
		b.handlePortView(s, i)
	case "ports":
		b.handlePortsList(s, i)
	case "items":
		b.handleItemsList(s, i)
	case "stats":
		b.handleStats(s, i)

	// Admin port commands
	case "admin-port-add":
		b.handleAdminPortAdd(s, i)
	case "admin-port-edit":
		b.handleAdminPortEdit(s, i)
	case "admin-port-remove":
		b.handleAdminPortRemove(s, i)
	case "admin-port-alias":
		b.handleAdminPortAlias(s, i)

	// Admin item commands
	case "admin-item-list-untagged":
		b.handleAdminItemListUntagged(s, i)
	case "admin-item-tag":
		b.handleAdminItemTag(s, i)
	case "admin-item-untag":
		b.handleAdminItemUntag(s, i)
	case "admin-item-alias":
		b.handleAdminItemAlias(s, i)
	case "admin-item-rename":
		b.handleAdminItemRename(s, i)
	case "admin-item-merge":
		b.handleAdminItemMerge(s, i)

	// Admin tag commands
	case "admin-tag-create":
		b.handleAdminTagCreate(s, i)
	case "admin-tag-list":
		b.handleAdminTagList(s, i)
	case "admin-tag-delete":
		b.handleAdminTagDelete(s, i)

	// Admin system commands
	case "admin-expire":
		b.handleAdminExpire(s, i)
	case "admin-purge":
		b.handleAdminPurge(s, i)

	// Configuration commands
	case "config-set-admin-role":
		b.handleConfigSetAdminRole(s, i)
	case "config-show":
		b.handleConfigShow(s, i)

	// Player trading commands
	case "trade-set-name":
		b.handleTradeSetName(s, i)
	case "trade-create":
		b.handleTradeCreate(s, i)
	case "trade-search":
		b.handleTradeSearch(s, i)
	case "trade-my-orders":
		b.handleTradeMyOrders(s, i)
	case "trade-cancel":
		b.handleTradeCancel(s, i)
	case "trade-contact":
		b.handleTradeContact(s, i)
	case "trade-end":
		b.handleTradeEnd(s, i)
	case "trade-report":
		b.handleTradeReport(s, i)

	// Admin trade moderation commands
	case "admin-trade-ban":
		b.handleAdminTradeBan(s, i)
	case "admin-trade-unban":
		b.handleAdminTradeUnban(s, i)
	case "admin-trade-bans":
		b.handleAdminTradeBans(s, i)
	case "admin-trade-reports":
		b.handleAdminTradeReports(s, i)
	case "admin-trade-report-action":
		b.handleAdminTradeReportAction(s, i)

	default:
		b.respondError(s, i, "Unknown command")
	}
}

// Helper functions

func (b *Bot) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", message),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) followUpError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: stringPtr(fmt.Sprintf("❌ %s", message)),
	})
}

func (b *Bot) updateInteractionError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", message),
			Components: []discordgo.MessageComponent{}, // Clear components
		},
	})
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func formatAge(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

func stringPtr(s string) *string {
	return &s
}

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}

// checkAdmin validates if the user is an admin and responds if not
func (b *Bot) checkAdmin(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		b.respondError(s, i, "This command must be used in a server")
		return false
	}
	if !b.isAdmin(i.GuildID, i.Member) {
		b.respondError(s, i, "This command requires the admin role")
		return false
	}
	return true
}

// formatItemList formats a slice of item names for display
func formatItemList(items []string, maxLength int) string {
	result := ""
	for idx, item := range items {
		line := fmt.Sprintf("%d. %s\n", idx+1, item)
		if len(result)+len(line) > maxLength {
			result += fmt.Sprintf("... and %d more", len(items)-idx)
			break
		}
		result += line
	}
	if result == "" {
		result = "None"
	}
	return strings.TrimSpace(result)
}
