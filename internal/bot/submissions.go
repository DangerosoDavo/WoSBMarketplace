package bot

import (
	"sync"
	"time"
	"wosbTrade/internal/database"
	"wosbTrade/internal/ocr"
)

// PendingSubmission represents a submission awaiting user confirmation
type PendingSubmission struct {
	UserID          string
	ChannelID       string
	InteractionID   string
	ImagePath       string
	OCRResult       *ocr.MarketData
	CreatedAt       time.Time
	ExpiresAt       time.Time
	ScreenshotHash  string
	OrderType       string

	// Port confirmation state
	PortConfirmed   bool
	PortID          *int

	// Item mapping: OCR name -> confirmed item_id
	// This ensures we only ask once per unique item name
	ItemMappings    map[string]int
	ItemsConfirmed  bool
}

// SubmissionManager manages pending submissions
type SubmissionManager struct {
	mu          sync.RWMutex
	submissions map[string]*PendingSubmission // userID -> submission
	timeout     time.Duration
}

// NewSubmissionManager creates a new submission manager
func NewSubmissionManager(timeout time.Duration) *SubmissionManager {
	sm := &SubmissionManager{
		submissions: make(map[string]*PendingSubmission),
		timeout:     timeout,
	}

	// Start cleanup goroutine
	go sm.cleanupLoop()

	return sm
}

// Create creates a new pending submission
func (sm *SubmissionManager) Create(userID, channelID, interactionID, imagePath, screenshotHash, orderType string, ocrResult *ocr.MarketData) *PendingSubmission {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	sub := &PendingSubmission{
		UserID:         userID,
		ChannelID:      channelID,
		InteractionID:  interactionID,
		ImagePath:      imagePath,
		OCRResult:      ocrResult,
		CreatedAt:      now,
		ExpiresAt:      now.Add(sm.timeout),
		ScreenshotHash: screenshotHash,
		OrderType:      orderType,
		PortConfirmed:  false,
		ItemsConfirmed: false,
		ItemMappings:   make(map[string]int),
	}

	sm.submissions[userID] = sub
	return sub
}

// Get retrieves a pending submission
func (sm *SubmissionManager) Get(userID string) (*PendingSubmission, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sub, ok := sm.submissions[userID]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Now().After(sub.ExpiresAt) {
		return nil, false
	}

	return sub, true
}

// Remove removes a pending submission
func (sm *SubmissionManager) Remove(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.submissions, userID)
}

// ConfirmPort confirms the port for a submission
func (sm *SubmissionManager) ConfirmPort(userID string, portID int) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, ok := sm.submissions[userID]
	if !ok {
		return false
	}

	sub.PortID = &portID
	sub.PortConfirmed = true
	return true
}

// AddItemMapping adds an item mapping (OCR name -> item_id)
// Returns true if this is a new mapping (first time seeing this OCR name)
func (sm *SubmissionManager) AddItemMapping(userID, ocrName string, itemID int) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, ok := sm.submissions[userID]
	if !ok {
		return false
	}

	// Check if we've already mapped this OCR name
	if _, exists := sub.ItemMappings[ocrName]; exists {
		return false // Already mapped
	}

	sub.ItemMappings[ocrName] = itemID
	return true // New mapping
}

// GetItemMapping gets the mapped item ID for an OCR name
func (sm *SubmissionManager) GetItemMapping(userID, ocrName string) (int, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sub, ok := sm.submissions[userID]
	if !ok {
		return 0, false
	}

	itemID, ok := sub.ItemMappings[ocrName]
	return itemID, ok
}

// MarkItemsConfirmed marks all items as confirmed
func (sm *SubmissionManager) MarkItemsConfirmed(userID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, ok := sm.submissions[userID]
	if !ok {
		return false
	}

	sub.ItemsConfirmed = true
	return true
}

// IsReady returns true if submission is ready to be committed
func (sm *SubmissionManager) IsReady(userID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sub, ok := sm.submissions[userID]
	if !ok {
		return false
	}

	return sub.PortConfirmed && sub.ItemsConfirmed
}

// GetMarketOrders builds the final market orders from a pending submission
func (sm *SubmissionManager) GetMarketOrders(userID string) ([]database.Market, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sub, ok := sm.submissions[userID]
	if !ok || !sub.PortConfirmed || !sub.ItemsConfirmed {
		return nil, nil
	}

	var orders []database.Market
	for _, ocrItem := range sub.OCRResult.Items {
		itemID, ok := sub.ItemMappings[ocrItem.Name]
		if !ok {
			// This shouldn't happen if items are confirmed
			continue
		}

		orders = append(orders, database.Market{
			ItemID:   itemID,
			Price:    ocrItem.Price,
			Quantity: ocrItem.Quantity,
		})
	}

	return orders, nil
}

// cleanupLoop periodically removes expired submissions
func (sm *SubmissionManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanup()
	}
}

func (sm *SubmissionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for userID, sub := range sm.submissions {
		if now.After(sub.ExpiresAt) {
			// TODO: Notify user that submission expired
			// TODO: Clean up temp image file
			delete(sm.submissions, userID)
		}
	}
}

// GetUniqueOCRItems returns unique item names from OCR result
// This is used to avoid asking the user to confirm duplicates
func (sub *PendingSubmission) GetUniqueOCRItems() []ocr.MarketItem {
	seen := make(map[string]bool)
	var unique []ocr.MarketItem

	for _, item := range sub.OCRResult.Items {
		if !seen[item.Name] {
			seen[item.Name] = true
			unique = append(unique, item)
		}
	}

	return unique
}

// GetUnconfirmedItems returns items that haven't been mapped yet
func (sub *PendingSubmission) GetUnconfirmedItems() []string {
	uniqueItems := sub.GetUniqueOCRItems()
	var unconfirmed []string

	for _, item := range uniqueItems {
		if _, ok := sub.ItemMappings[item.Name]; !ok {
			unconfirmed = append(unconfirmed, item.Name)
		}
	}

	return unconfirmed
}

// IsComplete returns true if all unique items have been mapped
func (sub *PendingSubmission) IsComplete() bool {
	uniqueItems := sub.GetUniqueOCRItems()
	return len(sub.ItemMappings) == len(uniqueItems)
}
