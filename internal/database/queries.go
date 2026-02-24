package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ReplacePortOrders replaces all orders for a given port and order type
// This is atomic - deletes old orders and inserts new ones in a transaction
func (db *DB) ReplacePortOrders(ctx context.Context, portID int, orderType string, orders []Market, submittedBy, screenshotHash string) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing orders for this port and order type
	deleteQuery := `DELETE FROM markets WHERE port_id = ? AND order_type = ?`
	result, err := tx.ExecContext(ctx, deleteQuery, portID, orderType)
	if err != nil {
		return fmt.Errorf("failed to delete old orders: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()

	// Insert new orders
	insertQuery := `
		INSERT INTO markets (port_id, item_id, order_type, price, quantity, submitted_by, expires_at, screenshot_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	expiresAt := time.Now().AddDate(0, 0, 7) // 7 days from now

	for _, order := range orders {
		_, err := tx.ExecContext(ctx, insertQuery,
			portID,
			order.ItemID,
			orderType,
			order.Price,
			order.Quantity,
			submittedBy,
			expiresAt,
			screenshotHash,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order for item_id %d: %w", order.ItemID, err)
		}
	}

	// Log the action
	auditQuery := `
		INSERT INTO audit_log (action, user_id, details)
		VALUES (?, ?, ?)
	`
	details := fmt.Sprintf(`{"port_id":%d,"order_type":"%s","deleted":%d,"inserted":%d}`,
		portID, orderType, rowsDeleted, len(orders))

	_, err = tx.ExecContext(ctx, auditQuery, "replace_orders", submittedBy, details)
	if err != nil {
		return fmt.Errorf("failed to log action: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetPricesByItem returns best buy and sell prices for an item across all ports
func (db *DB) GetPricesByItem(ctx context.Context, itemID int, tagIDs []int, region string, minPrice, maxPrice int) ([]Market, error) {
	query := `
		SELECT m.id, m.port_id, m.item_id, m.order_type, m.price, m.quantity,
		       m.submitted_by, m.submitted_at, m.expires_at, m.screenshot_hash,
		       p.name as port_name, p.display_name as port_display, p.region,
		       i.name as item_name, i.display_name as item_display
		FROM markets m
		JOIN ports p ON m.port_id = p.id
		JOIN items i ON m.item_id = i.id
		WHERE m.item_id = ?
		  AND m.expires_at > datetime('now')
	`
	args := []interface{}{itemID}

	// Add region filter
	if region != "" {
		query += ` AND p.region = ?`
		args = append(args, region)
	}

	// Add price range filter
	if minPrice > 0 {
		query += ` AND m.price >= ?`
		args = append(args, minPrice)
	}
	if maxPrice > 0 {
		query += ` AND m.price <= ?`
		args = append(args, maxPrice)
	}

	query += ` ORDER BY m.order_type, m.price ASC LIMIT 20`

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query prices: %w", err)
	}
	defer rows.Close()

	return scanMarketsWithJoins(rows)
}

// GetOrdersByPort returns all active orders for a specific port
func (db *DB) GetOrdersByPort(ctx context.Context, portID int) ([]Market, error) {
	query := `
		SELECT m.id, m.port_id, m.item_id, m.order_type, m.price, m.quantity,
		       m.submitted_by, m.submitted_at, m.expires_at, m.screenshot_hash,
		       p.name as port_name, p.display_name as port_display, p.region,
		       i.name as item_name, i.display_name as item_display
		FROM markets m
		JOIN ports p ON m.port_id = p.id
		JOIN items i ON m.item_id = i.id
		WHERE m.port_id = ? AND m.expires_at > datetime('now')
		ORDER BY m.order_type, i.name ASC
	`

	rows, err := db.conn.QueryContext(ctx, query, portID)
	if err != nil {
		return nil, fmt.Errorf("failed to query port orders: %w", err)
	}
	defer rows.Close()

	return scanMarketsWithJoins(rows)
}

// GetOrdersByTags returns orders for items with specified tags
func (db *DB) GetOrdersByTags(ctx context.Context, tagIDs []int, region string) ([]Market, error) {
	if len(tagIDs) == 0 {
		return nil, fmt.Errorf("no tags specified")
	}

	// Build query with tag filters
	query := `
		SELECT DISTINCT m.id, m.port_id, m.item_id, m.order_type, m.price, m.quantity,
		       m.submitted_by, m.submitted_at, m.expires_at, m.screenshot_hash,
		       p.name as port_name, p.display_name as port_display, p.region,
		       i.name as item_name, i.display_name as item_display
		FROM markets m
		JOIN ports p ON m.port_id = p.id
		JOIN items i ON m.item_id = i.id
		JOIN item_tags it ON i.id = it.item_id
		WHERE it.tag_id IN (?` + repeatPlaceholders(len(tagIDs)-1) + `)
		  AND m.expires_at > datetime('now')
	`

	args := make([]interface{}, len(tagIDs))
	for i, id := range tagIDs {
		args[i] = id
	}

	if region != "" {
		query += ` AND p.region = ?`
		args = append(args, region)
	}

	query += ` ORDER BY m.order_type, m.price ASC LIMIT 50`

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query by tags: %w", err)
	}
	defer rows.Close()

	return scanMarketsWithJoins(rows)
}

// DeleteExpiredOrders removes all orders past their expiry date
func (db *DB) DeleteExpiredOrders(ctx context.Context) (int64, error) {
	query := `DELETE FROM markets WHERE expires_at <= datetime('now')`

	result, err := db.conn.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired orders: %w", err)
	}

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the expiry
	if rowsDeleted > 0 {
		auditQuery := `
			INSERT INTO audit_log (action, user_id, details)
			VALUES (?, ?, ?)
		`
		details := fmt.Sprintf(`{"expired_count":%d}`, rowsDeleted)
		_, _ = db.conn.ExecContext(ctx, auditQuery, "expire_orders", "system", details)
	}

	return rowsDeleted, nil
}

// PurgePort removes all orders for a specific port
func (db *DB) PurgePort(ctx context.Context, portID int, adminUserID string) (int64, error) {
	query := `DELETE FROM markets WHERE port_id = ?`

	result, err := db.conn.ExecContext(ctx, query, portID)
	if err != nil {
		return 0, fmt.Errorf("failed to purge port: %w", err)
	}

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the purge
	auditQuery := `
		INSERT INTO audit_log (action, user_id, details)
		VALUES (?, ?, ?)
	`
	details := fmt.Sprintf(`{"port_id":%d,"deleted":%d}`, portID, rowsDeleted)
	_, _ = db.conn.ExecContext(ctx, auditQuery, "purge_port", adminUserID, details)

	return rowsDeleted, nil
}

// GetStats returns bot statistics
func (db *DB) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total active orders
	var totalOrders int
	err := db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM markets WHERE expires_at > datetime('now')`).Scan(&totalOrders)
	if err != nil {
		return nil, err
	}
	stats["total_orders"] = totalOrders

	// Unique ports
	var uniquePorts int
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(DISTINCT port_id) FROM markets WHERE expires_at > datetime('now')`).Scan(&uniquePorts)
	if err != nil {
		return nil, err
	}
	stats["unique_ports"] = uniquePorts

	// Untagged items count
	var untaggedItems int
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE is_tagged = FALSE`).Scan(&untaggedItems)
	if err != nil {
		return nil, err
	}
	stats["untagged_items"] = untaggedItems

	// Total items
	var totalItems int
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM items`).Scan(&totalItems)
	if err != nil {
		return nil, err
	}
	stats["total_items"] = totalItems

	// Total ports
	var totalPorts int
	err = db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM ports`).Scan(&totalPorts)
	if err != nil {
		return nil, err
	}
	stats["total_ports"] = totalPorts

	// Last update
	var lastUpdate sql.NullTime
	err = db.conn.QueryRowContext(ctx, `SELECT MAX(submitted_at) FROM markets`).Scan(&lastUpdate)
	if err != nil {
		return nil, err
	}
	if lastUpdate.Valid {
		stats["last_update"] = lastUpdate.Time
	}

	// Total submissions today
	var submissionsToday int
	err = db.conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM audit_log
		WHERE action = 'replace_orders'
		AND timestamp > datetime('now', '-1 day')
	`).Scan(&submissionsToday)
	if err != nil {
		return nil, err
	}
	stats["submissions_today"] = submissionsToday

	return stats, nil
}

// GetUntaggedItems returns all items that need tagging
func (db *DB) GetUntaggedItems(ctx context.Context, limit int) ([]Item, error) {
	query := `
		SELECT id, name, display_name, is_tagged, added_at, added_by, notes
		FROM items
		WHERE is_tagged = FALSE
		ORDER BY added_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(&item.ID, &item.Name, &item.DisplayName, &item.IsTagged,
			&item.AddedAt, &item.AddedBy, &item.Notes)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// AddTagsToItem adds tags to an item and marks it as tagged
