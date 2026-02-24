package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// --- Trade Ban Operations ---

// IsUserBanned checks if a user has an active, non-expired ban.
// Returns nil, nil if the user is not banned.
func (db *DB) IsUserBanned(ctx context.Context, userID string) (*TradeBan, error) {
	query := `
		SELECT id, user_id, reason, banned_by, banned_at, expires_at, active
		FROM trade_bans
		WHERE user_id = ? AND active = TRUE
		  AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY banned_at DESC
		LIMIT 1
	`
	var ban TradeBan
	var expiresAt sql.NullTime

	err := db.conn.QueryRowContext(ctx, query, userID).Scan(
		&ban.ID, &ban.UserID, &ban.Reason, &ban.BannedBy,
		&ban.BannedAt, &expiresAt, &ban.Active,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check trade ban: %w", err)
	}
	if expiresAt.Valid {
		ban.ExpiresAt = &expiresAt.Time
	}
	return &ban, nil
}

// CreateTradeBan inserts a new ban record and logs the action.
func (db *DB) CreateTradeBan(ctx context.Context, ban TradeBan) (*TradeBan, error) {
	query := `INSERT INTO trade_bans (user_id, reason, banned_by, expires_at) VALUES (?, ?, ?, ?)`
	result, err := db.conn.ExecContext(ctx, query, ban.UserID, ban.Reason, ban.BannedBy, ban.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create trade ban: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get ban ID: %w", err)
	}

	ban.ID = int(id)
	ban.Active = true
	ban.BannedAt = time.Now()

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"banned_user": ban.UserID,
		"reason":      ban.Reason,
		"banned_by":   ban.BannedBy,
		"expires_at":  ban.ExpiresAt,
	})
	db.conn.ExecContext(ctx,
		`INSERT INTO audit_log (action, user_id, details) VALUES (?, ?, ?)`,
		"trade_ban", ban.BannedBy, string(details),
	)

	return &ban, nil
}

// RemoveTradeBan deactivates all active bans for a user.
func (db *DB) RemoveTradeBan(ctx context.Context, userID string, unbannedBy string) error {
	query := `UPDATE trade_bans SET active = FALSE WHERE user_id = ? AND active = TRUE`
	result, err := db.conn.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to remove trade ban: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user is not banned from trading")
	}

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"unbanned_user": userID,
		"unbanned_by":   unbannedBy,
	})
	db.conn.ExecContext(ctx,
		`INSERT INTO audit_log (action, user_id, details) VALUES (?, ?, ?)`,
		"trade_unban", unbannedBy, string(details),
	)

	return nil
}

// GetActiveTradeBans returns all currently active, non-expired bans.
func (db *DB) GetActiveTradeBans(ctx context.Context) ([]TradeBan, error) {
	query := `
		SELECT id, user_id, reason, banned_by, banned_at, expires_at, active
		FROM trade_bans
		WHERE active = TRUE
		  AND (expires_at IS NULL OR expires_at > datetime('now'))
		ORDER BY banned_at DESC
	`
	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active trade bans: %w", err)
	}
	defer rows.Close()
	return scanTradeBans(rows)
}

// CancelAllUserOrders cancels all active player orders for a user.
func (db *DB) CancelAllUserOrders(ctx context.Context, userID string) (int64, error) {
	query := `UPDATE player_orders SET status = 'cancelled' WHERE user_id = ? AND status = 'active'`
	result, err := db.conn.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to cancel user orders: %w", err)
	}
	return result.RowsAffected()
}

// --- Trade Report Operations ---

// CreateTradeReport inserts a new report and logs the action.
func (db *DB) CreateTradeReport(ctx context.Context, report TradeReport) (*TradeReport, error) {
	query := `INSERT INTO trade_reports (reporter_user_id, reported_user_id, order_id, reason) VALUES (?, ?, ?, ?)`
	result, err := db.conn.ExecContext(ctx, query,
		report.ReporterUserID, report.ReportedUserID, report.OrderID, report.Reason,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trade report: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get report ID: %w", err)
	}

	report.ID = int(id)
	report.Status = "pending"
	report.CreatedAt = time.Now()

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"reporter":  report.ReporterUserID,
		"reported":  report.ReportedUserID,
		"order_id":  report.OrderID,
		"reason":    report.Reason,
	})
	db.conn.ExecContext(ctx,
		`INSERT INTO audit_log (action, user_id, details) VALUES (?, ?, ?)`,
		"trade_report", report.ReporterUserID, string(details),
	)

	return &report, nil
}

