package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
-- Items master table
CREATE TABLE IF NOT EXISTS items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	display_name TEXT NOT NULL,
	is_tagged BOOLEAN DEFAULT FALSE,
	added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	added_by TEXT,
	notes TEXT
);

-- Item aliases for OCR matching (handles variations and typos)
CREATE TABLE IF NOT EXISTS item_aliases (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	item_id INTEGER NOT NULL,
	alias TEXT NOT NULL UNIQUE COLLATE NOCASE,
	added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE
);

-- Tags master table
CREATE TABLE IF NOT EXISTS tags (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	category TEXT,
	color TEXT,
	icon TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Item-Tag relationships (many-to-many)
CREATE TABLE IF NOT EXISTS item_tags (
	item_id INTEGER NOT NULL,
	tag_id INTEGER NOT NULL,
	added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (item_id, tag_id),
	FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
	FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Ports master table
CREATE TABLE IF NOT EXISTS ports (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	display_name TEXT NOT NULL,
	region TEXT,
	added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	added_by TEXT,
	notes TEXT
);

-- Port aliases for OCR matching
CREATE TABLE IF NOT EXISTS port_aliases (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	port_id INTEGER NOT NULL,
	alias TEXT NOT NULL UNIQUE COLLATE NOCASE,
	added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (port_id) REFERENCES ports(id) ON DELETE CASCADE
);

-- Markets table with foreign keys
CREATE TABLE IF NOT EXISTS markets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	port_id INTEGER NOT NULL,
	item_id INTEGER NOT NULL,
	order_type TEXT NOT NULL CHECK(order_type IN ('buy', 'sell')),
	price INTEGER NOT NULL,
	quantity INTEGER NOT NULL,
	submitted_by TEXT NOT NULL,
	submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP NOT NULL,
	screenshot_hash TEXT NOT NULL,
	FOREIGN KEY (port_id) REFERENCES ports(id) ON DELETE CASCADE,
	FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_markets_port_id ON markets(port_id);
CREATE INDEX IF NOT EXISTS idx_markets_item_id ON markets(item_id);
CREATE INDEX IF NOT EXISTS idx_markets_order_type ON markets(order_type);
CREATE INDEX IF NOT EXISTS idx_markets_expires_at ON markets(expires_at);
CREATE INDEX IF NOT EXISTS idx_markets_port_order ON markets(port_id, order_type);
CREATE INDEX IF NOT EXISTS idx_items_tagged ON items(is_tagged);
CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category);
CREATE INDEX IF NOT EXISTS idx_ports_region ON ports(region);

-- Audit log
CREATE TABLE IF NOT EXISTS audit_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	action TEXT NOT NULL,
	user_id TEXT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	details TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id);

