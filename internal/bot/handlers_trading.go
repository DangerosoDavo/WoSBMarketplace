package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"wosbTrade/internal/database"

	"github.com/bwmarrin/discordgo"
)

// parseTradeDuration converts duration choice strings to time.Duration
func parseTradeDuration(d string) time.Duration {
	switch d {
	case "1d":
		return 24 * time.Hour
	case "3d":
		return 3 * 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "14d":
		return 14 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

// getUserID extracts user ID from an interaction, handling both guild and DM contexts
func getUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

// --- /trade-set-name ---

func (b *Bot) handleTradeSetName(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := parseOptions(i.ApplicationCommandData().Options)
	name := strings.TrimSpace(options["name"].StringValue())

	if len(name) < 2 || len(name) > 50 {
		b.respondError(s, i, "In-game name must be between 2 and 50 characters")
		return
	}

	userID := getUserID(i)
	ctx := context.Background()

	err := b.db.SetPlayerProfile(ctx, userID, name)
	if err != nil {
		log.Printf("Error setting player profile: %v", err)
		b.respondError(s, i, "Failed to save your in-game name")
		return
	}

	b.respondEphemeral(s, i, fmt.Sprintf("Your in-game name has been set to **%s**", name))
}

// --- /trade-create ---

func (b *Bot) handleTradeCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	ctx := context.Background()

	// Check player has set their name
	profile, err := b.db.GetPlayerProfile(ctx, userID)
	if err != nil || profile == nil {
		b.respondError(s, i, "You need to set your in-game name first. Use `/trade-set-name`")
		return
	}

	// Check if user is banned from trading
	ban, err := b.db.IsUserBanned(ctx, userID)
	if err != nil {
		log.Printf("Error checking trade ban: %v", err)
		b.respondError(s, i, "Failed to verify trading status")
		return
	}
	if ban != nil {
		msg := fmt.Sprintf("You are banned from trading. Reason: %s", ban.Reason)
		if ban.ExpiresAt != nil {
			msg += fmt.Sprintf("\nBan expires: <t:%d:R>", ban.ExpiresAt.Unix())
		}
		b.respondError(s, i, msg)
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	orderType := options["type"].StringValue()
	itemName := options["item"].StringValue()
	price := int(options["price"].IntValue())
	quantity := int(options["quantity"].IntValue())
	duration := options["duration"].StringValue()

	if price <= 0 {
		b.respondError(s, i, "Price must be greater than 0")
		return
	}
	if quantity <= 0 {
		b.respondError(s, i, "Quantity must be greater than 0")
		return
	}

	// Find item using fuzzy matching
	matches, err := b.db.FindItemMatches(ctx, itemName, 5)
	if err != nil {
		log.Printf("Error finding item matches: %v", err)
		b.respondError(s, i, "Database error during item search")
		return
	}

	var itemID int
	var itemDisplay string
	if len(matches) > 0 && matches[0].Confidence >= database.ConfidenceMedium {
		itemID = matches[0].Item.ID
		itemDisplay = matches[0].Item.DisplayName
	} else {
		// Create new item
		newItem, err := b.db.CreateItem(ctx, itemName, itemName, userID)
		if err != nil {
			log.Printf("Error creating item: %v", err)
			b.respondError(s, i, "Failed to create new item")
			return
		}
		itemID = newItem.ID
		itemDisplay = itemName
	}

	// Optional port
	var portID *int
	var portDisplay string
	if opt := options["port"]; opt != nil {
		portName := opt.StringValue()
		portMatches, err := b.db.FindPortMatches(ctx, portName, 1)
		if err == nil && len(portMatches) > 0 && portMatches[0].Confidence >= database.ConfidenceMedium {
			id := portMatches[0].Port.ID
			portID = &id
			portDisplay = portMatches[0].Port.DisplayName
		} else {
			b.respondError(s, i, fmt.Sprintf("Port not found: '%s'. Ask an admin to add it with `/admin-port-add`, or omit the port.", portName))
			return
		}
	}

	// Optional notes
	notes := ""
	if opt := options["notes"]; opt != nil {
		notes = opt.StringValue()
	}

	// Calculate expiry
	dur := parseTradeDuration(duration)
	expiresAt := time.Now().Add(dur)

	order := database.PlayerOrder{
		UserID:     userID,
		ItemID:     itemID,
		OrderType:  orderType,
		Price:      price,
		Quantity:   quantity,
		PortID:     portID,
		Notes:      notes,
		IngameName: profile.IngameName,
		ExpiresAt:  expiresAt,
	}

	created, err := b.db.CreatePlayerOrder(ctx, order)
	if err != nil {
		log.Printf("Error creating player order: %v", err)
		b.respondError(s, i, "Failed to create order")
		return
	}

	typeEmoji := "📗"
	if orderType == "sell" {
		typeEmoji = "📕"
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s Trade Order Created", typeEmoji),
		Color: 0x2ecc71,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Order ID", Value: fmt.Sprintf("#%d", created.ID), Inline: true},
			{Name: "Type", Value: strings.ToUpper(orderType), Inline: true},
			{Name: "Item", Value: itemDisplay, Inline: true},
			{Name: "Price", Value: fmt.Sprintf("%d gold", price), Inline: true},
			{Name: "Quantity", Value: fmt.Sprintf("%d", quantity), Inline: true},
			{Name: "Expires", Value: fmt.Sprintf("<t:%d:R>", expiresAt.Unix()), Inline: true},
			{Name: "Trader", Value: profile.IngameName, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Other players can contact you about this order with /trade-contact",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if portDisplay != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Port", Value: portDisplay, Inline: true,
		})
	}
	if notes != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Notes", Value: notes,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// --- /trade-search ---

func (b *Bot) handleTradeSearch(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := parseOptions(i.ApplicationCommandData().Options)
	ctx := context.Background()

	var itemID, portID, minPrice, maxPrice int
	var orderType string

	if opt := options["item"]; opt != nil {
		matches, err := b.db.FindItemMatches(ctx, opt.StringValue(), 1)
		if err == nil && len(matches) > 0 {
			itemID = matches[0].Item.ID
		} else {
			b.respondError(s, i, fmt.Sprintf("Item not found: '%s'", opt.StringValue()))
			return
		}
	}

	if opt := options["port"]; opt != nil {
		matches, err := b.db.FindPortMatches(ctx, opt.StringValue(), 1)
		if err == nil && len(matches) > 0 {
			portID = matches[0].Port.ID
		}
	}

	if opt := options["type"]; opt != nil {
		orderType = opt.StringValue()
	}
	if opt := options["min-price"]; opt != nil {
		minPrice = int(opt.IntValue())
	}
	if opt := options["max-price"]; opt != nil {
		maxPrice = int(opt.IntValue())
	}

	orders, err := b.db.SearchPlayerOrders(ctx, itemID, orderType, portID, minPrice, maxPrice, 20)
	if err != nil {
		log.Printf("Error searching player orders: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	if len(orders) == 0 {
		b.respondError(s, i, "No player orders found matching your criteria")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔍 Player Trade Orders",
		Description: fmt.Sprintf("Found %d order(s)", len(orders)),
		Color:       0xf39c12,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	displayCount := len(orders)
	if displayCount > 10 {
		displayCount = 10
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing 10 of %d results. Refine your search for more specific results.", len(orders)),
		}
	}

	for idx := 0; idx < displayCount; idx++ {
		o := orders[idx]
		typeEmoji := "📗"
		if o.OrderType == "sell" {
			typeEmoji = "📕"
		}

		portInfo := ""
		if o.Port != nil {
			portInfo = fmt.Sprintf(" @ %s", o.Port.DisplayName)
		}

		value := fmt.Sprintf("%s **%s** %s%s - %d gold x%d\nBy: **%s** | Expires <t:%d:R>",
			typeEmoji, strings.ToUpper(o.OrderType), o.Item.DisplayName, portInfo,
			o.Price, o.Quantity, o.IngameName, o.ExpiresAt.Unix())

		if o.Notes != "" {
			value += fmt.Sprintf("\n> %s", o.Notes)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("Order #%d", o.ID),
			Value: value,
		})
	}

	// Add contact buttons (max 5 per action row)
	var buttons []discordgo.MessageComponent
	buttonCount := displayCount
	if buttonCount > 5 {
		buttonCount = 5
	}
	for idx := 0; idx < buttonCount; idx++ {
		o := orders[idx]
		buttons = append(buttons, discordgo.Button{
			Label:    fmt.Sprintf("Contact #%d", o.ID),
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("trade_contact_%d", o.ID),
		})
	}

	var components []discordgo.MessageComponent
	if len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{Components: buttons})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

// --- /trade-my-orders ---

func (b *Bot) handleTradeMyOrders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	ctx := context.Background()

	orders, err := b.db.GetPlayerOrdersByUser(ctx, userID)
	if err != nil {
		log.Printf("Error getting user orders: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	if len(orders) == 0 {
		b.respondEphemeral(s, i, "You have no active trade orders. Create one with `/trade-create`")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📋 Your Active Trade Orders",
		Description: fmt.Sprintf("%d active order(s)", len(orders)),
		Color:       0x3498db,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	for _, o := range orders {
		typeEmoji := "📗"
		if o.OrderType == "sell" {
			typeEmoji = "📕"
		}

		portInfo := "Any port"
		if o.Port != nil {
			portInfo = o.Port.DisplayName
		}

		value := fmt.Sprintf("%s %s | %d gold x%d | Port: %s\nExpires <t:%d:R>",
			typeEmoji, o.Item.DisplayName, o.Price, o.Quantity,
			portInfo, o.ExpiresAt.Unix())

		if o.Notes != "" {
			value += fmt.Sprintf("\n> %s", o.Notes)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("Order #%d", o.ID),
			Value: value,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// --- /trade-cancel ---

func (b *Bot) handleTradeCancel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	options := parseOptions(i.ApplicationCommandData().Options)
	orderID := int(options["order-id"].IntValue())

	ctx := context.Background()
	err := b.db.CancelPlayerOrder(ctx, orderID, userID)
	if err != nil {
		log.Printf("Error cancelling order: %v", err)
		b.respondError(s, i, "Failed to cancel order. Make sure the order ID is correct and belongs to you.")
		return
	}

	b.respondEphemeral(s, i, fmt.Sprintf("Order #%d has been cancelled.", orderID))
}

// --- /trade-contact (slash command) ---

func (b *Bot) handleTradeContact(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	options := parseOptions(i.ApplicationCommandData().Options)
	orderID := int(options["order-id"].IntValue())

	b.initiateTradeContact(s, i, userID, orderID)
}

// --- trade_contact_ button handler ---

func (b *Bot) handleTradeContactButton(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	if len(parts) < 3 {
		return
	}
	userID := getUserID(i)
	var orderID int
	fmt.Sscanf(parts[2], "%d", &orderID)
	if orderID == 0 {
		return
	}

	b.initiateTradeContact(s, i, userID, orderID)
}

// --- Core contact initiation logic ---

func (b *Bot) initiateTradeContact(s *discordgo.Session, i *discordgo.InteractionCreate, userID string, orderID int) {
	ctx := context.Background()

	// Check user has a profile
	profile, err := b.db.GetPlayerProfile(ctx, userID)
	if err != nil || profile == nil {
		b.respondError(s, i, "You need to set your in-game name first. Use `/trade-set-name`")
		return
	}

	// Check if initiating user is banned from trading
	ban, err := b.db.IsUserBanned(ctx, userID)
	if err != nil {
		log.Printf("Error checking trade ban: %v", err)
		b.respondError(s, i, "Failed to verify trading status")
		return
	}
	if ban != nil {
		b.respondError(s, i, "You are banned from trading and cannot contact other traders.")
		return
	}

	// Get the order
	order, err := b.db.GetPlayerOrder(ctx, orderID)
	if err != nil || order == nil {
		b.respondError(s, i, "Order not found or has expired")
		return
	}

	// Check if order creator is banned (safety net)
	creatorBan, _ := b.db.IsUserBanned(ctx, order.UserID)
	if creatorBan != nil {
		b.respondError(s, i, "This order is no longer available.")
		return
	}

	// Can't contact yourself
	if order.UserID == userID {
		b.respondError(s, i, "You cannot contact yourself about your own order")
		return
	}

	// Create conversation in DB
	conv := database.TradeConversation{
		OrderID:             orderID,
		InitiatorUserID:     userID,
		InitiatorIngameName: profile.IngameName,
		CreatorUserID:       order.UserID,
		CreatorIngameName:   order.IngameName,
	}

	// TryRegister atomically checks neither party has an active conversation
	ac := &ActiveConversation{
		OrderID:             orderID,
		InitiatorUserID:     userID,
		InitiatorIngameName: profile.IngameName,
		CreatorUserID:       order.UserID,
		CreatorIngameName:   order.IngameName,
	}

	if !b.tradeConversations.TryRegister(ac) {
		// Check which party is busy
		if b.tradeConversations.HasActiveConversation(userID) {
			b.respondError(s, i, "You already have an active trade conversation. End it with `/trade-end` first.")
		} else {
			b.respondError(s, i, "The order creator is currently in another trade conversation. Try again later.")
		}
		return
	}

	created, err := b.db.CreateTradeConversation(ctx, conv)
	if err != nil {
		log.Printf("Error creating trade conversation: %v", err)
		b.tradeConversations.Remove(ac) // Rollback in-memory registration
		b.respondError(s, i, "Failed to start trade conversation")
		return
	}

	// Update the in-memory conversation with the DB ID
	ac.ConversationID = created.ID

	// Respond to the initiator
	b.respondEphemeral(s, i, fmt.Sprintf(
		"Trade conversation started! Check your DMs to chat with **%s** about order #%d (%s %s).\n\nUse `/trade-end` to close the conversation.",
		order.IngameName, orderID, strings.ToUpper(order.OrderType), order.Item.DisplayName,
	))

	// DM the initiator with instructions
	initiatorCh, err := s.UserChannelCreate(userID)
	if err == nil {
		initiatorEmbed := &discordgo.MessageEmbed{
			Title:       "🤝 Trade Conversation Started",
			Description: fmt.Sprintf("You're now chatting with **%s** about order #%d", order.IngameName, orderID),
			Color:       0x2ecc71,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Order", Value: fmt.Sprintf("%s %s - %d gold x%d",
					strings.ToUpper(order.OrderType), order.Item.DisplayName, order.Price, order.Quantity)},
				{Name: "How to chat", Value: "Type your messages here and they'll be relayed to the other trader."},
				{Name: "To end", Value: "Use `/trade-end` to close this conversation."},
			},
		}
		s.ChannelMessageSendEmbed(initiatorCh.ID, initiatorEmbed)
	}

	// DM the order creator
	creatorCh, err := s.UserChannelCreate(order.UserID)
	if err != nil {
		log.Printf("Error creating DM channel with order creator %s: %v", order.UserID, err)
		return
	}

	creatorEmbed := &discordgo.MessageEmbed{
		Title:       "🤝 Trade Conversation Started",
		Description: fmt.Sprintf("**%s** wants to discuss your order #%d", profile.IngameName, orderID),
		Color:       0x2ecc71,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Order", Value: fmt.Sprintf("%s %s - %d gold x%d",
				strings.ToUpper(order.OrderType), order.Item.DisplayName, order.Price, order.Quantity)},
			{Name: "How to respond", Value: "Type your messages here and they'll be relayed to the other trader."},
			{Name: "To end", Value: "Use `/trade-end` to close this conversation."},
		},
	}

	s.ChannelMessageSendEmbed(creatorCh.ID, creatorEmbed)
}

// --- /trade-end ---

func (b *Bot) handleTradeEnd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)

	ac, ok := b.tradeConversations.GetByUser(userID)
	if !ok {
		b.respondError(s, i, "You don't have an active trade conversation")
		return
	}

	// Close in DB
	ctx := context.Background()
	b.db.CloseTradeConversation(ctx, ac.ConversationID)

	// Determine other party
	otherUserID, otherIngameName := ac.GetOtherParty(userID)
	myIngameName := ac.GetIngameName(userID)

	// Remove from memory
	b.tradeConversations.Remove(ac)

	// Respond to the user who ended it
	b.respondEphemeral(s, i, fmt.Sprintf("Trade conversation with **%s** has been ended.", otherIngameName))

	// Notify the other party via DM
	otherCh, err := s.UserChannelCreate(otherUserID)
	if err == nil {
		s.ChannelMessageSend(otherCh.ID, fmt.Sprintf(
			"**%s** has ended the trade conversation. You can browse more trades with `/trade-search`.",
			myIngameName,
		))
	}
}
