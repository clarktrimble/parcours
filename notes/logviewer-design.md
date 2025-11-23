# Log Viewer with DuckDB - Design Document

## Overview

A log viewer backed by in-memory DuckDB for fast analytics on structured logs. Supports dynamic schema evolution as new fields appear in logs.

## Target Scale

- **Initial**: Few million log lines
- **Memory**: ~1-2GB for 3-5M lines
- **Growth path**: Disk-backed or Parquet archival for 10M+

## Technology Stack

- **Database**: DuckDB (in-memory)
- **Driver**: `github.com/marcboeker/go-duckdb` (CGO)
- **Interface**: Go `database/sql`
- **File watching**: `github.com/fsnotify/fsnotify`
- **Platforms**: Linux amd64, macOS arm64

## File Loading and Tailing

### Initial Load: DuckDB Native JSON Import

DuckDB has optimized native JSON loading - **much faster** than parsing in Go and inserting.

**For NDJSON logs (newline-delimited JSON, most common):**
```sql
-- Direct load with schema inference and field mapping
CREATE TABLE logs AS
SELECT
    CAST(ts as TIMESTAMP) as timestamp,  -- map ts -> timestamp
    level,                                -- level stays as-is
    msg as message,                       -- map msg -> message
    to_json(*) as raw                     -- full record as JSON (all fields)
FROM read_json_auto('app.log');

-- Add indexes after bulk load
CREATE INDEX idx_timestamp ON logs(timestamp);
CREATE INDEX idx_level ON logs(level);
```

**Note:** Adjust field names to match your actual log format. Example above assumes logs have `ts`, `level`, `msg` fields.

**For JSON array format:**
```sql
CREATE TABLE logs AS
SELECT * FROM read_json('app.log');
```

**Benefits:**
- **Blazing fast**: DuckDB's optimized bulk loader
- **No Go parsing**: Let DuckDB handle schema inference
- **Handles millions of lines**: Loads 5M lines in seconds
- **Automatic schema detection**: Infers types from data

### Tailing: fsnotify for Real-time Updates

After initial load, watch for new log lines using `fsnotify`.

**Architecture:**
```go
import "github.com/fsnotify/fsnotify"

// 1. Initial bulk load
db.Exec(`CREATE TABLE logs AS SELECT ... FROM read_json_auto('app.log')`)

// 2. Track file position
file, _ := os.Open("app.log")
file.Seek(0, io.SeekEnd)  // start at end

// 3. Watch for changes
watcher, _ := fsnotify.NewWatcher()
watcher.Add("app.log")

for event := range watcher.Events {
    switch {
    case event.Op&fsnotify.Write == fsnotify.Write:
        // Read new lines, parse, INSERT
        newLines := readNewLines(file)
        for _, line := range newLines {
            insertLog(db, line)
        }

    case event.Op&fsnotify.Rename == fsnotify.Rename:
        // Log rotation: close old file, open new one
        file.Close()
        file, _ = os.Open("app.log")

    case event.Op&fsnotify.Remove == fsnotify.Remove:
        // File deleted (rotation), wait for new file
        // Re-add watch when file reappears
    }
}
```

**Why fsnotify over tail libraries:**
- **More actively maintained** (~9.3k stars, used by k8s/Docker)
- **Lower-level control** for custom handling
- **Production-proven** at massive scale
- **Not much more code** than tail libraries

### Handling Log Rotation

Common rotation patterns and how to handle them:

**1. Rename + Create (copytruncate=false)**
```
app.log → app.log.1
(new) app.log created
```
- fsnotify fires RENAME event
- Close old file handle
- Open new app.log
- Continue tailing

**2. Truncate in-place (copytruncate=true)**
```
app.log truncated to 0 bytes
```
- fsnotify fires WRITE event
- Detect file size decreased
- Seek to beginning
- Continue tailing

**3. Delete + Create**
```
app.log deleted
(new) app.log created
```
- fsnotify fires REMOVE then CREATE
- Close old file handle
- Re-add watch on new file
- Open and tail new file

### Batching Strategy

**Initial load:**
- Use DuckDB's bulk loader (single SQL statement)
- No batching needed - DuckDB optimizes internally

**Tailing:**
- Option A: **Per-line INSERT** (simple, real-time)
  ```go
  db.Exec("INSERT INTO logs VALUES (?, ?, ?, ?)", ts, level, msg, raw)
  ```

