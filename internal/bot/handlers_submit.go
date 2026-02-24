package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wosbTrade/internal/database"

	"github.com/bwmarrin/discordgo"
)

// handleSubmit processes screenshot submissions with port and item confirmation
func (b *Bot) handleSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer response to allow processing time
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	options := parseOptions(i.ApplicationCommandData().Options)
	orderType := options["order-type"].StringValue()
	attachmentID := options["screenshot"].Value.(string)

	// Get attachment
	var attachment *discordgo.MessageAttachment
	for _, att := range i.ApplicationCommandData().Resolved.Attachments {
		if att.ID == attachmentID {
			attachment = att
			break
		}
	}

	if attachment == nil {
		b.followUpError(s, i, "Could not find attached image")
		return
	}

	// Validate image type
	if !strings.HasPrefix(attachment.ContentType, "image/") {
		b.followUpError(s, i, "Attachment must be an image (PNG, JPEG, WebP)")
		return
	}

	// Download image
	userID := i.Member.User.ID
	imagePath := filepath.Join(b.imagePath, fmt.Sprintf("%s_%d_%s", userID, time.Now().Unix(), attachment.Filename))

	if err := downloadFile(attachment.URL, imagePath); err != nil {
		log.Printf("Error downloading image: %v", err)
		b.followUpError(s, i, "Failed to download image")
		return
	}

	// Hash the image
	imgHash, err := hashImage(imagePath)
	if err != nil {
		log.Printf("Error hashing image: %v", err)
		imgHash = "unknown"
	}

	// Analyze with Claude
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	marketData, err := b.claudeClient.AnalyzeScreenshot(ctx, imagePath)
	if err != nil {
		log.Printf("Error analyzing screenshot: %v", err)
		os.Remove(imagePath)
		b.followUpError(s, i, fmt.Sprintf("Failed to analyze screenshot: %v", err))
		return
	}

	// Validate order type matches detected type
	if marketData.OrderType != orderType {
		os.Remove(imagePath)
		b.followUpError(s, i, fmt.Sprintf(
			"Order type mismatch: you selected '%s' but the screenshot shows '%s' orders",
			orderType, marketData.OrderType,
		))
		return
	}

	// Create pending submission
	submission := b.submissionManager.Create(
		userID,
		i.ChannelID,
		i.Interaction.ID,
		imagePath,
		imgHash,
		orderType,
		marketData,
	)

	// Start port matching process
	b.processPortMatching(s, i, submission)
}

// processPortMatching handles port validation and confirmation
func (b *Bot) processPortMatching(s *discordgo.Session, i *discordgo.InteractionCreate, sub *PendingSubmission) {
	ctx := context.Background()

	// Find port matches
	matches, err := b.db.FindPortMatches(ctx, sub.OCRResult.Port, 10)
	if err != nil {
		log.Printf("Error finding port matches: %v", err)
		b.submissionManager.Remove(sub.UserID)
		os.Remove(sub.ImagePath)
		b.followUpError(s, i, "Database error during port matching")
		return
	}

	// Check for exact match
	if len(matches) > 0 && matches[0].Confidence == database.ConfidenceExact {
		// Auto-confirm exact match
		b.submissionManager.ConfirmPort(sub.UserID, matches[0].Port.ID)

		// Move to item matching
		b.processItemMatching(s, i, sub)
		return
	}

	// Show port selection UI
	b.showPortSelectionUI(s, i, sub, matches)
}

// showPortSelectionUI displays port options to user
func (b *Bot) showPortSelectionUI(s *discordgo.Session, i *discordgo.InteractionCreate, sub *PendingSubmission, matches []database.PortMatch) {
	embed := &discordgo.MessageEmbed{
		Title:       "🏴‍☠️ Port Confirmation Needed",
		Description: fmt.Sprintf("OCR detected port: **%s**\n\nPlease select the correct port or create a new one:", sub.OCRResult.Port),
		Color:       0xffa500,
	}

	// Build select menu options
	var options []discordgo.SelectMenuOption
	for idx, match := range matches {
		if idx >= 10 {
			break
		}

		label := match.Port.DisplayName
		if match.Port.Region != "" {
			label += fmt.Sprintf(" (%s)", match.Port.Region)
		}

		description := fmt.Sprintf("%.0f%% match", match.Score*100)
		if match.MatchedVia == "exact" {
			description = "Exact match"
		} else if match.MatchedVia == "alias" {
			description = "Matched via alias"
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", match.Port.ID),
			Description: description,
		})
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("port_select:%s", sub.UserID),
					Placeholder: "Select a port",
					Options:     options,
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Create New Port",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("port_create:%s", sub.UserID),
				},
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

// handlePortSelect processes port selection from dropdown
func (b *Bot) handlePortSelect(s *discordgo.Session, i *discordgo.InteractionCreate, parts []string) {
	userID := i.Member.User.ID
	data := i.MessageComponentData()

	if len(data.Values) == 0 {
		return
	}

	var portID int
	fmt.Sscanf(data.Values[0], "%d", &portID)

	// Confirm port
	if !b.submissionManager.ConfirmPort(userID, portID) {
		b.respondError(s, i, "Submission expired or not found")
		return
	}

	// Get submission
	sub, ok := b.submissionManager.Get(userID)
	if !ok {
		b.respondError(s, i, "Submission expired")
		return
	}

	// Acknowledge selection
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	// Move to item matching
	b.processItemMatching(s, i, sub)
}

// handlePortCreate shows modal for creating new port
func (b *Bot) handlePortCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	sub, ok := b.submissionManager.Get(userID)
	if !ok {
		b.respondError(s, i, "Submission expired")
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("create_port:%s", userID),
			Title:    "Create New Port",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "port_name",
							Label:       "Port Name",
							Style:       discordgo.TextInputShort,
							Placeholder: sub.OCRResult.Port,
							Value:       sub.OCRResult.Port,
							Required:    true,
							MaxLength:   100,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "port_region",
							Label:       "Region",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g., Caribbean, Mediterranean",
							Required:    true,
							MaxLength:   50,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "port_notes",
							Label:       "Notes (optional)",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Any additional information...",
							Required:    false,
							MaxLength:   500,
						},
					},
				},
			},
		},
	})
}

// handleCreatePortModal processes the create port modal submission
func (b *Bot) handleCreatePortModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	sub, ok := b.submissionManager.Get(userID)
	if !ok {
		b.respondError(s, i, "Submission expired")
		return
	}

	data := i.ModalSubmitData()

	var portName, portRegion, portNotes string
	for _, row := range data.Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			textInput := comp.(*discordgo.TextInput)
			switch textInput.CustomID {
			case "port_name":
				portName = textInput.Value
			case "port_region":
				portRegion = textInput.Value
			case "port_notes":
				portNotes = textInput.Value
			}
		}
	}

	// Create port in database
	ctx := context.Background()
	port, err := b.db.CreatePort(ctx, portName, portName, portRegion, userID)
	if err != nil {
		log.Printf("Error creating port: %v", err)
		b.respondError(s, i, "Failed to create port")
		return
	}

	// Add notes if provided
	if portNotes != "" {
		// TODO: Update port notes
	}

	// Confirm port
	b.submissionManager.ConfirmPort(userID, port.ID)

	// Acknowledge
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	// Move to item matching
	b.processItemMatching(s, i, sub)
}

// Continued in handlers_submit_items.go...
