package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// MatchConfidence represents how confident we are in a match
type MatchConfidence int

const (
	ConfidenceNone   MatchConfidence = iota // No match
	ConfidenceLow                            // < 60% similarity
	ConfidenceMedium                         // 60-85% similarity
	ConfidenceHigh                           // > 85% similarity
	ConfidenceExact                          // 100% match
)

const (
	HighConfidenceThreshold   = 0.85
	MediumConfidenceThreshold = 0.60
)

// ItemMatch represents a potential item match
type ItemMatch struct {
	Item       *Item
	Score      float64
	Confidence MatchConfidence
	MatchedVia string // "exact", "alias", "fuzzy"
}

// PortMatch represents a potential port match
type PortMatch struct {
	Port       *Port
	Score      float64
	Confidence MatchConfidence
	MatchedVia string
}

// FindItemMatches finds the best matching items for a given name
func (db *DB) FindItemMatches(ctx context.Context, name string, limit int) ([]ItemMatch, error) {
	normalized := normalize(name)

	// Check for exact match on canonical name
	exactItem, err := db.getItemByName(ctx, name)
	if err == nil && exactItem != nil {
		return []ItemMatch{{
			Item:       exactItem,
			Score:      1.0,
			Confidence: ConfidenceExact,
			MatchedVia: "exact",
		}}, nil
	}

	// Check aliases
	aliasItem, err := db.getItemByAlias(ctx, name)
	if err == nil && aliasItem != nil {
		return []ItemMatch{{
			Item:       aliasItem,
			Score:      1.0,
			Confidence: ConfidenceExact,
			MatchedVia: "alias",
		}}, nil
	}

	// Fuzzy search all items
	items, err := db.getAllItems(ctx)
	if err != nil {
		return nil, err
	}

	var matches []ItemMatch
	for _, item := range items {
		score := calculateSimilarity(normalized, normalize(item.Name))
		if score >= MediumConfidenceThreshold {
			confidence := getConfidence(score)
			matches = append(matches, ItemMatch{
				Item:       &item,
				Score:      score,
				Confidence: confidence,
				MatchedVia: "fuzzy",
			})
		}

		// Also check against aliases
		aliases, _ := db.getItemAliases(ctx, item.ID)
		for _, alias := range aliases {
			aliasScore := calculateSimilarity(normalized, normalize(alias.Alias))
			if aliasScore > score {
				score = aliasScore
			}
		}
	}

	// Sort by score descending
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Limit results
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// FindPortMatches finds the best matching ports for a given name
func (db *DB) FindPortMatches(ctx context.Context, name string, limit int) ([]PortMatch, error) {
	normalized := normalize(name)

	// Check for exact match
	exactPort, err := db.getPortByName(ctx, name)
	if err == nil && exactPort != nil {
		return []PortMatch{{
			Port:       exactPort,
			Score:      1.0,
			Confidence: ConfidenceExact,
			MatchedVia: "exact",
		}}, nil
	}

	// Check aliases
	aliasPort, err := db.getPortByAlias(ctx, name)
	if err == nil && aliasPort != nil {
		return []PortMatch{{
			Port:       aliasPort,
			Score:      1.0,
			Confidence: ConfidenceExact,
			MatchedVia: "alias",
		}}, nil
	}

	// Fuzzy search all ports
	ports, err := db.getAllPorts(ctx)
	if err != nil {
		return nil, err
	}

	var matches []PortMatch
	for _, port := range ports {
		score := calculateSimilarity(normalized, normalize(port.Name))
		if score >= MediumConfidenceThreshold {
			confidence := getConfidence(score)
			matches = append(matches, PortMatch{
				Port:       &port,
				Score:      score,
				Confidence: confidence,
				MatchedVia: "fuzzy",
			})
		}

		// Also check against aliases
		aliases, _ := db.getPortAliases(ctx, port.ID)
		for _, alias := range aliases {
			aliasScore := calculateSimilarity(normalized, normalize(alias.Alias))
			if aliasScore > score {
				score = aliasScore
			}
		}
	}

	// Sort by score descending
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Limit results
	if len(matches) > limit {
		matches = matches[:limit]
	}

	return matches, nil
}

// Helper functions

func normalize(s string) string {
	// Lowercase
	s = strings.ToLower(s)

	// Trim whitespace
	s = strings.TrimSpace(s)

	// Remove special characters except spaces
	var result []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result = append(result, r)
		}
	}

	// Collapse multiple spaces
	s = string(result)
	s = strings.Join(strings.Fields(s), " ")

	return s
}

func calculateSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	// Levenshtein distance
	distance := levenshtein(a, b)
	maxLen := max(len(a), len(b))

	if maxLen == 0 {
		return 0.0
	}

	return 1.0 - (float64(distance) / float64(maxLen))
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(a); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func getConfidence(score float64) MatchConfidence {
	if score >= 1.0 {
		return ConfidenceExact
	} else if score >= HighConfidenceThreshold {
		return ConfidenceHigh
	} else if score >= MediumConfidenceThreshold {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Database helper queries

// GetItemByName retrieves an item by exact name match (exported for handlers)
func (db *DB) GetItemByName(ctx context.Context, name string) (*Item, error) {
	return db.getItemByName(ctx, name)
}

func (db *DB) getItemByName(ctx context.Context, name string) (*Item, error) {
	query := `SELECT id, name, display_name, is_tagged, added_at, added_by, notes FROM items WHERE name = ? COLLATE NOCASE`
	var item Item
	var addedBy sql.NullString
	err := db.conn.QueryRowContext(ctx, query, name).Scan(
		&item.ID, &item.Name, &item.DisplayName, &item.IsTagged,
		&item.AddedAt, &addedBy, &item.Notes,
	)
	if err != nil {
		return nil, err
	}
	if addedBy.Valid {
		item.AddedBy = addedBy.String
	}
	return &item, nil
}

func (db *DB) getItemByAlias(ctx context.Context, alias string) (*Item, error) {
	query := `
		SELECT i.id, i.name, i.display_name, i.is_tagged, i.added_at, i.added_by, i.notes
		FROM items i
		JOIN item_aliases a ON i.id = a.item_id
		WHERE a.alias = ? COLLATE NOCASE
	`
	var item Item
	err := db.conn.QueryRowContext(ctx, query, alias).Scan(
		&item.ID, &item.Name, &item.DisplayName, &item.IsTagged,
		&item.AddedAt, &item.AddedBy, &item.Notes,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (db *DB) getAllItems(ctx context.Context) ([]Item, error) {
	query := `SELECT id, name, display_name, is_tagged, added_at, added_by, notes FROM items`
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

func (db *DB) getItemAliases(ctx context.Context, itemID int) ([]ItemAlias, error) {
	query := `SELECT id, item_id, alias, added_at FROM item_aliases WHERE item_id = ?`
	rows, err := db.conn.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []ItemAlias
	for rows.Next() {
		var alias ItemAlias
		err := rows.Scan(&alias.ID, &alias.ItemID, &alias.Alias, &alias.AddedAt)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, rows.Err()
}

// GetPortByName retrieves a port by exact name match (exported for handlers)
func (db *DB) GetPortByName(ctx context.Context, name string) (*Port, error) {
	return db.getPortByName(ctx, name)
}

func (db *DB) getPortByName(ctx context.Context, name string) (*Port, error) {
	query := `SELECT id, name, display_name, region, added_at, added_by, notes FROM ports WHERE name = ? COLLATE NOCASE`
	var port Port
	var addedBy sql.NullString
	var region sql.NullString
	err := db.conn.QueryRowContext(ctx, query, name).Scan(
		&port.ID, &port.Name, &port.DisplayName, &region,
		&port.AddedAt, &addedBy, &port.Notes,
	)
	if err != nil {
		return nil, err
	}
	if addedBy.Valid {
		port.AddedBy = addedBy.String
	}
	if region.Valid {
		port.Region = region.String
	}
	return &port, nil
}

func (db *DB) getPortByAlias(ctx context.Context, alias string) (*Port, error) {
	query := `
		SELECT p.id, p.name, p.display_name, p.region, p.added_at, p.added_by, p.notes
		FROM ports p
		JOIN port_aliases a ON p.id = a.port_id
		WHERE a.alias = ? COLLATE NOCASE
	`
	var port Port
	err := db.conn.QueryRowContext(ctx, query, alias).Scan(
		&port.ID, &port.Name, &port.DisplayName, &port.Region,
		&port.AddedAt, &port.AddedBy, &port.Notes,
	)
	if err != nil {
		return nil, err
	}
	return &port, nil
}

// GetAllPorts retrieves all ports (exported for handlers)
func (db *DB) GetAllPorts(ctx context.Context) ([]Port, error) {
	return db.getAllPorts(ctx)
}

func (db *DB) getAllPorts(ctx context.Context) ([]Port, error) {
	query := `SELECT id, name, display_name, region, added_at, added_by, notes FROM ports ORDER BY name`
	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []Port
	for rows.Next() {
		var port Port
		var addedBy sql.NullString
		var region sql.NullString
		err := rows.Scan(&port.ID, &port.Name, &port.DisplayName, &region,
			&port.AddedAt, &addedBy, &port.Notes)
		if err != nil {
			return nil, err
		}
		if addedBy.Valid {
			port.AddedBy = addedBy.String
		}
		if region.Valid {
			port.Region = region.String
		}
		ports = append(ports, port)
	}
	return ports, rows.Err()
}

func (db *DB) getPortAliases(ctx context.Context, portID int) ([]PortAlias, error) {
	query := `SELECT id, port_id, alias, added_at FROM port_aliases WHERE port_id = ?`
	rows, err := db.conn.QueryContext(ctx, query, portID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []PortAlias
	for rows.Next() {
		var alias PortAlias
		err := rows.Scan(&alias.ID, &alias.PortID, &alias.Alias, &alias.AddedAt)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, rows.Err()
}

// CreateItem creates a new item
func (db *DB) CreateItem(ctx context.Context, name, displayName, addedBy string) (*Item, error) {
	query := `INSERT INTO items (name, display_name, is_tagged, added_by) VALUES (?, ?, FALSE, ?)`
	result, err := db.conn.ExecContext(ctx, query, name, displayName, addedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Item{
		ID:          int(id),
		Name:        name,
		DisplayName: displayName,
		IsTagged:    false,
		AddedAt:     time.Now(),
		AddedBy:     addedBy,
	}, nil
}

// CreatePort creates a new port
func (db *DB) CreatePort(ctx context.Context, name, displayName, region, addedBy string) (*Port, error) {
	query := `INSERT INTO ports (name, display_name, region, added_by) VALUES (?, ?, ?, ?)`
	result, err := db.conn.ExecContext(ctx, query, name, displayName, region, addedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create port: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Port{
		ID:          int(id),
		Name:        name,
		DisplayName: displayName,
		Region:      region,
		AddedAt:     time.Now(),
		AddedBy:     addedBy,
	}, nil
}