func (db *DB) AddTagsToItem(ctx context.Context, itemID int, tagIDs []int) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert item_tags
	for _, tagID := range tagIDs {
		query := `INSERT OR IGNORE INTO item_tags (item_id, tag_id) VALUES (?, ?)`
		_, err := tx.ExecContext(ctx, query, itemID, tagID)
		if err != nil {
			return err
		}
	}

	// Mark item as tagged
	updateQuery := `UPDATE items SET is_tagged = TRUE WHERE id = ?`
	_, err = tx.ExecContext(ctx, updateQuery, itemID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveTagsFromItem removes tags from an item
func (db *DB) RemoveTagsFromItem(ctx context.Context, itemID int, tagIDs []int) error {
	query := `DELETE FROM item_tags WHERE item_id = ? AND tag_id IN (?` + repeatPlaceholders(len(tagIDs)-1) + `)`
	args := []interface{}{itemID}
	for _, tagID := range tagIDs {
		args = append(args, tagID)
	}

	_, err := db.conn.ExecContext(ctx, query, args...)
	return err
}

// GetItemTags returns all tags for an item
func (db *DB) GetItemTags(ctx context.Context, itemID int) ([]Tag, error) {
	query := `
		SELECT t.id, t.name, t.category, t.color, t.icon, t.created_at
		FROM tags t
		JOIN item_tags it ON t.id = it.tag_id
		WHERE it.item_id = ?
		ORDER BY t.category, t.name
	`

	rows, err := db.conn.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.Color, &tag.Icon, &tag.CreatedAt)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// CreateTag creates a new tag
func (db *DB) CreateTag(ctx context.Context, name, category, color, icon string) (*Tag, error) {
	query := `INSERT INTO tags (name, category, color, icon) VALUES (?, ?, ?, ?)`
	result, err := db.conn.ExecContext(ctx, query, name, category, color, icon)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Tag{
		ID:        int(id),
		Name:      name,
		Category:  category,
		Color:     color,
		Icon:      icon,
		CreatedAt: time.Now(),
	}, nil
}

// GetAllTags returns all tags, optionally filtered by category
func (db *DB) GetAllTags(ctx context.Context, category string) ([]Tag, error) {
	query := `SELECT id, name, category, color, icon, created_at FROM tags`
	var args []interface{}

	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}

	query += ` ORDER BY category, name`

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.Color, &tag.Icon, &tag.CreatedAt)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// Helper functions

