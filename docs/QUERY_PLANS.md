# QUERY_PLANS — Hot-Path SQL Evidence (WP-4.2)

This document records the REAL `EXPLAIN QUERY PLAN` output for the hot
queries behind the dashboard, chart, roster, and audit pages, plus the
reasoning (with measurements) behind every index decision — including the
indexes we deliberately did NOT add.

Captured against a copy of the development database (`backend/data/log.db`)
at 2026-08-20, using the schema after `MigrateForeignKeys` + the WP-4.2
index additions. Verbatim planner output; nothing edited.

## Pool sizing (db.go)

SQLite is a single-writer engine. `InitDB` now pins the connection pool:

- `SetMaxOpenConns(1)` / `SetMaxIdleConns(1)` — goroutines serialize on
  SQLite's internal lock instead of tripping `SQLITE_BUSY`; zero
  busy-retry churn on low-end school hardware.
- `SetConnMaxLifetime(0)` — SQLite connections are cheap file handles;
  never recycle mid-flight.

Reads stay WAL-fast and the app is low-traffic, so the serialization cost
is negligible at LOG's scale.

## Dashboard — active daily learners (last 24h)

```sql
SELECT DISTINCT learner_id FROM learner_activities WHERE completed_at > '2026-08-19 20:00:00';
```

```
SCAN learner_activities USING INDEX sqlite_autoindex_learner_activities_1
```

The planner serves this from the `(learner_id, activity_id)` PK index —
already optimal at this scale. **Studied-and-rejected:** a dedicated
`completed_at` index was benchmarked (synthetic 20k / 200k rows, 10% recent,
with and without `ANALYZE`): the plan stayed a PK scan and runtime was
identical (18.806 ms vs 18.825 ms per query at 200k rows). SQLite's cost
model keeps the PK scan for the `DISTINCT` shape, so the index would be
dead weight for the life of the school. No index added — the honest
decision is documented here instead of an index that changes nothing.

## Chart data — daily activity for one learner

```sql
SELECT * FROM daily_activities WHERE learner_id = 'user-123' ORDER BY date ASC;
```

```
SEARCH daily_activities USING INDEX idx_da_learner_date (learner_id=?)
```

The WP-4.2 composite index `(learner_id, date)` (was: two separate
single-column indexes) serves the filter **and** the sort in one scan — no
temp b-tree.

## Learner observations / guidance (desc)

```sql
SELECT * FROM observations WHERE learner_id = 'user-123' ORDER BY created_at DESC;
SELECT * FROM guidances WHERE learner_id = 'user-123' ORDER BY created_at DESC;
```

```
SEARCH observations USING INDEX idx_learner_created (learner_id=?)
SEARCH guidances USING INDEX idx_guidance_learner (learner_id=?)
```

Pre-existing composite `(learner_id, created_at)` indexes already cover
both.

## Audit log pagination (admin)

```sql
SELECT * FROM audit_logs ORDER BY id DESC LIMIT 20 OFFSET 0;
```

```
SCAN audit_logs
```

`id` is the INTEGER PRIMARY KEY (rowid) — a reverse rowid scan, no temp
b-tree. **Studied-and-rejected:** a `created_at` index (the earlier plan
assumption) — the real query orders by `id`, which the PK already covers.
Not added.

## Roster — members of a class / batch progress

```sql
SELECT * FROM class_members WHERE class_id = 'cls-1';
SELECT * FROM progresses WHERE learner_id IN ('user-123','user-456','user-789');
```

```
SEARCH class_members USING INDEX sqlite_autoindex_class_members_1 (class_id=?)
SEARCH progresses USING INDEX sqlite_autoindex_progresses_1 (learner_id=?)
```

Composite PKs serve both joins directly.

## Announcements (newest first, all roles)

```sql
SELECT * FROM announcements ORDER BY created_at DESC LIMIT 20;
```

Without index:
```
SCAN announcements
USE TEMP B-TREE FOR ORDER BY
```

With the WP-4.2 `idx_ann_created_at` index:
```
SCAN announcements USING INDEX idx_ann_created_at
```

The temp sort is gone; the index scan walks the newest rows directly. This
is the one genuinely measurable index win on the read side.

## Token blocklist purge (startup)

```sql
DELETE FROM token_blocklists WHERE expires_at < '2026-08-20 00:00:00';
```

```
SEARCH token_blocklists USING INDEX idx_token_blocklists_expires_at (expires_at<?)
```

Existing expiry index is used as-is.

## Re-running this evidence

```bash
sqlite3 data/log.db "EXPLAIN QUERY PLAN <query>;"
```

Plans were captured from a live copy with `_foreign_keys=on`; run
`ANALYZE` first if you want statistics-driven plans (this document's
conclusions hold either way — verified both with and without stats).