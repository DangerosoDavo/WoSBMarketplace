package bot

import (
	"sync"
	"time"
)

// ActiveConversation tracks an in-memory active trade conversation
type ActiveConversation struct {
	ConversationID      int
	OrderID             int
	InitiatorUserID     string
	InitiatorIngameName string
	CreatorUserID       string
	CreatorIngameName   string
	LastActivity        time.Time
}

// GetOtherParty returns the other participant's user ID and in-game name
func (ac *ActiveConversation) GetOtherParty(userID string) (otherUserID, otherIngameName string) {
	if userID == ac.InitiatorUserID {
		return ac.CreatorUserID, ac.CreatorIngameName
	}
	return ac.InitiatorUserID, ac.InitiatorIngameName
}

// GetIngameName returns the in-game name of the given user in this conversation
func (ac *ActiveConversation) GetIngameName(userID string) string {
	if userID == ac.InitiatorUserID {
		return ac.InitiatorIngameName
	}
	return ac.CreatorIngameName
}

// TradeConversationManager manages active trade conversations in memory
type TradeConversationManager struct {
	mu            sync.RWMutex
	conversations map[string]*ActiveConversation // userID -> conversation (both parties have entries)
	timeout       time.Duration
}

// NewTradeConversationManager creates a new manager with the given inactivity timeout
func NewTradeConversationManager(timeout time.Duration) *TradeConversationManager {
	tcm := &TradeConversationManager{
		conversations: make(map[string]*ActiveConversation),
		timeout:       timeout,
	}
	go tcm.cleanupLoop()
	return tcm
}

// TryRegister atomically checks that neither party is in a conversation, then registers both.
// Returns false if either party already has an active conversation.
func (tcm *TradeConversationManager) TryRegister(conv *ActiveConversation) bool {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	// Check neither party is already in an active (non-timed-out) conversation
	now := time.Now()
	if existing, ok := tcm.conversations[conv.InitiatorUserID]; ok {
		if now.Sub(existing.LastActivity) <= tcm.timeout {
			return false
		}
	}
	if existing, ok := tcm.conversations[conv.CreatorUserID]; ok {
		if now.Sub(existing.LastActivity) <= tcm.timeout {
			return false
		}
	}

	conv.LastActivity = now
	tcm.conversations[conv.InitiatorUserID] = conv
	tcm.conversations[conv.CreatorUserID] = conv
	return true
}

// Register adds both participants (used for recovery on restart, skips conflict check)
func (tcm *TradeConversationManager) Register(conv *ActiveConversation) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()
	conv.LastActivity = time.Now()
	tcm.conversations[conv.InitiatorUserID] = conv
	tcm.conversations[conv.CreatorUserID] = conv
}

// GetByUser retrieves the active conversation for a user
func (tcm *TradeConversationManager) GetByUser(userID string) (*ActiveConversation, bool) {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()
	conv, ok := tcm.conversations[userID]
	if !ok {
		return nil, false
	}
	if time.Since(conv.LastActivity) > tcm.timeout {
		return nil, false
	}
	return conv, true
}

// Touch updates the last activity timestamp
func (tcm *TradeConversationManager) Touch(userID string) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()
	if conv, ok := tcm.conversations[userID]; ok {
		conv.LastActivity = time.Now()
	}
}

// Remove removes both participants from the in-memory lookup
func (tcm *TradeConversationManager) Remove(conv *ActiveConversation) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()
	// Only remove if the entries still point to this conversation
	if existing, ok := tcm.conversations[conv.InitiatorUserID]; ok && existing.ConversationID == conv.ConversationID {
		delete(tcm.conversations, conv.InitiatorUserID)
	}
	if existing, ok := tcm.conversations[conv.CreatorUserID]; ok && existing.ConversationID == conv.ConversationID {
		delete(tcm.conversations, conv.CreatorUserID)
	}
}

// HasActiveConversation checks if a user is in any active conversation
func (tcm *TradeConversationManager) HasActiveConversation(userID string) bool {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()
	conv, ok := tcm.conversations[userID]
	if !ok {
		return false
	}
	return time.Since(conv.LastActivity) <= tcm.timeout
}

// cleanupLoop periodically removes timed-out conversations
func (tcm *TradeConversationManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		tcm.mu.Lock()
		now := time.Now()
		for userID, conv := range tcm.conversations {
			if now.Sub(conv.LastActivity) > tcm.timeout {
				delete(tcm.conversations, userID)
			}
		}
		tcm.mu.Unlock()
	}
}