func scanMarketsWithJoins(rows *sql.Rows) ([]Market, error) {
	var markets []Market

	for rows.Next() {
		var m Market
		var portName, portDisplay, portRegion string
		var itemName, itemDisplay string

		err := rows.Scan(
			&m.ID, &m.PortID, &m.ItemID, &m.OrderType, &m.Price, &m.Quantity,
			&m.SubmittedBy, &m.SubmittedAt, &m.ExpiresAt, &m.ScreenshotHash,
			&portName, &portDisplay, &portRegion,
			&itemName, &itemDisplay,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		m.Port = &Port{
			ID:          m.PortID,
			Name:        portName,
			DisplayName: portDisplay,
			Region:      portRegion,
		}

		m.Item = &Item{
			ID:          m.ItemID,
			Name:        itemName,
			DisplayName: itemDisplay,
		}

		markets = append(markets, m)
	}

	return markets, rows.Err()
}

func repeatPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < count; i++ {
		result += ",?"
	}
	return result
}

// Guild Settings

type GuildSettings struct {
	GuildID       string
	AdminRoleID   string
	ConfiguredAt  time.Time
	ConfiguredBy  string
	UpdatedAt     time.Time
}

// GetGuildSettings retrieves settings for a specific guild
func (db *DB) GetGuildSettings(ctx context.Context, guildID string) (*GuildSettings, error) {
	query := `
		SELECT guild_id, admin_role_id, configured_at, configured_by, updated_at
		FROM guild_settings
		WHERE guild_id = ?
	`

	var settings GuildSettings
	var adminRoleID sql.NullString

	err := db.conn.QueryRowContext(ctx, query, guildID).Scan(
		&settings.GuildID,
		&adminRoleID,
		&settings.ConfiguredAt,
		&settings.ConfiguredBy,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No settings configured yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get guild settings: %w", err)
	}

	if adminRoleID.Valid {
		settings.AdminRoleID = adminRoleID.String
	}

	return &settings, nil
}

// SetGuildAdminRole sets or updates the admin role for a guild
func (db *DB) SetGuildAdminRole(ctx context.Context, guildID, adminRoleID, configuredBy string) error {
	query := `
		INSERT INTO guild_settings (guild_id, admin_role_id, configured_by, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(guild_id) DO UPDATE SET
			admin_role_id = excluded.admin_role_id,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.conn.ExecContext(ctx, query, guildID, adminRoleID, configuredBy)
	if err != nil {
		return fmt.Errorf("failed to set guild admin role: %w", err)
	}

	return nil
}

// GetAllGuildSettings retrieves all configured guilds
func (db *DB) GetAllGuildSettings(ctx context.Context) ([]GuildSettings, error) {
	query := `
		SELECT guild_id, admin_role_id, configured_at, configured_by, updated_at
		FROM guild_settings
		ORDER BY updated_at DESC
	`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query guild settings: %w", err)
	}
	defer rows.Close()

	var settings []GuildSettings
	for rows.Next() {
		var s GuildSettings
		var adminRoleID sql.NullString

		err := rows.Scan(
			&s.GuildID,
			&adminRoleID,
			&s.ConfiguredAt,
			&s.ConfiguredBy,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guild settings: %w", err)
		}

		if adminRoleID.Valid {
			s.AdminRoleID = adminRoleID.String
		}

		settings = append(settings, s)
	}

	return settings, nil
}