- Option B: **Micro-batching** (better throughput)
  ```go
  batch := make([]LogEntry, 0, 100)

  // Collect lines for 100ms or 100 lines
  if len(batch) >= 100 || time.Since(lastFlush) > 100*time.Millisecond {
      tx, _ := db.Begin()
      for _, log := range batch {
          tx.Exec("INSERT INTO logs VALUES (?, ?, ?, ?)", ...)
      }
      tx.Commit()
      batch = batch[:0]
  }
  ```

**Recommendation**: Start with per-line, add batching if throughput becomes an issue.

### Performance Characteristics

**Initial load (DuckDB native):**
- 1M lines: ~1-2 seconds
- 5M lines: ~5-10 seconds
- Bottleneck: Disk I/O, not parsing

**Tailing (fsnotify + INSERT):**
- Per-line: ~1000-5000 inserts/sec
- Batched: ~10k-50k inserts/sec
- Real-time: <10ms latency from write to queryable

**Edge cases handled:**
- File doesn't exist yet: fsnotify waits for creation
- Rapid writes: Event coalescing handles bursts
- Multiple rotations: Track inode to detect file changes

## Schema Design

### Core Schema

```sql
CREATE TABLE logs (
    id INTEGER PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,  -- canonical: mapped from 'ts' in logs
    level VARCHAR NOT NULL,         -- stays as-is
    message VARCHAR,                -- canonical: mapped from 'msg' in logs
    raw JSON,                       -- full structured log entry
    -- promoted fields added dynamically via ALTER TABLE
);

CREATE INDEX idx_timestamp ON logs(timestamp);
CREATE INDEX idx_level ON logs(level);
```

**Field Mapping:**
- Log field `ts` → table column `timestamp` (canonical)
- Log field `level` → table column `level` (no mapping)
- Log field `msg` → table column `message` (canonical)
- All fields (including originals) → `raw` JSON column

### Field Categories

1. **Core fields** (always columns): `timestamp`, `level`, `message`
2. **Raw JSON blob**: Complete structured log for all fields
3. **Promoted fields**: Hot/frequently queried fields materialized as indexed columns

## Data Flow

### 1. Ingestion

Two ingestion paths depending on scenario:

**A. Initial Bulk Load (startup):**
```
Existing Log File → DuckDB read_json_auto() → Table Created with Indexes
```
- DuckDB reads and parses NDJSON directly
- Schema inferred automatically
- Extremely fast (millions of lines in seconds)
- No Go parsing overhead

**B. Live Tailing (after initial load):**
```
New Log Line → fsnotify WRITE event → Parse in Go → INSERT to DuckDB
                                           ↓
                              Extract: ts→timestamp, level, msg→message, raw JSON
```
- fsnotify detects file changes
- Parse new lines in Go (map `ts`→`timestamp`, `msg`→`message`)
- Extract core fields + store full JSON
- INSERT per line (or micro-batched)

**No schema checks during ingestion** - new fields appear in JSON automatically.

### 2. Querying

**Fast queries (indexed):**
```sql
SELECT * FROM logs
WHERE timestamp > '2024-01-01'
  AND level = 'ERROR'
```

**Ad-hoc field queries (JSON extraction):**
```sql
SELECT timestamp, level,
       raw->>'user_id' as user_id,
       raw->>'request_id' as request_id
FROM logs
WHERE raw->>'user_id' = '12345'
```

**Promoted field queries (fast + indexed):**
```sql
-- After promotion
SELECT * FROM logs WHERE user_id = '12345'  -- uses index
```

### 3. Field Promotion

When a field is queried frequently, promote it to a materialized column:

```go
func PromoteField(db *sql.DB, fieldName string) error {
    // 1. Add column
    _, err := db.Exec(fmt.Sprintf(
        `ALTER TABLE logs ADD COLUMN IF NOT EXISTS %s VARCHAR`,
        fieldName))
    if err != nil { return err }

    // 2. Backfill from JSON
    _, err = db.Exec(fmt.Sprintf(
        `UPDATE logs SET %s = raw->>?`, fieldName), fieldName)
    if err != nil { return err }

    // 3. Create index
    _, err = db.Exec(fmt.Sprintf(
        `CREATE INDEX IF NOT EXISTS idx_%s ON logs(%s)`,
        fieldName, fieldName))
    return err
}
```

**Triggers for promotion:**
- Manual: User clicks "promote field" in UI
- Automatic: Field queried N times without index
- Heuristic: Track query patterns, promote top 10 fields