// GetTradeReports returns reports filtered by status.
func (db *DB) GetTradeReports(ctx context.Context, status string) ([]TradeReport, error) {
	query := `
		SELECT id, reporter_user_id, reported_user_id, order_id, reason,
		       status, reviewed_by, reviewed_at, created_at
		FROM trade_reports
		WHERE status = ?
		ORDER BY created_at DESC
		LIMIT 25
	`
	rows, err := db.conn.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade reports: %w", err)
	}
	defer rows.Close()
	return scanTradeReports(rows)
}

// GetTradeReport retrieves a single report by ID.
func (db *DB) GetTradeReport(ctx context.Context, reportID int) (*TradeReport, error) {
	query := `
		SELECT id, reporter_user_id, reported_user_id, order_id, reason,
		       status, reviewed_by, reviewed_at, created_at
		FROM trade_reports
		WHERE id = ?
	`
	var report TradeReport
	var orderID sql.NullInt64
	var reviewedBy sql.NullString
	var reviewedAt sql.NullTime

	err := db.conn.QueryRowContext(ctx, query, reportID).Scan(
		&report.ID, &report.ReporterUserID, &report.ReportedUserID,
		&orderID, &report.Reason, &report.Status,
		&reviewedBy, &reviewedAt, &report.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get trade report: %w", err)
	}
	if orderID.Valid {
		id := int(orderID.Int64)
		report.OrderID = &id
	}
	if reviewedBy.Valid {
		report.ReviewedBy = reviewedBy.String
	}
	if reviewedAt.Valid {
		report.ReviewedAt = &reviewedAt.Time
	}
	return &report, nil
}

// UpdateTradeReportStatus sets a report's status and reviewer info.
func (db *DB) UpdateTradeReportStatus(ctx context.Context, reportID int, status string, reviewedBy string) error {
	query := `UPDATE trade_reports SET status = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, status, reviewedBy, reportID)
	if err != nil {
		return fmt.Errorf("failed to update trade report: %w", err)
	}

	// Audit log
	details, _ := json.Marshal(map[string]interface{}{
		"report_id":   reportID,
		"action":      status,
		"reviewed_by": reviewedBy,
	})
	db.conn.ExecContext(ctx,
		`INSERT INTO audit_log (action, user_id, details) VALUES (?, ?, ?)`,
		"trade_report_action", reviewedBy, string(details),
	)

	return nil
}

// --- Helpers ---

func scanTradeBans(rows *sql.Rows) ([]TradeBan, error) {
	var bans []TradeBan
	for rows.Next() {
		var ban TradeBan
		var expiresAt sql.NullTime

		err := rows.Scan(
			&ban.ID, &ban.UserID, &ban.Reason, &ban.BannedBy,
			&ban.BannedAt, &expiresAt, &ban.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade ban: %w", err)
		}
		if expiresAt.Valid {
			ban.ExpiresAt = &expiresAt.Time
		}
		bans = append(bans, ban)
	}
	return bans, rows.Err()
}

func scanTradeReports(rows *sql.Rows) ([]TradeReport, error) {
	var reports []TradeReport
	for rows.Next() {
		var report TradeReport
		var orderID sql.NullInt64
		var reviewedBy sql.NullString
		var reviewedAt sql.NullTime

		err := rows.Scan(
			&report.ID, &report.ReporterUserID, &report.ReportedUserID,
			&orderID, &report.Reason, &report.Status,
			&reviewedBy, &reviewedAt, &report.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade report: %w", err)
		}
		if orderID.Valid {
			id := int(orderID.Int64)
			report.OrderID = &id
		}
		if reviewedBy.Valid {
			report.ReviewedBy = reviewedBy.String
		}
		if reviewedAt.Valid {
			report.ReviewedAt = &reviewedAt.Time
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}
