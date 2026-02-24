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

// parseBanDuration converts duration choice strings to time.Duration
func parseBanDuration(d string) time.Duration {
	switch d {
	case "1d":
		return 24 * time.Hour
	case "3d":
		return 3 * 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "14d":
		return 14 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return 0
	}
}

// --- /trade-report ---

func (b *Bot) handleTradeReport(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := getUserID(i)
	options := parseOptions(i.ApplicationCommandData().Options)
	orderID := int(options["order-id"].IntValue())
	reason := strings.TrimSpace(options["reason"].StringValue())

	if len(reason) < 5 || len(reason) > 500 {
		b.respondError(s, i, "Report reason must be between 5 and 500 characters")
		return
	}

	ctx := context.Background()

	// Look up the order to get the reported user
	order, err := b.db.GetPlayerOrder(ctx, orderID)
	if err != nil {
		log.Printf("Error getting order for report: %v", err)
		b.respondError(s, i, "Failed to look up order")
		return
	}
	if order == nil {
		b.respondError(s, i, "Order not found or has expired")
		return
	}

	// Can't report yourself
	if order.UserID == userID {
		b.respondError(s, i, "You cannot report your own order")
		return
	}

	report := database.TradeReport{
		ReporterUserID: userID,
		ReportedUserID: order.UserID,
		OrderID:        &orderID,
		Reason:         reason,
	}

	_, err = b.db.CreateTradeReport(ctx, report)
	if err != nil {
		log.Printf("Error creating trade report: %v", err)
		b.respondError(s, i, "Failed to submit report")
		return
	}

	b.respondEphemeral(s, i, "Your report has been submitted and will be reviewed by an admin. Thank you.")
}

// --- /admin-trade-ban ---

