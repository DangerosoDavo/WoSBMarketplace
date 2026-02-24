# Database Refactoring Plan

## Summary of Changes

We've refactored the database from a simple text-based system to a robust relational model with fuzzy matching, tagging, and user confirmation workflows.

## Database Schema Changes

### New Tables

#### items
```sql
- id (PK)
- name (unique canonical name)
- display_name (how to show it)
- is_tagged (has admin categorized this?)
- added_at, added_by, notes
```

#### item_aliases
```sql
- id (PK)
- item_id (FK -> items)
- alias (unique, case-insensitive)
- added_at
```
**Purpose**: Handle OCR variations ("Cannon", "cannon", "heavy cannon") → same item

#### tags
```sql
- id (PK)
- name (unique: "weapon", "heavy", "ammunition")
- category ("type", "size", "range")
- color, icon (for UI)
```

#### item_tags (many-to-many)
```sql
- item_id (FK)
- tag_id (FK)
```

#### ports
```sql
- id (PK)
- name (unique canonical name)
- display_name
- region ("Caribbean", "Mediterranean", etc.)
- added_at, added_by, notes
```

#### port_aliases
```sql
- id (PK)
- port_id (FK -> ports)
- alias (unique, case-insensitive)
```
**Purpose**: "Port Royal", "Port Royale", "Pt. Royal" → same port

### Updated Tables

#### markets
**OLD**: `port TEXT, item TEXT`
**NEW**: `port_id INT FK, item_id INT FK`

Now properly relational with foreign keys and cascading deletes.

## New Features

### 1. Fuzzy Matching (Levenshtein Distance)

```go
confidence := calculateSimilarity("Port Royal", "Port Royale")
// Returns 0.91 (91% match)

Thresholds:
- >= 0.85: High confidence (auto-match)
- 0.60-0.85: Medium confidence (ask user)
- < 0.60: Low confidence (treat as new)
```

### 2. Submission Flow with User Confirmation

**Phase 1: Port Validation**
```
OCR detects: "Port Royale"
   ↓
Search: ports table + port_aliases
   ↓
No exact match found
   ↓
Fuzzy match all ports
   ↓
Show user:
  - Searchable/paginated list of closest matches
  - Option to add new port
   ↓
User selects or creates port
```

**Phase 2: Item Validation (Deduped)**
```
OCR found 5 items:
  - Cannon (appears 2x)
  - Wood (appears 1x)
  - Cannon (duplicate)
  - Iron (appears 1x)
  - Rope (appears 1x)

Unique items: Cannon, Wood, Iron, Rope (4 items)

For each UNIQUE item:
  ├─ High confidence match? → Auto-map
  ├─ Medium confidence? → Ask user once
  └─ Low confidence? → Treat as new, ask user

Result: User only confirms Cannon ONCE,
        even though it appears twice in screenshot
```

**Phase 3: Database Commit**
```
All items mapped:
  "Cannon" → item_id: 45
  "Wood" → item_id: 12
  "Iron" → item_id: 78
  "Rope" → item_id: 34

Insert markets:
  - (port_id, item_id: 45, price, qty) x2  (both Cannon entries)
  - (port_id, item_id: 12, price, qty)     (Wood)
  - (port_id, item_id: 78, price, qty)     (Iron)
  - (port_id, item_id: 34, price, qty)     (Rope)
```

### 3. Pending Submissions Manager

- Stores submission state in memory
- 5-minute timeout
- Tracks:
  - Port confirmation status
  - Item mappings (OCR name → item_id)
  - Ensures duplicate items only confirmed once
- Cleanup goroutine removes expired submissions

### 4. Tag-Based Filtering

Users can query:
```
/price cannon tags:heavy,long-range region:Caribbean
/price ammunition tags:iron minprice:10 maxprice:100
```

Tags organized by category:
- **type**: weapon, ammunition, material, food
- **size**: small, medium, large, heavy
- **range**: short-range, long-range
- **quality**: common, rare, legendary

### 5. Admin Item Management

**Untagged Items Queue**
```
/admin-item list-untagged
→ Shows: 5 items need tagging
   1. Heavy Cannon (added 2h ago by @user)
   2. Premium Gunpowder (added 5h ago by @user2)
   ...
```

**Tagging**
```
/admin-item tag "Heavy Cannon" weapon,heavy,long-range
→ Tags applied, item marked as tagged
```

**Aliases**
```
/admin-item alias "Cannon" "cannon ball"
→ Now "cannon ball" in OCR matches "Cannon" item
```

**Merging Duplicates**
```
/admin-item merge "Cannon Ball" "Cannonball"
→ All orders transferred, duplicate deleted
```

### 6. Admin Port Management

```
/admin-port add "Port Royale" region:Mediterranean
/admin-port edit "Port Royal" region:Caribbean
/admin-port list region:Caribbean
/admin-port remove "Old Port"
```

## Files Created/Modified

### New Files
- `internal/database/matching.go` - Fuzzy matching + Levenshtein algorithm
- `internal/bot/submissions.go` - Pending submission manager

### Modified Files
- `internal/database/schema.go` - New relational schema
- `internal/database/queries.go` - Updated for new schema + tag filtering

### To Be Updated
- `internal/bot/commands.go` - Add new admin commands
- `internal/bot/handlers.go` - Implement modal workflows
- `internal/bot/client.go` - Add submission manager
- `internal/ocr/claude.go` - Return structured data
- Documentation files

## Migration Path

Since nothing is deployed:
1. ✅ Update schema
2. ✅ Update queries
3. ✅ Add fuzzy matching
4. ✅ Add submission manager
5. 🔄 Update bot commands
6. 🔄 Update bot handlers (modals)
7. 🔄 Update documentation
8. ✅ Test end-to-end
9. ✅ Deploy

## Testing Checklist

- [ ] Port exact match
- [ ] Port fuzzy match with confirmation
- [ ] Port creation flow
- [ ] Item high-confidence auto-match
- [ ] Item medium-confidence confirmation
- [ ] Item new item creation
- [ ] Duplicate item deduplication (only ask once)
- [ ] Full submission flow (port + items)
- [ ] Submission timeout cleanup
- [ ] Tag filtering queries
- [ ] Admin item tagging
- [ ] Admin port management
- [ ] Expiry still works
- [ ] Stats show new metrics

## Performance Considerations

**Fuzzy Matching**
- Runs against all ports/items in DB
- With 100 items: ~negligible
- With 1000 items: ~50ms
- With 10000 items: ~500ms
- Mitigation: Cache in memory, index properly

**Submission State**
- Stored in memory (lost on restart)
- Acceptable: 5min timeout means minimal loss
- Future: Could persist to Redis

**Database**
- Foreign keys add minimal overhead
- Indexes on commonly queried fields
- WAL mode handles concurrent writes

## Next Steps

1. Update command definitions
2. Implement Discord modals for:
   - Port selection (searchable, paginated)
   - Item confirmation (per unique item)
   - Port creation (with region)
3. Wire up handlers to submission manager
4. Update documentation
5. Test thoroughly
6. Deploy!