## Query Performance Characteristics

### Fast Queries
- Indexed columns: `timestamp`, `level`, promoted fields
- Time-range scans: ~100-500ms for 5M rows
- Aggregations: Very fast (DuckDB's strength)

### Slow Queries
- JSON extraction without materialization: Full table scan
- Still acceptable for millions of rows (~1-3s)
- One-off queries where speed doesn't matter

## Schema Evolution Strategy

### New Field Appears

```
1. Log arrives with new field "deployment_id"
2. Stored in raw JSON automatically
3. User queries: WHERE raw->>'deployment_id' = 'prod-123'
4. Query works immediately (slower, unindexed)
5. If queried frequently → promote to column + index
6. Future queries fast
```

### No Downtime
- ALTER TABLE happens in-memory (fast)
- Backfill UPDATE runs once
- No ingestion interruption needed

## Trade-offs

### Pros
✅ New fields appear automatically (JSON storage)
✅ Fast queries on promoted fields (materialized + indexed)
✅ Flexible: Schema evolves with logs
✅ DuckDB excellent for analytics/aggregations
✅ In-memory: Very fast

### Cons
❌ JSON queries slower until promoted (acceptable for ad-hoc)
❌ Field promotion requires schema change (one-time cost)
❌ In-memory limited to ~10M lines practically
❌ CGO dependency for builds

## Implementation Phases

### Phase 1: File Loading & Basic Queries
- [ ] In-memory DuckDB setup
- [ ] Initial bulk load using `read_json_auto()`
- [ ] Core schema with indexes
- [ ] Basic queries on indexed fields (timestamp, level)
- [ ] Simple web UI for viewing logs

### Phase 2: Live Tailing
- [ ] fsnotify integration for file watching
- [ ] Handle WRITE events (new lines)
- [ ] Handle RENAME/REMOVE events (log rotation)
- [ ] Real-time log updates in UI
- [ ] Per-line INSERT during tail

### Phase 3: JSON Queries & Field Discovery
- [ ] Query builder for JSON extraction
- [ ] UI shows all available fields from JSON
- [ ] Filter/search on any field (including JSON fields)
- [ ] Field usage tracking

### Phase 4: Field Promotion
- [ ] Manual promotion API
- [ ] Promote hot fields based on query patterns
- [ ] UI shows promoted vs JSON fields
- [ ] Automatic promotion threshold

### Phase 5: Scale & Performance
- [ ] Micro-batching for tail inserts
- [ ] Disk-backed mode for larger datasets
- [ ] Parquet export for archival
- [ ] Time-based partitioning
- [ ] Multi-file support

## Example Queries

### Get error logs with request context
```sql
SELECT
    timestamp,
    level,
    message,
    raw->>'request_id' as request_id,
    raw->>'user_id' as user_id,
    raw->>'duration_ms' as duration
FROM logs
WHERE level = 'ERROR'
  AND timestamp > now() - INTERVAL '1 hour'
ORDER BY timestamp DESC
LIMIT 100;
```

### Top error messages
```sql
SELECT
    message,
    COUNT(*) as count
FROM logs
WHERE level = 'ERROR'
  AND timestamp > now() - INTERVAL '24 hours'
GROUP BY message
ORDER BY count DESC
LIMIT 20;
```

### P95 latency by endpoint
```sql
SELECT
    raw->>'endpoint' as endpoint,
    percentile_cont(0.95) WITHIN GROUP (
        ORDER BY CAST(raw->>'duration_ms' AS DOUBLE)
    ) as p95_ms
FROM logs
WHERE timestamp > now() - INTERVAL '1 hour'
GROUP BY endpoint
ORDER BY p95_ms DESC;
```

## Open Questions

- [ ] Promotion threshold: How many queries before auto-promote?
- [ ] Column data types: VARCHAR for everything or infer types?
- [ ] Archival strategy: When to move old logs to Parquet?
- [ ] Multi-source: Multiple log streams in same DB or separate?
- [ ] Rotation detection: Track inode changes or rely on fsnotify events?
- [ ] Batching strategy: Time-based (100ms) or count-based (100 lines) or both?
- [ ] Initial load: Load entire file or last N lines only?

## Future Enhancements

- Full-text search on message field (FTS)
- Saved query templates
- Alerting on query patterns
- Export to CSV/JSON/Parquet
- Log correlation (trace IDs across services)