func (b *Bot) handleAdminTradeBan(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	targetUser := options["user"].UserValue(s)
	reason := options["reason"].StringValue()

	var expiresAt *time.Time
	if opt := options["duration"]; opt != nil {
		durStr := opt.StringValue()
		if durStr != "permanent" {
			dur := parseBanDuration(durStr)
			if dur > 0 {
				t := time.Now().Add(dur)
				expiresAt = &t
			}
		}
	}

	ctx := context.Background()

	// Check if already banned
	existing, _ := b.db.IsUserBanned(ctx, targetUser.ID)
	if existing != nil {
		b.respondError(s, i, "This user is already banned from trading")
		return
	}

	ban := database.TradeBan{
		UserID:    targetUser.ID,
		Reason:    reason,
		BannedBy:  i.Member.User.ID,
		ExpiresAt: expiresAt,
	}

	_, err := b.db.CreateTradeBan(ctx, ban)
	if err != nil {
		log.Printf("Error creating trade ban: %v", err)
		b.respondError(s, i, "Failed to create trade ban")
		return
	}

	// Cancel all their active orders
	cancelled, _ := b.db.CancelAllUserOrders(ctx, targetUser.ID)

	expStr := "Permanent"
	if expiresAt != nil {
		expStr = fmt.Sprintf("<t:%d:F>", expiresAt.Unix())
	}

	embed := &discordgo.MessageEmbed{
		Title: "Trade Ban Issued",
		Color: 0xe74c3c,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "User", Value: fmt.Sprintf("<@%s>", targetUser.ID), Inline: true},
			{Name: "Reason", Value: reason, Inline: true},
			{Name: "Duration", Value: expStr, Inline: true},
			{Name: "Banned By", Value: fmt.Sprintf("<@%s>", i.Member.User.ID), Inline: true},
			{Name: "Orders Cancelled", Value: fmt.Sprintf("%d", cancelled), Inline: true},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// --- /admin-trade-unban ---

func (b *Bot) handleAdminTradeUnban(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	targetUser := options["user"].UserValue(s)

	ctx := context.Background()
	err := b.db.RemoveTradeBan(ctx, targetUser.ID, i.Member.User.ID)
	if err != nil {
		b.respondError(s, i, err.Error())
		return
	}

	b.respondEphemeral(s, i, fmt.Sprintf("Trade ban removed for <@%s>.", targetUser.ID))
}

// --- /admin-trade-bans ---

func (b *Bot) handleAdminTradeBans(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	ctx := context.Background()
	bans, err := b.db.GetActiveTradeBans(ctx)
	if err != nil {
		log.Printf("Error getting trade bans: %v", err)
		b.respondError(s, i, "Failed to retrieve trade bans")
		return
	}

	if len(bans) == 0 {
		b.respondEphemeral(s, i, "No active trade bans.")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Active Trade Bans",
		Description: fmt.Sprintf("%d active ban(s)", len(bans)),
		Color:       0xe74c3c,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	for _, ban := range bans {
		expStr := "Never (permanent)"
		if ban.ExpiresAt != nil {
			expStr = fmt.Sprintf("<t:%d:R>", ban.ExpiresAt.Unix())
		}

		value := fmt.Sprintf("Reason: %s\nBanned by: <@%s>\nExpires: %s",
			ban.Reason, ban.BannedBy, expStr)

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("Ban #%d — <@%s>", ban.ID, ban.UserID),
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

// --- /admin-trade-reports ---

func (b *Bot) handleAdminTradeReports(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	status := "pending"
	if opt := options["status"]; opt != nil {
		status = opt.StringValue()
	}

	ctx := context.Background()
	reports, err := b.db.GetTradeReports(ctx, status)
	if err != nil {
		log.Printf("Error getting trade reports: %v", err)
		b.respondError(s, i, "Failed to retrieve trade reports")
		return
	}

	if len(reports) == 0 {
		b.respondEphemeral(s, i, fmt.Sprintf("No %s trade reports.", status))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Trade Reports (%s)", strings.Title(status)),
		Description: fmt.Sprintf("%d report(s)", len(reports)),
		Color:       0xf39c12,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	for _, report := range reports {
		orderInfo := "N/A"
		if report.OrderID != nil {
			orderInfo = fmt.Sprintf("#%d", *report.OrderID)
		}

		value := fmt.Sprintf("Reporter: <@%s>\nReported: <@%s>\nOrder: %s\nReason: %s\nSubmitted: <t:%d:R>",
			report.ReporterUserID, report.ReportedUserID, orderInfo,
			report.Reason, report.CreatedAt.Unix())

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("Report #%d", report.ID),
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

// --- /admin-trade-report-action ---

func (b *Bot) handleAdminTradeReportAction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.checkAdmin(s, i) {
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	reportID := int(options["report-id"].IntValue())
	action := options["action"].StringValue()

	ctx := context.Background()

	report, err := b.db.GetTradeReport(ctx, reportID)
	if err != nil {
		log.Printf("Error getting trade report: %v", err)
		b.respondError(s, i, "Failed to retrieve report")
		return
	}
	if report == nil {
		b.respondError(s, i, "Report not found")
		return
	}
	if report.Status != "pending" {
		b.respondError(s, i, fmt.Sprintf("Report has already been actioned (status: %s)", report.Status))
		return
	}

	adminID := i.Member.User.ID

	switch action {
	case "dismiss":
		err := b.db.UpdateTradeReportStatus(ctx, reportID, "dismissed", adminID)
		if err != nil {
			log.Printf("Error dismissing report: %v", err)
			b.respondError(s, i, "Failed to dismiss report")
			return
		}
		b.respondEphemeral(s, i, fmt.Sprintf("Report #%d dismissed.", reportID))

	case "ban":
		// Mark report as reviewed
		err := b.db.UpdateTradeReportStatus(ctx, reportID, "reviewed", adminID)
		if err != nil {
			log.Printf("Error updating report status: %v", err)
			b.respondError(s, i, "Failed to update report")
			return
		}

		// Determine ban reason
		reason := fmt.Sprintf("Reported: %s", report.Reason)
		if opt := options["reason"]; opt != nil {
			reason = opt.StringValue()
		}

		// Check if already banned
		existing, _ := b.db.IsUserBanned(ctx, report.ReportedUserID)
		if existing != nil {
			b.respondEphemeral(s, i, fmt.Sprintf("Report #%d reviewed. User <@%s> is already banned.", reportID, report.ReportedUserID))
			return
		}

		// Create permanent ban
		ban := database.TradeBan{
			UserID:   report.ReportedUserID,
			Reason:   reason,
			BannedBy: adminID,
		}
		_, err = b.db.CreateTradeBan(ctx, ban)
		if err != nil {
			log.Printf("Error creating ban from report: %v", err)
			b.respondError(s, i, "Failed to ban user")
			return
		}

		// Cancel their active orders
		cancelled, _ := b.db.CancelAllUserOrders(ctx, report.ReportedUserID)

		embed := &discordgo.MessageEmbed{
			Title: fmt.Sprintf("Report #%d — User Banned", reportID),
			Color: 0xe74c3c,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Reported User", Value: fmt.Sprintf("<@%s>", report.ReportedUserID), Inline: true},
				{Name: "Ban Reason", Value: reason, Inline: true},
				{Name: "Orders Cancelled", Value: fmt.Sprintf("%d", cancelled), Inline: true},
				{Name: "Original Reporter", Value: fmt.Sprintf("<@%s>", report.ReporterUserID), Inline: true},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
			},
		})
	}
}
