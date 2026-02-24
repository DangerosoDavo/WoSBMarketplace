package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	// Create temporary database file
	tmpfile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp db: %v", err)
	}
	tmpfile.Close()

	db, err := New(tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("failed to initialize database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpfile.Name())
	}

	return db, cleanup
}

func TestDatabaseInitialization(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Verify tables exist
	ctx := context.Background()
	var count int

	// Check markets table
	err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='markets'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query markets table: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 markets table, got %d", count)
	}

	// Check audit_log table
	err = db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='audit_log'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query audit_log table: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 audit_log table, got %d", count)
	}
}

func TestReplacePortOrders(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create initial orders
	orders1 := []Market{
		{Item: "Cannon", Price: 100, Quantity: 10},
		{Item: "Wood", Price: 50, Quantity: 100},
	}

	err := db.ReplacePortOrders(ctx, "Port Royal", "buy", orders1, "user123", "hash1")
	if err != nil {
		t.Fatalf("failed to insert initial orders: %v", err)
	}

	// Verify orders were inserted
	markets, err := db.GetOrdersByPort(ctx, "Port Royal")
	if err != nil {
		t.Fatalf("failed to query orders: %v", err)
	}
	if len(markets) != 2 {
		t.Errorf("expected 2 orders, got %d", len(markets))
	}

	// Replace with new orders
	orders2 := []Market{
		{Item: "Cannon", Price: 110, Quantity: 5},
		{Item: "Iron", Price: 75, Quantity: 50},
		{Item: "Rope", Price: 25, Quantity: 200},
	}

	err = db.ReplacePortOrders(ctx, "Port Royal", "buy", orders2, "user456", "hash2")
	if err != nil {
		t.Fatalf("failed to replace orders: %v", err)
	}

	// Verify old orders were replaced
	markets, err = db.GetOrdersByPort(ctx, "Port Royal")
	if err != nil {
		t.Fatalf("failed to query updated orders: %v", err)
	}
	if len(markets) != 3 {
		t.Errorf("expected 3 orders after replacement, got %d", len(markets))
	}

	// Verify new data
	found := false
	for _, m := range markets {
		if m.Item == "Iron" && m.Price == 75 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find Iron order, but didn't")
	}
}

func TestDeleteExpiredOrders(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert order that expires in the past
	query := `
		INSERT INTO markets (port, item, order_type, price, quantity, submitted_by, expires_at, screenshot_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	expiredTime := time.Now().Add(-1 * time.Hour)
	_, err := db.conn.ExecContext(ctx, query, "Test Port", "Test Item", "buy", 100, 10, "user123", expiredTime, "hash1")
	if err != nil {
		t.Fatalf("failed to insert test order: %v", err)
	}

	// Insert order that hasn't expired
	futureTime := time.Now().Add(24 * time.Hour)
	_, err = db.conn.ExecContext(ctx, query, "Test Port", "Valid Item", "buy", 200, 20, "user456", futureTime, "hash2")
	if err != nil {
		t.Fatalf("failed to insert valid order: %v", err)
	}

	// Delete expired orders
	count, err := db.DeleteExpiredOrders(ctx)
	if err != nil {
		t.Fatalf("failed to delete expired orders: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 expired order, got %d", count)
	}

	// Verify only valid order remains
	markets, err := db.GetOrdersByPort(ctx, "Test Port")
	if err != nil {
		t.Fatalf("failed to query remaining orders: %v", err)
	}
	if len(markets) != 1 {
		t.Errorf("expected 1 remaining order, got %d", len(markets))
	}
	if markets[0].Item != "Valid Item" {
		t.Errorf("expected 'Valid Item', got '%s'", markets[0].Item)
	}
}

func TestGetPricesByItem(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert orders at different ports
	orders := []struct {
		port      string
		item      string
		orderType string
		price     int
	}{
		{"Port Royal", "Cannon", "buy", 100},
		{"Tortuga", "Cannon", "buy", 95},
		{"Nassau", "Cannon", "sell", 120},
		{"Port Royal", "Wood", "buy", 50},
	}

	for _, o := range orders {
		markets := []Market{{Item: o.item, Price: o.price, Quantity: 10}}
		err := db.ReplacePortOrders(ctx, o.port, o.orderType, markets, "user123", "hash")
		if err != nil {
			t.Fatalf("failed to insert order: %v", err)
		}
	}

	// Query for Cannon
	results, err := db.GetPricesByItem(ctx, "Cannon")
	if err != nil {
		t.Fatalf("failed to query prices: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 Cannon orders, got %d", len(results))
	}

	// Verify sorted by price (buy orders first, then sell)
	if results[0].Price > results[1].Price {
		t.Error("expected results sorted by price")
	}
}

func TestGetStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Insert some test data
	orders := []Market{
		{Item: "Cannon", Price: 100, Quantity: 10},
		{Item: "Wood", Price: 50, Quantity: 100},
	}
	err := db.ReplacePortOrders(ctx, "Port Royal", "buy", orders, "user123", "hash1")
	if err != nil {
		t.Fatalf("failed to insert orders: %v", err)
	}

	err = db.ReplacePortOrders(ctx, "Tortuga", "sell", orders, "user456", "hash2")
	if err != nil {
		t.Fatalf("failed to insert orders: %v", err)
	}

	// Get stats
	stats, err := db.GetStats(ctx)
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	// Verify stats
	if total, ok := stats["total_orders"].(int); !ok || total != 4 {
		t.Errorf("expected 4 total orders, got %v", stats["total_orders"])
	}

	if ports, ok := stats["unique_ports"].(int); !ok || ports != 2 {
		t.Errorf("expected 2 unique ports, got %v", stats["unique_ports"])
	}

	if submissions, ok := stats["submissions_today"].(int); !ok || submissions != 2 {
		t.Errorf("expected 2 submissions today, got %v", stats["submissions_today"])
	}
}
