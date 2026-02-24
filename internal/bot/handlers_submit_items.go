package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"wosbTrade/internal/database"

	"github.com/bwmarrin/discordgo"
)

// processItemMatching handles item validation and confirmation
func (b *Bot) processItemMatching(s *discordgo.Session, i *discordgo.InteractionCreate, sub *PendingSubmission) {
	ctx := context.Background()

	// Get unique items that haven't been confirmed yet
	_ = sub.GetUniqueOCRItems() // For future use
	unconfirmedItems := sub.GetUnconfirmedItems()

	// If all items are confirmed, proceed to database commit
	if len(unconfirmedItems) == 0 {
		b.commitSubmission(s, i, sub)
		return
	}

	// Process next unconfirmed item
	nextItem := unconfirmedItems[0]

	// Find matches for this item
	matches, err := b.db.FindItemMatches(ctx, nextItem, 5)
	if err != nil {
		log.Printf("Error finding item matches: %v", err)
		b.submissionManager.Remove(sub.UserID)
		os.Remove(sub.ImagePath)
		b.followUpError(s, i, "Database error during item matching")
		return
	}

	// High confidence auto-match
	if len(matches) > 0 && matches[0].Confidence == database.ConfidenceHigh {
		b.submissionManager.AddItemMapping(sub.UserID, nextItem, matches[0].Item.ID)

		// Check if all items done
		if sub.IsComplete() {
			b.commitSubmission(s, i, sub)
		} else {
			// Process next item
			b.processItemMatching(s, i, sub)
		}
		return
	}

	// Exact match auto-confirm
	if len(matches) > 0 && matches[0].Confidence == database.ConfidenceExact {
		b.submissionManager.AddItemMapping(sub.UserID, nextItem, matches[0].Item.ID)

		if sub.IsComplete() {
			b.commitSubmission(s, i, sub)
		} else {
			b.processItemMatching(s, i, sub)
		}
		return
	}

	// Medium/Low confidence - ask user
	b.showItemConfirmationUI(s, i, sub, nextItem, matches)
}

// showItemConfirmationUI displays item matching options to user
func (b *Bot) showItemConfirmationUI(s *discordgo.Session, i *discordgo.InteractionCreate, sub *PendingSubmission, itemName string, matches []database.ItemMatch) {
	totalItems := len(sub.GetUniqueOCRItems())
	confirmedItems := len(sub.ItemMappings)

	embed := &discordgo.MessageEmbed{
		Title:       "🎯 Item Confirmation",
		Description: fmt.Sprintf("**OCR detected**: `%s`\n\nProgress: %d/%d items confirmed", itemName, confirmedItems, totalItems),
		Color:       0x3498db,
	}

	// Build select menu options
	var options []discordgo.SelectMenuOption

	for idx, match := range matches {
		if idx >= 5 {
			break
		}

		label := match.Item.DisplayName
		description := fmt.Sprintf("%.0f%% match", match.Score*100)

		// Add tag info if available
		tags, _ := b.db.GetItemTags(context.Background(), match.Item.ID)
		if len(tags) > 0 {
			tagNames := []string{}
			for _, tag := range tags {
				if len(tagNames) < 3 {
					tagNames = append(tagNames, tag.Name)
				}
			}
			if len(tagNames) > 0 {
				description += " • " + strings.Join(tagNames, ", ")
			}
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", match.Item.ID),
			Description: description,
		})
	}

	// Add "Create New Item" option
	options = append(options, discordgo.SelectMenuOption{
		Label:       "✨ Add as new item: " + itemName,
		Value:       "new",
		Description: "This will create a new untagged item",
	})

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("item_confirm:%s:%s", sub.UserID, itemName),
					Placeholder: "Select matching item",
					Options:     options,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Cancel",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("submission_cancel:%s", sub.UserID),
				},
			},
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
}

// handleItemConfirm processes item selection from dropdown
func (b *Bot) handleItemConfirm(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	if len(parts) < 3 {
		return
	}

	userID := i.Member.User.ID
	itemName := parts[2]
	data := i.MessageComponentData()

	if len(data.Values) == 0 {
		return
	}

	sub, ok := b.submissionManager.Get(userID)
	if !ok {
		b.respondError(s, i, "Submission expired")
		return
	}

	selectedValue := data.Values[0]

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	if selectedValue == "new" {
		// Create new item
		ctx := context.Background()
		newItem, err := b.db.CreateItem(ctx, itemName, itemName, userID)
		if err != nil {
			log.Printf("Error creating item: %v", err)
			b.followUpError(s, i, "Failed to create new item")
			return
		}

		b.submissionManager.AddItemMapping(userID, itemName, newItem.ID)
	} else {
		// Use selected item
		var itemID int
		fmt.Sscanf(selectedValue, "%d", &itemID)
		b.submissionManager.AddItemMapping(userID, itemName, itemID)
	}

	// Continue with next item or commit
	if sub.IsComplete() {
		b.commitSubmission(s, i, sub)
	} else {
		b.processItemMatching(s, i, sub)
	}
}

// commitSubmission finalizes the submission and stores in database
func (b *Bot) commitSubmission(s *discordgo.Session, i *discordgo.InteractionCreate, sub *PendingSubmission) {
	ctx := context.Background()

	// Build market orders
	orders, err := b.submissionManager.GetMarketOrders(sub.UserID)
	if err != nil || orders == nil {
		log.Printf("Error building market orders: %v", err)
		b.followUpError(s, i, "Failed to build market orders")
		return
	}

	// Commit to database
	err = b.db.ReplacePortOrders(
		ctx,
		*sub.PortID,
		sub.OrderType,
		orders,
		sub.UserID,
		sub.ScreenshotHash,
	)
	if err != nil {
		log.Printf("Error storing orders: %v", err)
		b.followUpError(s, i, "Failed to store market data")
		return
	}

	// Get port name for response
	port, _ := b.db.GetPortByName(ctx, sub.OCRResult.Port)
	portName := sub.OCRResult.Port
	if port != nil {
		portName = port.DisplayName
	}

	// Count new items added
	newItems := []string{}
	for ocrName, itemID := range sub.ItemMappings {
		item, err := b.db.GetItemByName(ctx, ocrName)
		if err == nil && item != nil && !item.IsTagged {
			newItems = append(newItems, item.DisplayName)
		}
		_ = itemID // suppress unused warning
	}

	// Cleanup
	b.submissionManager.Remove(sub.UserID)
	os.Remove(sub.ImagePath)

	// Success response
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Market Data Updated",
		Description: fmt.Sprintf("Successfully processed %s orders for **%s**", sub.OrderType, portName),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Items Updated",
				Value:  fmt.Sprintf("%d", len(sub.OCRResult.Items)),
				Inline: true,
			},
			{
				Name:   "Unique Items",
				Value:  fmt.Sprintf("%d", len(sub.GetUniqueOCRItems())),
				Inline: true,
			},
			{
				Name:   "Expires",
				Value:  fmt.Sprintf("<t:%d:R>", time.Now().AddDate(0, 0, 7).Unix()),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data will automatically expire after 7 days",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if len(newItems) > 0 {
		newItemsList := strings.Join(newItems, ", ")
		if len(newItemsList) > 1024 {
			newItemsList = newItemsList[:1021] + "..."
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "ℹ️ New Items Added (Untagged)",
			Value: newItemsList + "\n\nAdmins can tag these with `/admin-item-tag`",
		})
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{}, // Clear components
	})
}
