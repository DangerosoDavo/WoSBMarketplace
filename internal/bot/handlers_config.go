package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleConfigSetAdminRole sets the admin role for the current guild
func (b *Bot) handleConfigSetAdminRole(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// This command requires Manage Server permission (enforced by Discord via DefaultMemberPermissions)
	if i.GuildID == "" {
		b.respondError(s, i, "This command must be used in a server")
		return
	}

	options := parseOptions(i.ApplicationCommandData().Options)
	roleOption := options["role"]
	if roleOption == nil {
		b.respondError(s, i, "Role is required")
		return
	}

	roleID := roleOption.RoleValue(s, i.GuildID).ID

	// Save to database
	ctx := context.Background()
	err := b.db.SetGuildAdminRole(ctx, i.GuildID, roleID, i.Member.User.ID)
	if err != nil {
		log.Printf("Error setting guild admin role: %v", err)
		b.respondError(s, i, "Failed to save configuration")
		return
	}

	// Get role name for display
	role, err := s.State.Role(i.GuildID, roleID)
	if err != nil {
		// Fallback if we can't get the role name
		role = &discordgo.Role{ID: roleID, Name: "Unknown"}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Configuration Updated",
		Description: fmt.Sprintf("Admin role has been set to **@%s**", role.Name),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Role ID",
				Value:  roleID,
				Inline: true,
			},
			{
				Name:   "Configured By",
				Value:  i.Member.User.Mention(),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Users with this role can now use admin commands",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// handleConfigShow displays current server configuration
func (b *Bot) handleConfigShow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" {
		b.respondError(s, i, "This command must be used in a server")
		return
	}

	ctx := context.Background()
	settings, err := b.db.GetGuildSettings(ctx, i.GuildID)
	if err != nil {
		log.Printf("Error fetching guild settings: %v", err)
		b.respondError(s, i, "Failed to fetch configuration")
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⚙️ Server Configuration",
		Description: "Current bot settings for this server",
		Color:       0x3498db,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if settings == nil || settings.AdminRoleID == "" {
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:  "Admin Role",
				Value: "❌ Not configured",
			},
			{
				Name:  "Setup Instructions",
				Value: "Use `/config-set-admin-role` to configure the admin role for this server",
			},
		}
		embed.Color = 0xe74c3c // Red
	} else {
		// Try to get role name
		roleName := "Unknown Role"
		role, err := s.State.Role(i.GuildID, settings.AdminRoleID)
		if err == nil {
			roleName = "@" + role.Name
		}

		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Admin Role",
				Value:  fmt.Sprintf("**%s** (`%s`)", roleName, settings.AdminRoleID),
				Inline: false,
			},
			{
				Name:   "Configured By",
				Value:  fmt.Sprintf("<@%s>", settings.ConfiguredBy),
				Inline: true,
			},
			{
				Name:   "Last Updated",
				Value:  fmt.Sprintf("<t:%d:R>", settings.UpdatedAt.Unix()),
				Inline: true,
			},
		}

		// Check if global admin role is also set
		if b.adminRoleID != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Global Admin Role (from config)",
				Value: fmt.Sprintf("`%s`", b.adminRoleID),
			})
			embed.Footer = &discordgo.MessageEmbedFooter{
				Text: "Both server-specific and global admin roles are active",
			}
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral, // Only visible to the user
		},
	})
}
