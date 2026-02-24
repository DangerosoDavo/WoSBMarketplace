package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// --- Player Profile Operations ---

// GetPlayerProfile retrieves a player's profile by Discord user ID
func (db *DB) GetPlayerProfile(ctx context.Context, userID string) (*PlayerProfile, error) {
	query := `SELECT user_id, ingame_name, created_at, updated_at FROM player_profiles WHERE user_id = ?`

	var profile PlayerProfile
	err := db.conn.QueryRowContext(ctx, query, userID).Scan(
		&profile.UserID, &profile.IngameName, &profile.CreatedAt, &profile.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player profile: %w", err)
	}
	return &profile, nil
}

// SetPlayerProfile creates or updates a player's in-game name
func (db *DB) SetPlayerProfile(ctx context.Context, userID, ingameName string) error {
	query := `
		INSERT INTO player_profiles (user_id, ingame_name)
		VALUES (?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			ingame_name = excluded.ingame_name,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.conn.ExecContext(ctx, query, userID, ingameName)
	if err != nil {
		return fmt.Errorf("failed to set player profile: %w", err)
	}
	return nil
}

// --- Player Order Operations ---

// CreatePlayerOrder inserts a new player trade order
func (db *DB) CreatePlayerOrder(ctx context.Context, order PlayerOrder) (*PlayerOrder, error) {
	query := `
		INSERT INTO player_orders (user_id, item_id, order_type, price, quantity, port_id, notes, ingame_name, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.conn.ExecContext(ctx, query,
		order.UserID, order.ItemID, order.OrderType, order.Price, order.Quantity,
		order.PortID, order.Notes, order.IngameName, order.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create player order: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get order ID: %w", err)
	}

	order.ID = int(id)
	order.Status = "active"
	order.CreatedAt = time.Now()
	return &order, nil
}

// GetPlayerOrder retrieves a single order by ID (with item/port joins)
func (db *DB) GetPlayerOrder(ctx context.Context, orderID int) (*PlayerOrder, error) {
	query := `
		SELECT po.id, po.user_id, po.item_id, po.order_type, po.price, po.quantity,
		       po.port_id, po.notes, po.ingame_name, po.status, po.created_at, po.expires_at,
		       i.name, i.display_name,
		       p.name, p.display_name, p.region
		FROM player_orders po
		JOIN items i ON po.item_id = i.id
		LEFT JOIN ports p ON po.port_id = p.id
		WHERE po.id = ? AND po.status = 'active' AND po.expires_at > datetime('now')
	`

	var po PlayerOrder
	var portID sql.NullInt64
	var notes sql.NullString
	var itemName, itemDisplay string
	var portName, portDisplay, portRegion sql.NullString

	err := db.conn.QueryRowContext(ctx, query, orderID).Scan(
		&po.ID, &po.UserID, &po.ItemID, &po.OrderType, &po.Price, &po.Quantity,
		&portID, &notes, &po.IngameName, &po.Status, &po.CreatedAt, &po.ExpiresAt,
		&itemName, &itemDisplay,
		&portName, &portDisplay, &portRegion,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player order: %w", err)
	}

	po.Item = &Item{ID: po.ItemID, Name: itemName, DisplayName: itemDisplay}
	if portID.Valid {
		id := int(portID.Int64)
		po.PortID = &id
		po.Port = &Port{ID: id, Name: portName.String, DisplayName: portDisplay.String, Region: portRegion.String}
	}
	if notes.Valid {
		po.Notes = notes.String
	}
	return &po, nil
}

// GetPlayerOrdersByUser retrieves all active orders for a specific user
func (db *DB) GetPlayerOrdersByUser(ctx context.Context, userID string) ([]PlayerOrder, error) {
	query := `
		SELECT po.id, po.user_id, po.item_id, po.order_type, po.price, po.quantity,
		       po.port_id, po.notes, po.ingame_name, po.status, po.created_at, po.expires_at,
		       i.name, i.display_name,
		       p.name, p.display_name, p.region
		FROM player_orders po
		JOIN items i ON po.item_id = i.id
		LEFT JOIN ports p ON po.port_id = p.id
		WHERE po.user_id = ? AND po.status = 'active' AND po.expires_at > datetime('now')
		ORDER BY po.created_at DESC
	`
	rows, err := db.conn.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}
	defer rows.Close()
	return scanPlayerOrdersWithJoins(rows)
}

