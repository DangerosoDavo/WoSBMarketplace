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

// User Query Handlers

func (b *Bot) handlePrice(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := parseOptions(i.ApplicationCommandData().Options)
	itemName := options["item"].StringValue()

	region := ""
	minPrice := 0
	maxPrice := 0

	if opt := options["region"]; opt != nil {
		region = opt.StringValue()
	}
	if opt := options["min-price"]; opt != nil {
		minPrice = int(opt.IntValue())
	}
	if opt := options["max-price"]; opt != nil {
		maxPrice = int(opt.IntValue())
	}

	ctx := context.Background()

	// Find item
	matches, err := b.db.FindItemMatches(ctx, itemName, 1)
	if err != nil || len(matches) == 0 {
		b.respondError(s, i, fmt.Sprintf("Item not found: %s", itemName))
		return
	}

	item := matches[0].Item

	// Query prices
	markets, err := b.db.GetPricesByItem(ctx, item.ID, nil, region, minPrice, maxPrice)
	if err != nil {
		log.Printf("Error querying prices: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	if len(markets) == 0 {
		filterInfo := ""
		if region != "" || minPrice > 0 || maxPrice > 0 {
			filterInfo = " (with current filters)"
		}
		b.respondError(s, i, fmt.Sprintf("No active orders found for '%s'%s", item.DisplayName, filterInfo))
		return
	}

	// Group by buy/sell
	buyOrders := []database.Market{}
	sellOrders := []database.Market{}
	for _, m := range markets {
		if m.OrderType == "buy" {
			buyOrders = append(buyOrders, m)
		} else {
			sellOrders = append(sellOrders, m)
		}
	}

	description := fmt.Sprintf("Showing best prices across all ports")
	if region != "" {
		description += fmt.Sprintf(" (Region: %s)", region)
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("💰 Prices for: %s", item.DisplayName),
		Description: description,
		Color:       0x3498db,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if len(buyOrders) > 0 {
		buyText := ""
		for idx, m := range buyOrders {
			if idx >= 5 {
				break
			}
			age := time.Since(m.SubmittedAt)
			buyText += fmt.Sprintf("**%s**: %d gold (qty: %d) - %s\n",
				m.Port.DisplayName, m.Price, m.Quantity, formatAge(age))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Buy Orders",
			Value: buyText,
		})
	}

	if len(sellOrders) > 0 {
		sellText := ""
		for idx, m := range sellOrders {
			if idx >= 5 {
				break
			}
			age := time.Since(m.SubmittedAt)
			sellText += fmt.Sprintf("**%s**: %d gold (qty: %d) - %s\n",
				m.Port.DisplayName, m.Price, m.Quantity, formatAge(age))
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Sell Orders",
			Value: sellText,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (b *Bot) handlePortView(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := parseOptions(i.ApplicationCommandData().Options)
	portName := options["name"].StringValue()

	ctx := context.Background()

	// Find port
	matches, err := b.db.FindPortMatches(ctx, portName, 1)
	if err != nil || len(matches) == 0 {
		b.respondError(s, i, fmt.Sprintf("Port not found: %s", portName))
		return
	}

	port := matches[0].Port

	// Get orders
	markets, err := b.db.GetOrdersByPort(ctx, port.ID)
	if err != nil {
		log.Printf("Error querying port: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	if len(markets) == 0 {
		b.respondError(s, i, fmt.Sprintf("No active orders found for port '%s'", port.DisplayName))
		return
	}

	// Group by buy/sell
	buyOrders := []database.Market{}
	sellOrders := []database.Market{}
	for _, m := range markets {
		if m.OrderType == "buy" {
			buyOrders = append(buyOrders, m)
		} else {
			sellOrders = append(sellOrders, m)
		}
	}

	description := "All active market orders"
	if port.Region != "" {
		description += fmt.Sprintf(" (Region: %s)", port.Region)
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🏴‍☠️ Port: %s", port.DisplayName),
		Description: description,
		Color:       0x9b59b6,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if len(buyOrders) > 0 {
		buyText := ""
		for _, m := range buyOrders {
			buyText += fmt.Sprintf("**%s**: %d gold (qty: %d)\n", m.Item.DisplayName, m.Price, m.Quantity)
		}
		if len(buyText) > 1024 {
			buyText = buyText[:1021] + "..."
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Buy Orders",
			Value: buyText,
		})
	}

	if len(sellOrders) > 0 {
		sellText := ""
		for _, m := range sellOrders {
			sellText += fmt.Sprintf("**%s**: %d gold (qty: %d)\n", m.Item.DisplayName, m.Price, m.Quantity)
		}
		if len(sellText) > 1024 {
			sellText = sellText[:1021] + "..."
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Sell Orders",
			Value: sellText,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (b *Bot) handlePortsList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := parseOptions(i.ApplicationCommandData().Options)
	region := ""
	if opt := options["region"]; opt != nil {
		region = opt.StringValue()
	}

	ctx := context.Background()
	ports, err := b.db.GetAllPorts(ctx)
	if err != nil {
		log.Printf("Error getting ports: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	if len(ports) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No ports found",
			},
		})
		return
	}

	// Filter by region if specified
	if region != "" {
		filtered := []database.Port{}
		for _, port := range ports {
			if strings.EqualFold(port.Region, region) {
				filtered = append(filtered, port)
			}
		}
		ports = filtered
	}

	// Group by region
	byRegion := make(map[string][]string)
	for _, port := range ports {
		reg := port.Region
		if reg == "" {
			reg = "Unknown"
		}
		byRegion[reg] = append(byRegion[reg], port.DisplayName)
	}

	title := "🗺️ All Ports"
	if region != "" {
		title = fmt.Sprintf("🗺️ Ports in %s", region)
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("Total: %d ports", len(ports)),
		Color:       0x2ecc71,
	}

	for reg, portList := range byRegion {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   reg,
			Value:  strings.Join(portList, ", "),
			Inline: false,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (b *Bot) handleItemsList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := parseOptions(i.ApplicationCommandData().Options)
	tagsStr := ""
	if opt := options["tags"]; opt != nil {
		tagsStr = opt.StringValue()
	}

	ctx := context.Background()

	if tagsStr == "" {
		// Show all items grouped by tags
		b.respondError(s, i, "Tag filtering not yet fully implemented")
		return
	}

	// Parse tags
	tagNames := strings.Split(tagsStr, ",")
	var tagIDs []int

	allTags, err := b.db.GetAllTags(ctx, "")
	if err != nil {
		b.respondError(s, i, "Database error")
		return
	}

	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		for _, tag := range allTags {
			if strings.EqualFold(tag.Name, tagName) {
				tagIDs = append(tagIDs, tag.ID)
				break
			}
		}
	}

	if len(tagIDs) == 0 {
		b.respondError(s, i, "No valid tags found")
		return
	}

	// Query items with these tags
	markets, err := b.db.GetOrdersByTags(ctx, tagIDs, "")
	if err != nil {
		b.respondError(s, i, "Database error")
		return
	}

	if len(markets) == 0 {
		b.respondError(s, i, "No items found with those tags")
		return
	}

	// Get unique items
	itemMap := make(map[int]bool)
	var itemNames []string
	for _, m := range markets {
		if !itemMap[m.ItemID] {
			itemMap[m.ItemID] = true
			itemNames = append(itemNames, m.Item.DisplayName)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📦 Items",
		Description: fmt.Sprintf("Items tagged with: %s", tagsStr),
		Color:       0xe74c3c,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  fmt.Sprintf("Found %d items", len(itemNames)),
				Value: strings.Join(itemNames, ", "),
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (b *Bot) handleStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	stats, err := b.db.GetStats(ctx)
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 Bot Statistics",
		Description: "World of Sea Battle Market Tracker",
		Color:       0xe67e22,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Active Orders",
				Value:  fmt.Sprintf("%d", stats["total_orders"]),
				Inline: true,
			},
			{
				Name:   "Ports Tracked",
				Value:  fmt.Sprintf("%d", stats["unique_ports"]),
				Inline: true,
			},
			{
				Name:   "Total Ports",
				Value:  fmt.Sprintf("%d", stats["total_ports"]),
				Inline: true,
			},
			{
				Name:   "Total Items",
				Value:  fmt.Sprintf("%d", stats["total_items"]),
				Inline: true,
			},
			{
				Name:   "Untagged Items",
				Value:  fmt.Sprintf("%d", stats["untagged_items"]),
				Inline: true,
			},
			{
				Name:   "Submissions Today",
				Value:  fmt.Sprintf("%d", stats["submissions_today"]),
				Inline: true,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if lastUpdate, ok := stats["last_update"].(time.Time); ok {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Last Update",
			Value: fmt.Sprintf("<t:%d:R>", lastUpdate.Unix()),
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