-- Guild settings (per-server configuration)
CREATE TABLE IF NOT EXISTS guild_settings (
	guild_id TEXT PRIMARY KEY,
	admin_role_id TEXT,
	configured_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	configured_by TEXT NOT NULL,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Player profiles for trading
CREATE TABLE IF NOT EXISTS player_profiles (
	user_id TEXT PRIMARY KEY,
	ingame_name TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Player-created trade orders (distinct from OCR market data)
CREATE TABLE IF NOT EXISTS player_orders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT NOT NULL,
	item_id INTEGER NOT NULL,
	order_type TEXT NOT NULL CHECK(order_type IN ('buy', 'sell')),
	price INTEGER NOT NULL,
	quantity INTEGER NOT NULL,
	port_id INTEGER,
	notes TEXT,
	ingame_name TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'completed', 'cancelled')),
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP NOT NULL,
	FOREIGN KEY (item_id) REFERENCES items(id) ON DELETE CASCADE,
	FOREIGN KEY (port_id) REFERENCES ports(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_player_orders_user ON player_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_player_orders_item ON player_orders(item_id);
CREATE INDEX IF NOT EXISTS idx_player_orders_status ON player_orders(status);
CREATE INDEX IF NOT EXISTS idx_player_orders_type ON player_orders(order_type);
CREATE INDEX IF NOT EXISTS idx_player_orders_expires ON player_orders(expires_at);
CREATE INDEX IF NOT EXISTS idx_player_orders_port ON player_orders(port_id);

-- Trade conversations between players
CREATE TABLE IF NOT EXISTS trade_conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	order_id INTEGER NOT NULL,
	initiator_user_id TEXT NOT NULL,
	initiator_ingame_name TEXT NOT NULL,
	creator_user_id TEXT NOT NULL,
	creator_ingame_name TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'closed')),
	started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	ended_at TIMESTAMP,
	last_message_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (order_id) REFERENCES player_orders(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_trade_conv_initiator ON trade_conversations(initiator_user_id);
CREATE INDEX IF NOT EXISTS idx_trade_conv_creator ON trade_conversations(creator_user_id);
CREATE INDEX IF NOT EXISTS idx_trade_conv_status ON trade_conversations(status);
CREATE INDEX IF NOT EXISTS idx_trade_conv_order ON trade_conversations(order_id);

-- Trade bans
CREATE TABLE IF NOT EXISTS trade_bans (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT NOT NULL,
	reason TEXT NOT NULL,
	banned_by TEXT NOT NULL,
	banned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP,
	active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_trade_bans_user ON trade_bans(user_id);
CREATE INDEX IF NOT EXISTS idx_trade_bans_active ON trade_bans(active);

-- Trade reports
CREATE TABLE IF NOT EXISTS trade_reports (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	reporter_user_id TEXT NOT NULL,
	reported_user_id TEXT NOT NULL,
	order_id INTEGER,
	reason TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'reviewed', 'dismissed')),
	reviewed_by TEXT,
	reviewed_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (order_id) REFERENCES player_orders(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_trade_reports_reported ON trade_reports(reported_user_id);
CREATE INDEX IF NOT EXISTS idx_trade_reports_status ON trade_reports(status);
`

type DB struct {
	conn *sql.DB
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Initialize schema
	if _, err := conn.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Item represents an item in the game
type Item struct {
	ID          int
	Name        string
	DisplayName string
	IsTagged    bool
	AddedAt     time.Time
	AddedBy     string
	Notes       string
	Tags        []Tag // Populated when loading with tags
}

// ItemAlias represents an alias for fuzzy matching
type ItemAlias struct {
	ID      int
	ItemID  int
	Alias   string
	AddedAt time.Time
}

// Tag represents a categorization tag
type Tag struct {
	ID        int
	Name      string
	Category  string
	Color     string
	Icon      string
	CreatedAt time.Time
}

// Port represents a trading port
type Port struct {
	ID          int
	Name        string
	DisplayName string
	Region      string
	AddedAt     time.Time
	AddedBy     string
	Notes       string
}

// PortAlias represents an alias for port matching
type PortAlias struct {
	ID      int
	PortID  int
	Alias   string
	AddedAt time.Time
}

// Market represents a market order entry
type Market struct {
	ID             int
	PortID         int
	ItemID         int
	OrderType      string
	Price          int
	Quantity       int
	SubmittedBy    string
	SubmittedAt    time.Time
	ExpiresAt      time.Time
	ScreenshotHash string
	// Populated when joined
	Port *Port
	Item *Item
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int
	Action    string
	UserID    string
	Timestamp time.Time
	Details   string
}

// PlayerProfile represents a player's trading profile
type PlayerProfile struct {
	UserID     string
	IngameName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PlayerOrder represents a player-created trade order
type PlayerOrder struct {
	ID        int
	UserID    string
	ItemID    int
	OrderType string
	Price     int
	Quantity  int
	PortID    *int
	Notes     string
	IngameName string
	Status    string // "active", "completed", "cancelled"
	CreatedAt time.Time
	ExpiresAt time.Time
	// Populated via joins
	Item *Item
	Port *Port
}

// TradeConversation represents a DM relay between two players
type TradeConversation struct {
	ID                  int
	OrderID             int
	InitiatorUserID     string
	InitiatorIngameName string
	CreatorUserID       string
	CreatorIngameName   string
	Status              string // "active", "closed"
	StartedAt           time.Time
	EndedAt             *time.Time
	LastMessageAt       time.Time
}

// TradeBan represents a ban preventing a user from trading
type TradeBan struct {
	ID        int
	UserID    string
	Reason    string
	BannedBy  string
	BannedAt  time.Time
	ExpiresAt *time.Time // nil = permanent
	Active    bool
}

// TradeReport represents a user report against a trader
type TradeReport struct {
	ID             int
	ReporterUserID string
	ReportedUserID string
	OrderID        *int
	Reason         string
	Status         string // "pending", "reviewed", "dismissed"
	ReviewedBy     string
	ReviewedAt     *time.Time
	CreatedAt      time.Time
}