// SearchPlayerOrders searches orders with optional filters
func (db *DB) SearchPlayerOrders(ctx context.Context, itemID int, orderType string, portID int, minPrice int, maxPrice int, limit int) ([]PlayerOrder, error) {
	query := `
		SELECT po.id, po.user_id, po.item_id, po.order_type, po.price, po.quantity,
		       po.port_id, po.notes, po.ingame_name, po.status, po.created_at, po.expires_at,
		       i.name, i.display_name,
		       p.name, p.display_name, p.region
		FROM player_orders po
		JOIN items i ON po.item_id = i.id
		LEFT JOIN ports p ON po.port_id = p.id
		WHERE po.status = 'active' AND po.expires_at > datetime('now')
	`
	args := []interface{}{}

	if itemID > 0 {
		query += ` AND po.item_id = ?`
		args = append(args, itemID)
	}
	if orderType != "" {
		query += ` AND po.order_type = ?`
		args = append(args, orderType)
	}
	if portID > 0 {
		query += ` AND po.port_id = ?`
		args = append(args, portID)
	}
	if minPrice > 0 {
		query += ` AND po.price >= ?`
		args = append(args, minPrice)
	}
	if maxPrice > 0 {
		query += ` AND po.price <= ?`
		args = append(args, maxPrice)
	}

	query += ` ORDER BY po.created_at DESC`
	if limit <= 0 {
		limit = 25
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search player orders: %w", err)
	}
	defer rows.Close()
	return scanPlayerOrdersWithJoins(rows)
}

// CancelPlayerOrder sets an order's status to "cancelled" (only owner can cancel)
func (db *DB) CancelPlayerOrder(ctx context.Context, orderID int, userID string) error {
	query := `UPDATE player_orders SET status = 'cancelled' WHERE id = ? AND user_id = ? AND status = 'active'`
	result, err := db.conn.ExecContext(ctx, query, orderID, userID)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("order not found or not owned by you")
	}
	return nil
}

// CompletePlayerOrder sets an order's status to "completed"
func (db *DB) CompletePlayerOrder(ctx context.Context, orderID int, userID string) error {
	query := `UPDATE player_orders SET status = 'completed' WHERE id = ? AND user_id = ? AND status = 'active'`
	_, err := db.conn.ExecContext(ctx, query, orderID, userID)
	if err != nil {
		return fmt.Errorf("failed to complete order: %w", err)
	}
	return nil
}

// DeleteExpiredPlayerOrders removes expired player orders
func (db *DB) DeleteExpiredPlayerOrders(ctx context.Context) (int64, error) {
	query := `UPDATE player_orders SET status = 'cancelled' WHERE status = 'active' AND expires_at <= datetime('now')`
	result, err := db.conn.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to expire player orders: %w", err)
	}
	return result.RowsAffected()
}

// --- Trade Conversation Operations ---

// CreateTradeConversation starts a new trade conversation
func (db *DB) CreateTradeConversation(ctx context.Context, conv TradeConversation) (*TradeConversation, error) {
	query := `
		INSERT INTO trade_conversations (order_id, initiator_user_id, initiator_ingame_name, creator_user_id, creator_ingame_name)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := db.conn.ExecContext(ctx, query,
		conv.OrderID, conv.InitiatorUserID, conv.InitiatorIngameName,
		conv.CreatorUserID, conv.CreatorIngameName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trade conversation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation ID: %w", err)
	}

	conv.ID = int(id)
	conv.Status = "active"
	conv.StartedAt = time.Now()
	conv.LastMessageAt = time.Now()
	return &conv, nil
}

// GetActiveConversationByUser finds an active conversation for a user (as either party)
func (db *DB) GetActiveConversationByUser(ctx context.Context, userID string) (*TradeConversation, error) {
	query := `
		SELECT id, order_id, initiator_user_id, initiator_ingame_name,
		       creator_user_id, creator_ingame_name, status, started_at,
		       ended_at, last_message_at
		FROM trade_conversations
		WHERE status = 'active'
		  AND (initiator_user_id = ? OR creator_user_id = ?)
		ORDER BY started_at DESC
		LIMIT 1
	`
	var conv TradeConversation
	var endedAt sql.NullTime

	err := db.conn.QueryRowContext(ctx, query, userID, userID).Scan(
		&conv.ID, &conv.OrderID, &conv.InitiatorUserID, &conv.InitiatorIngameName,
		&conv.CreatorUserID, &conv.CreatorIngameName, &conv.Status, &conv.StartedAt,
		&endedAt, &conv.LastMessageAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active conversation: %w", err)
	}
	if endedAt.Valid {
		conv.EndedAt = &endedAt.Time
	}
	return &conv, nil
}

// CloseTradeConversation ends a conversation
func (db *DB) CloseTradeConversation(ctx context.Context, convID int) error {
	query := `UPDATE trade_conversations SET status = 'closed', ended_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, convID)
	if err != nil {
		return fmt.Errorf("failed to close conversation: %w", err)
	}
	return nil
}

