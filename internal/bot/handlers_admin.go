package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Admin Port Management Handlers

func (b *Bot) handleAdminPortAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	name := options["name"].StringValue()
	region := options["region"].StringValue()
	notes := ""
	if opt := options["notes"]; opt != nil {
		notes = opt.StringValue()
	}

	ctx := context.Background()
	port, err := b.db.CreatePort(ctx, name, name, region, i.Member.User.ID)
	if err != nil {
		log.Printf("Error creating port: %v", err)
		b.respondError(s, i, "Failed to create port (may already exist)")
		return
	}

	_ = notes // TODO: Add notes support
	_ = port

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Created port: **%s** (Region: %s)", name, region),
		},
	})
}

func (b *Bot) handleAdminPortEdit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Port editing not yet implemented")
	// TODO: Implement port editing
}

func (b *Bot) handleAdminPortRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Port removal not yet implemented")
	// TODO: Implement port removal with confirmation
}

func (b *Bot) handleAdminPortAlias(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Port alias not yet implemented")
	// TODO: Implement port alias creation
}

// Admin Item Management Handlers

func (b *Bot) handleAdminItemListUntagged(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	limit := 10
	if opt := options["limit"]; opt != nil {
		limit = int(opt.IntValue())
	}

	ctx := context.Background()
	items, err := b.db.GetUntaggedItems(ctx, limit)
	if err != nil {
		log.Printf("Error getting untagged items: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	if len(items) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ No untagged items! All items have been categorized.",
			},
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📋 Untagged Items",
		Description: fmt.Sprintf("Showing %d untagged items that need categorization:", len(items)),
		Color:       0xe67e22,
	}

	var itemList string
	for idx, item := range items {
		itemList += fmt.Sprintf("%d. **%s** (added %s by <@%s>)\n",
			idx+1, item.DisplayName, formatAge(item.AddedAt.Sub(item.AddedAt)), item.AddedBy)
	}

	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:  "Items",
			Value: itemList,
		},
		{
			Name:  "How to Tag",
			Value: "Use `/admin-item-tag <item> <tags>` to categorize items",
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (b *Bot) handleAdminItemTag(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	itemName := options["item"].StringValue()
	tagNames := options["tags"].StringValue()

	ctx := context.Background()

	// Find item
	item, err := b.db.GetItemByName(ctx, itemName)
	if err != nil {
		b.respondError(s, i, fmt.Sprintf("Item not found: %s", itemName))
		return
	}

	// Parse tag names
	tagNameList := strings.Split(tagNames, ",")
	var tagIDs []int

	for _, tagName := range tagNameList {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}

		// Find or suggest tag
		allTags, err := b.db.GetAllTags(ctx, "")
		if err != nil {
			continue
		}

		found := false
		for _, tag := range allTags {
			if strings.EqualFold(tag.Name, tagName) {
				tagIDs = append(tagIDs, tag.ID)
				found = true
				break
			}
		}

		if !found {
			b.respondError(s, i, fmt.Sprintf("Tag not found: %s. Create it first with `/admin-tag-create`", tagName))
			return
		}
	}

	if len(tagIDs) == 0 {
		b.respondError(s, i, "No valid tags provided")
		return
	}

	// Add tags to item
	err = b.db.AddTagsToItem(ctx, item.ID, tagIDs)
	if err != nil {
		log.Printf("Error adding tags: %v", err)
		b.respondError(s, i, "Failed to add tags")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Tagged **%s** with: %s", item.DisplayName, tagNames),
		},
	})
}

func (b *Bot) handleAdminItemUntag(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Item untagging not yet implemented")
	// TODO: Implement item untagging
}

func (b *Bot) handleAdminItemAlias(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Item alias not yet implemented")
	// TODO: Implement item alias creation
}

func (b *Bot) handleAdminItemRename(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Item renaming not yet implemented")
	// TODO: Implement item renaming
}

func (b *Bot) handleAdminItemMerge(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Item merging not yet implemented")
	// TODO: Implement item merging with market order transfer
}

// Admin Tag Management Handlers

func (b *Bot) handleAdminTagCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	name := options["name"].StringValue()
	category := options["category"].StringValue()
	icon := ""
	color := ""

	if opt := options["icon"]; opt != nil {
		icon = opt.StringValue()
	}
	if opt := options["color"]; opt != nil {
		color = opt.StringValue()
	}

	ctx := context.Background()
	tag, err := b.db.CreateTag(ctx, name, category, color, icon)
	if err != nil {
		log.Printf("Error creating tag: %v", err)
		b.respondError(s, i, "Failed to create tag (may already exist)")
		return
	}

	response := fmt.Sprintf("✅ Created tag: **%s** (Category: %s)", tag.Name, tag.Category)
	if icon != "" {
		response += fmt.Sprintf(" %s", icon)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
		},
	})
}

func (b *Bot) handleAdminTagList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	category := ""
	if opt := options["category"]; opt != nil {
		category = opt.StringValue()
	}

	ctx := context.Background()
	tags, err := b.db.GetAllTags(ctx, category)
	if err != nil {
		log.Printf("Error getting tags: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	if len(tags) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No tags found",
			},
		})
		return
	}

	// Group by category
	byCategory := make(map[string][]string)
	for _, tag := range tags {
		cat := tag.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		tagStr := tag.Name
		if tag.Icon != "" {
			tagStr = tag.Icon + " " + tagStr
		}
		byCategory[cat] = append(byCategory[cat], tagStr)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏷️ Available Tags",
		Description: fmt.Sprintf("Total: %d tags", len(tags)),
		Color:       0x9b59b6,
	}

	for cat, tagList := range byCategory {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   cat,
			Value:  strings.Join(tagList, ", "),
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

func (b *Bot) handleAdminTagDelete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	b.respondError(s, i, "Tag deletion not yet implemented")
	// TODO: Implement tag deletion with confirmation
}

// Admin System Handlers

func (b *Bot) handleAdminExpire(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	ctx := context.Background()
	count, err := b.db.DeleteExpiredOrders(ctx)
	if err != nil {
		log.Printf("Error deleting expired orders: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Deleted %d expired orders", count),
		},
	})
}

func (b *Bot) handleAdminPurge(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	portName := options["port"].StringValue()

	ctx := context.Background()

	// Find port
	port, err := b.db.GetPortByName(ctx, portName)
	if err != nil {
		b.respondError(s, i, fmt.Sprintf("Port not found: %s", portName))
		return
	}

	count, err := b.db.PurgePort(ctx, port.ID, i.Member.User.ID)
	if err != nil {
		log.Printf("Error purging port: %v", err)
		b.respondError(s, i, "Database error")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Purged %d orders from port '%s'", count, port.DisplayName),
		},
	})
}
