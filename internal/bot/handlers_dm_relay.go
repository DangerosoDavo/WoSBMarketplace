package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// messageCreate handles incoming messages, specifically DMs for trade relay
func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore the bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore guild messages - only handle DMs
	if m.GuildID != "" {
		return
	}

	// Look up sender in active conversations
	conv, ok := b.tradeConversations.GetByUser(m.Author.ID)
	if !ok {
		// No active conversation - send help message
		s.ChannelMessageSend(m.ChannelID,
			"You don't have an active trade conversation.\n\n"+
				"Use `/trade-search` in a server to find orders, then `/trade-contact` to start chatting with a trader.")
		return
	}

	// Get the other party's info
	otherUserID, _ := conv.GetOtherParty(m.Author.ID)
	senderIngameName := conv.GetIngameName(m.Author.ID)

	// Open a DM channel to the other party
	otherCh, err := s.UserChannelCreate(otherUserID)
	if err != nil {
		log.Printf("Error creating DM channel to %s: %v", otherUserID, err)
		s.ChannelMessageSend(m.ChannelID, "Failed to deliver your message. The other trader may have DMs disabled.")
		return
	}

	// Relay the text message
	if m.Content != "" {
		relayMsg := fmt.Sprintf("**[%s]**: %s", senderIngameName, m.Content)
		_, err := s.ChannelMessageSend(otherCh.ID, relayMsg)
		if err != nil {
			log.Printf("Error relaying message to %s: %v", otherUserID, err)
			s.ChannelMessageSend(m.ChannelID, "Failed to deliver your message. The other trader may have DMs disabled.")
			return
		}
	}

	// Forward attachment URLs
	if len(m.Attachments) > 0 {
		var attachmentLines []string
		for _, att := range m.Attachments {
			attachmentLines = append(attachmentLines, att.URL)
		}
		attachMsg := fmt.Sprintf("**[%s]** shared:\n%s", senderIngameName, strings.Join(attachmentLines, "\n"))
		_, err := s.ChannelMessageSend(otherCh.ID, attachMsg)
		if err != nil {
			log.Printf("Error relaying attachments to %s: %v", otherUserID, err)
		}
	}

	// Add checkmark reaction to confirm delivery
	s.MessageReactionAdd(m.ChannelID, m.ID, "✅")

	// Update activity timestamp (memory + DB)
	b.tradeConversations.Touch(m.Author.ID)
	ctx := context.Background()
	if err := b.db.UpdateConversationActivity(ctx, conv.ConversationID); err != nil {
		log.Printf("Error updating conversation activity: %v", err)
	}
}