// UpdateConversationActivity updates the last_message_at timestamp
func (db *DB) UpdateConversationActivity(ctx context.Context, convID int) error {
	query := `UPDATE trade_conversations SET last_message_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, convID)
	if err != nil {
		return fmt.Errorf("failed to update conversation activity: %w", err)
	}
	return nil
}

// GetStaleConversations finds conversations inactive for a given duration
func (db *DB) GetStaleConversations(ctx context.Context, inactiveDuration time.Duration) ([]TradeConversation, error) {
	cutoff := time.Now().Add(-inactiveDuration)
	query := `
		SELECT id, order_id, initiator_user_id, initiator_ingame_name,
		       creator_user_id, creator_ingame_name, status, started_at,
		       ended_at, last_message_at
		FROM trade_conversations
		WHERE status = 'active' AND last_message_at < ?
	`
	rows, err := db.conn.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale conversations: %w", err)
	}
	defer rows.Close()
	return scanTradeConversations(rows)
}

// GetAllActiveConversations returns all conversations with status 'active' (for recovery on restart)
func (db *DB) GetAllActiveConversations(ctx context.Context) ([]TradeConversation, error) {
	query := `
		SELECT id, order_id, initiator_user_id, initiator_ingame_name,
		       creator_user_id, creator_ingame_name, status, started_at,
		       ended_at, last_message_at
		FROM trade_conversations
		WHERE status = 'active'
	`
	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active conversations: %w", err)
	}
	defer rows.Close()
	return scanTradeConversations(rows)
}

// --- Helpers ---

func scanPlayerOrdersWithJoins(rows *sql.Rows) ([]PlayerOrder, error) {
	var orders []PlayerOrder
	for rows.Next() {
		var po PlayerOrder
		var portID sql.NullInt64
		var notes sql.NullString
		var itemName, itemDisplay string
		var portName, portDisplay, portRegion sql.NullString

		err := rows.Scan(
			&po.ID, &po.UserID, &po.ItemID, &po.OrderType, &po.Price, &po.Quantity,
			&portID, &notes, &po.IngameName, &po.Status, &po.CreatedAt, &po.ExpiresAt,
			&itemName, &itemDisplay,
			&portName, &portDisplay, &portRegion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan player order: %w", err)
		}

		po.Item = &Item{ID: po.ItemID, Name: itemName, DisplayName: itemDisplay}
		if portID.Valid {
			id := int(portID.Int64)
			po.PortID = &id
			po.Port = &Port{ID: id, Name: portName.String, DisplayName: portDisplay.String, Region: portRegion.String}
		}
		if notes.Valid {
			po.Notes = notes.String
		}
		orders = append(orders, po)
	}
	return orders, rows.Err()
}

func scanTradeConversations(rows *sql.Rows) ([]TradeConversation, error) {
	var convs []TradeConversation
	for rows.Next() {
		var conv TradeConversation
		var endedAt sql.NullTime

		err := rows.Scan(
			&conv.ID, &conv.OrderID, &conv.InitiatorUserID, &conv.InitiatorIngameName,
			&conv.CreatorUserID, &conv.CreatorIngameName, &conv.Status, &conv.StartedAt,
			&endedAt, &conv.LastMessageAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade conversation: %w", err)
		}
		if endedAt.Valid {
			conv.EndedAt = &endedAt.Time
		}
		convs = append(convs, conv)
	}
	return convs, rows.Err()
}
