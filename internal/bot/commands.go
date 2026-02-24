package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

var (
	// Permission value for commands that require Manage Server permission
	adminPermission int64 = discordgo.PermissionManageServer
)

var commands = []*discordgo.ApplicationCommand{
	// User Commands
	{
		Name:        "submit",
		Description: "Submit a market screenshot (attach image)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "order-type",
				Description: "Type of orders in the screenshot",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Buy Orders",
						Value: "buy",
					},
					{
						Name:  "Sell Orders",
						Value: "sell",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Name:        "screenshot",
				Description: "Market screenshot image",
				Required:    true,
			},
		},
	},
	{
		Name:        "price",
		Description: "Query prices for an item across all ports",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to search for",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "region",
				Description: "Filter by port region (optional)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "min-price",
				Description: "Minimum price filter (optional)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "max-price",
				Description: "Maximum price filter (optional)",
				Required:    false,
			},
		},
	},
	{
		Name:        "port",
		Description: "View all active orders at a specific port",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Port name",
				Required:    true,
			},
		},
	},
	{
		Name:        "ports",
		Description: "List all ports",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "region",
				Description: "Filter by region (optional)",
				Required:    false,
			},
		},
	},
	{
		Name:        "items",
		Description: "Browse items by tags",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "tags",
				Description: "Comma-separated tag names (e.g., weapon,heavy)",
				Required:    false,
			},
		},
	},
	{
		Name:        "stats",
		Description: "Show bot statistics",
	},

	// Admin Commands - Port Management
	{
		Name:        "admin-port-add",
		Description: "Add a new port (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Port name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "region",
				Description: "Port region",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "notes",
				Description: "Additional notes (optional)",
				Required:    false,
			},
		},
	},
	{
		Name:        "admin-port-edit",
		Description: "Edit a port (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Port name to edit",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "new-name",
				Description: "New port name (optional)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "region",
				Description: "New region (optional)",
				Required:    false,
			},
		},
	},
	{
		Name:        "admin-port-remove",
		Description: "Remove a port (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Port name to remove",
				Required:    true,
			},
		},
	},
	{
		Name:        "admin-port-alias",
		Description: "Add an alias to a port for OCR matching (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "port",
				Description: "Port name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "alias",
				Description: "Alias to add (e.g., 'Pt Royal' for 'Port Royal')",
				Required:    true,
			},
		},
	},

	// Admin Commands - Item Management
	{
		Name:        "admin-item-list-untagged",
		Description: "List items that need tagging (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "Number of items to show (default: 10)",
				Required:    false,
			},
		},
	},
	{
		Name:        "admin-item-tag",
		Description: "Add tags to an item (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "tags",
				Description: "Comma-separated tag names (e.g., weapon,heavy,long-range)",
				Required:    true,
			},
		},
	},
	{
		Name:        "admin-item-untag",
		Description: "Remove tags from an item (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "tags",
				Description: "Comma-separated tag names to remove",
				Required:    true,
			},
		},
	},
	{
		Name:        "admin-item-alias",
		Description: "Add an alias to an item for OCR matching (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "alias",
				Description: "Alias to add (e.g., 'cannon ball' for 'Cannonball')",
				Required:    true,
			},
		},
	},
	{
		Name:        "admin-item-rename",
		Description: "Rename an item (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "old-name",
				Description: "Current item name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "new-name",
				Description: "New item name",
				Required:    true,
			},
		},
	},
	{
		Name:        "admin-item-merge",
		Description: "Merge duplicate items (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "from",
				Description: "Item to merge from (will be deleted)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "to",
				Description: "Item to merge into (will be kept)",
				Required:    true,
			},
		},
	},

	// Admin Commands - Tag Management
	{
		Name:        "admin-tag-create",
		Description: "Create a new tag (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Tag name (e.g., 'weapon', 'heavy')",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "category",
				Description: "Tag category (e.g., 'type', 'size', 'range')",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "icon",
				Description: "Emoji or icon (optional)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "color",
				Description: "Hex color code (optional, e.g., #FF5733)",
				Required:    false,
			},
		},
	},
	{
		Name:        "admin-tag-list",
		Description: "List all tags (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "category",
				Description: "Filter by category (optional)",
				Required:    false,
			},
		},
	},
	{
		Name:        "admin-tag-delete",
		Description: "Delete a tag (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Tag name to delete",
				Required:    true,
			},
		},
	},

	// Admin Commands - System
	{
		Name:        "admin-expire",
		Description: "Manually trigger order expiry check (admin only)",
	},
	{
		Name:        "admin-purge",
		Description: "Remove all orders for a port (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "port",
				Description: "Port name to purge",
				Required:    true,
			},
		},
	},

	// Configuration Commands
	{
		Name:        "config-set-admin-role",
		Description: "Set the admin role for this server (requires Manage Server permission)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionRole,
				Name:        "role",
				Description: "The role that will have admin permissions",
				Required:    true,
			},
		},
		DefaultMemberPermissions: &adminPermission,
	},
	{
		Name:        "config-show",
		Description: "Show current server configuration",
	},

	// Player Trading Commands
	{
		Name:        "trade-set-name",
		Description: "Set your in-game name for trading",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Your in-game character name",
				Required:    true,
			},
		},
	},
	{
		Name:        "trade-create",
		Description: "Create a buy or sell order",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Order type",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Buy (I want to buy)", Value: "buy"},
					{Name: "Sell (I want to sell)", Value: "sell"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name (fuzzy match supported)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "price",
				Description: "Price per unit in gold",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Number of units",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "duration",
				Description: "How long the order stays active",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "1 Day", Value: "1d"},
					{Name: "3 Days", Value: "3d"},
					{Name: "7 Days", Value: "7d"},
					{Name: "14 Days", Value: "14d"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "port",
				Description: "Port name (optional, fuzzy match)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "notes",
				Description: "Additional notes (optional)",
				Required:    false,
			},
		},
	},
	{
		Name:        "trade-search",
		Description: "Search player trade orders",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to search for",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Filter by order type",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Buy Orders", Value: "buy"},
					{Name: "Sell Orders", Value: "sell"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "port",
				Description: "Filter by port",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "min-price",
				Description: "Minimum price filter",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "max-price",
				Description: "Maximum price filter",
				Required:    false,
			},
		},
	},
	{
		Name:        "trade-my-orders",
		Description: "View your active trade orders",
	},
	{
		Name:        "trade-cancel",
		Description: "Cancel one of your trade orders",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "order-id",
				Description: "The order ID to cancel",
				Required:    true,
			},
		},
	},
	{
		Name:        "trade-contact",
		Description: "Contact the creator of a trade order via DM",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "order-id",
				Description: "The order ID you want to discuss",
				Required:    true,
			},
		},
	},
	{
		Name:        "trade-end",
		Description: "End your active trade conversation",
	},
	{
		Name:        "trade-report",
		Description: "Report a trader for misconduct",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "order-id",
				Description: "The order ID of the trader you want to report",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Why you are reporting this trader",
				Required:    true,
			},
		},
	},

	// Admin Commands - Trade Moderation
	{
		Name:        "admin-trade-ban",
		Description: "Ban a user from trading (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user to ban from trading",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Reason for the ban",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "duration",
				Description: "Ban duration (optional, omit for permanent)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "1 Day", Value: "1d"},
					{Name: "3 Days", Value: "3d"},
					{Name: "7 Days", Value: "7d"},
					{Name: "14 Days", Value: "14d"},
					{Name: "30 Days", Value: "30d"},
					{Name: "Permanent", Value: "permanent"},
				},
			},
		},
	},
	{
		Name:        "admin-trade-unban",
		Description: "Remove a trade ban from a user (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user to unban",
				Required:    true,
			},
		},
	},
	{
		Name:        "admin-trade-bans",
		Description: "List all active trade bans (admin only)",
	},
	{
		Name:        "admin-trade-reports",
		Description: "View trade reports (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "status",
				Description: "Filter by status (default: pending)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Pending", Value: "pending"},
					{Name: "Reviewed", Value: "reviewed"},
					{Name: "Dismissed", Value: "dismissed"},
				},
			},
		},
	},
	{
		Name:        "admin-trade-report-action",
		Description: "Take action on a trade report (admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "report-id",
				Description: "The report ID to act on",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "action",
				Description: "Action to take",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Dismiss report", Value: "dismiss"},
					{Name: "Ban reported user", Value: "ban"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Reason (used as ban reason if banning, optional for dismiss)",
				Required:    false,
			},
		},
	},
}

// registerCommands registers all slash commands with Discord
func (b *Bot) registerCommands() error {
	log.Println("Registering slash commands...")

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			return err
		}
		log.Printf("Registered command: %s", cmd.Name)
	}

	return nil
}

// cleanupCommands removes all registered commands (useful for development)
func (b *Bot) cleanupCommands() error {
	log.Println("Cleaning up slash commands...")

	registeredCommands, err := b.session.ApplicationCommands(b.session.State.User.ID, "")
	if err != nil {
		return err
	}

	for _, cmd := range registeredCommands {
		err := b.session.ApplicationCommandDelete(b.session.State.User.ID, "", cmd.ID)
		if err != nil {
			log.Printf("Failed to delete command %s: %v", cmd.Name, err)
		}
	}

	return nil
}
