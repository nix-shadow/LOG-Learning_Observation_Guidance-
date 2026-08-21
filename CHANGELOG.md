# Changelog

All notable changes to the LOG (Learning Observation Guidance) platform are documented here.
Entries are tied to work-package ids from `docs/html/07-phased-implementation-plan.html`.

---

## [0.5.0] — 2026-08-20 · Phase 4 — Scale & Trust at Depth

### 🏗️ Backend Refactors (WP-4.1)
- **Completion parity seam**: online completion and bulk sync now share one transactional
  `applyCompletion` engine (`completion_engine.go`) — identical status, score, and guidance
  derivation on both paths; parity proven by `TestCompletionParityOnlineVsSync`.
- **Admin console seam (C3)**: `AdminHandler` + `AdminService` + `AdminRepository` layering with
  DTO-guarded activity creation (server-owned `id`/`order`), last-admin demotion guard,
  role validation before the repo, and handler→status-code error mapping.
- **Referential integrity (WP-4.1 C4)**: idempotent FK migration adds `ON DELETE CASCADE`
  constraints to 20 columns across 14 child tables (`migrate_fks.go`); orphan rows are skipped
  (data preserved, warning logged), migrations are re-runnable. Erasure map extended to cover
  parent links, support issues, and learner notes.
- **Honest error states**: sync tests and the dev database cleaned of ghost users/rows that
  earlier passed silently (unique-email collisions); the review UI only shows success when the
  backend actually recorded the attempt (or the offline queue accepted it).

### ⚡ Scale & Performance (WP-4.2)
- **SQLite pool pinned to a single connection** (`db.go`) — zero `SQLITE_BUSY` churn on
  low-end school hardware; reads stay WAL-fast.
- **Hot-path indexes**: `daily_activities (learner_id, date)` composite and an
  `announcements (created_at)` index (kills the temp-b-tree sort). A `completed_at` index was
  benchmarked and **deliberately rejected** — SQLite's planner keeps the PK scan at every
  realistic scale (measured to 200k rows); `docs/QUERY_PLANS.md` records the real
  `EXPLAIN QUERY PLAN` output for all 10 hot queries.
- **RESPECT budget gate**: `frontend/scripts/check-budget.mjs` fails CI when any route's
  First Load JS exceeds 500 kB; wired into `.github/workflows/ci.yml` and `make budget`
  (all 15 routes currently ≤ 174 kB). Real-device TTI < 5s is a documented per-release
  manual step in `docs/RELEASE.md`.

### 📊 Monitoring & Honesty (WP-4.3)
- **Aggregate HTTP metrics** (`internal/metrics`): per-route-pattern request/status counters —
  never user ids (gin `FullPath()`), so the public `GET /metrics` is PII-proof by
  construction; admin JSON view at `GET /api/v1/admin/metrics`.
- **5xx spike alarm**: ≥ 5 server faults in a sliding 60s window trips a logged alert with
  last-alert state (no log spam); re-arms when the window clears.
- **Opt-in analytics**: new `analytics` consent type with a real withdrawal path
  (`status: granted|withdrawn` on `POST /me/consent`, audit trail records
  `privacy.consent_withdrawn`); Settings gets a three-state toggle (not-loaded / on / off).
  `GET /api/v1/admin/analytics/summary` returns **aggregate-only** numbers computed solely
  over opted-in learners — `avg_score` is `null` (never a fabricated 0) when no opted-in
  learner has completions. Proven by `TestAnalyticsSummaryOptInGate`.

### 📦 Release Process (WP-4.4)
- `VERSION`, release procedure in `docs/RELEASE.md` (incl. manual TTI check),
  docs hub in `docs/README.md`, and `CONTRIBUTING.md` for contributors and agents.

---

## [0.4.0] — 2026-08-19 · Phase 3 — Content & Ecosystem

### 📚 Open Educational Resources (WP-3.1)
- Batch OER pack import with license validation and audit logging
  (`POST /api/v1/admin/oer/import`), upstream-attribution pipeline, rate limited per IP.

### 🔄 Privacy Renewal & Transparency (WP-3.2)
- Consent renewals keep evidence fresh; policy notices and disclosure hashes stay in sync
  with the exact text shown.

### 📷 QR Pilot Program (WP-3.3)
- Poster QR scans (`POST /api/v1/pilot/scans`) with honest click-through starts; admin
  stats report only real scan rows — no fabricated impressions.

### 🎯 Module Scoring (WP-3.4)
- Per-activity micro-module completion scores feed the status engine.

### ♿ Accessibility & Inclusive Design (WP-3.5)
- `A11yProvider` with font-scale presets, contrast options, and reduced-motion awareness
  across the app; Nepali string sweep with native-speaker review gated to scope.

---

## [0.3.0] — 2026-08-18 · Phase 2 — Teacher & Parent

### 👨‍👩‍👧 Parent Portal (WP-2.1)
- Guardian self-service signup (`POST /api/v1/auth/parent-signup`) claiming teacher-issued
  invite codes (`POST /api/v1/moderator/students/:id/parent-invite`); read-only child
  digests with per-child opt-in; `parent_access` consent recorded as evidence.

### 🆘 Support Funnel (WP-2.2)
- Who-to-call issue filing for any user; escalated issues land in the moderator inbox
  (`GET /api/v1/moderator/support/inbox`) with resolve workflow.

### 📗 Honest Gradebook (WP-2.3)
- Class gradebook + CSV export built on the WP-1.1 status engine; per-learner teacher notes
  (`GET/PUT /api/v1/moderator/students/:id/note`).

### 🔔 Engagement (WP-2.4)
- Reminder toggles per learner; honest streak and attention flags (zero-streak and
  in-progress counts, computed from real data).

---

## [0.2.0] — 2026-08-16 · Phase 1 — Learning Engine

### ⚙️ Status Engine (WP-1.1)
- Canonical per-activity learner status (`not-started` / `in-progress` / `needs-practice` /
  `completed`) derived from real attempts; observations + next-step guidance generated in
  the same transaction as completion.

### 📋 Release List & Per-Learner Status (WP-1.2, WP-1.4)
- `/learning` renders real status per learner; journey and chart data reflect recorded
  activity, never placeholders.

### 🗣️ Native Language Strings (WP-1.3)
- Nepali (नेपाली) UI strings across key flows, with native-speaker review scope.

### 🏫 Class Join (WP-1.5)
- Learners join classes with teacher invite codes (`POST /api/v1/classes/join`), gated by
  guardian consent like every other learner mutation.

---

## [0.1.0] — 2026-08-10 · WP-0.x — Privacy, Security & Honesty Foundations

### 🔐 Privacy (WP-0.1)
- Consent evidence store (guardian/terms/privacy/parent_access/analytics), bilingual
  notices, disclosure hashes, policy `2026-08-v1`; server-side `RequireConsent` gate on all
  learner mutations; retention purge job (2y learner / 3y audit) at startup + daily ticker;
  forensic erasure (`DELETE /me` + WAL checkpoint + VACUUM, `_secure_delete=ON`);
  personal-data export envelope; `Clear-Site-Data` on logout; client IPs never persisted.
- Offline queue encryption at rest (AES-GCM-256, `sync-keys` store survives logout);
  `.logsync` sneakernet files; queue records preserved on 401/consent-rejection — learner
  work is never deleted.

### 📊 Real-Data Integrity & UX Wiring (WP-0.2/0.3/0.4)
- **Course Catalog from Real API**: `/courses` page now renders `GET /api/courses`
  (paginated, cached for offline) with dynamic category filters derived from data —
  removed the 24 fake `Math.random()` courses.
- **Live Micro-Modules in Lesson Player**: `/learning/[id]` fetches
  `GET /api/activities/:id/modules` and renders the `MicroModuleViewer`; the demo lesson
  remains only as an offline/catalog fallback.
- **Seeded Micro-Module Content**: `database/db.go` seeds 5 bite-sized modules for
  `act-1`/`act-2` independently of users, so existing databases receive them on startup.
- **Progress Upsert for New Learners**: `CompleteActivity` and `SyncBulk` create a
  `Progress` record on first completion instead of returning 500.
- **No Cross-User Data Leaks**: `GetDashboard` returns `404` when the authenticated learner
  doesn't exist instead of silently serving `user-123`'s data.
- **Computed Admin Analytics**: `GetAdminDashboard` now derives ActiveDaily (24h recency)
  and TotalCompletions (sum of progress) from the database instead of `users/2` and
  `activities*20`.
- **Computed Moderator Flags**: `needs_attention` counts zero-streak learners and
  `assignments_due` counts in-progress activities (were hardcoded `8`/`3`).
- **SyncBulk Mirrors Online Flow**: bulk sync now generates supportive observations +
  next-step guidance in the same transaction.
- **Public `/api/ping`**: health probe moved out of the authenticated route group.
- **Unified Logout**: `AuthContext` uses the `api.ts` logout lifecycle — JWT server-side
  revocation, cookie + storage cleanup, and IndexedDB cache purge before redirect.

### 🔒 Security Hardening
- **OTP Hashing**: OTPs hashed with `bcrypt` before persisting; never stored plaintext.
- **Cryptographically Secure OTPs**: replaced the hardcoded `"123456"` demo OTP with
  `crypto/rand`-based 6-digit generation.
- **JWT Revocation**: `TokenBlocklist` + per-token `jti`; `AuthMiddleware` rejects revoked
  tokens; role changes invalidate stale tokens (DB revalidation).
- **Strict Input DTOs**: `CreateActivity` uses a dedicated DTO — clients cannot inject
  `id`, `created_at`, or `order`.
- **Rate Limiter**: per-route IP buckets with `Retry-After` (429s) and RFC 9110 draft-7
  `X-RateLimit-*` headers; stale IP entries pruned by a background cleanup loop.
- **DoS Hardening**: global 4 MB body limit, server read/write/idle timeouts, panic
  recovery without stack leaks (`gin.New()` + `CustomRecovery`), security headers
  (CSP, HSTS, X-Frame-Options, etc.), `SetTrustedProxies(nil)` so `X-Forwarded-For`
  cannot spoof rate-limit keys.

### 🏗️ Backend Architecture
- **Real Database Models**: `Course` and `DailyActivity` GORM models replacing hardcoded
  mocks; `GetCourses` paginates from the `courses` table with real `COUNT`.
- **`GetChartData`**: aggregates `DailyActivity` records per learner from the DB.
- **`GetModeratorRoster`**: JOINs `users` with `progress` — real completion % and streaks.
- **`SyncBulk` (Transactional)**: processes offline sync payloads in a DB transaction,
  scoped to the authenticated user.
- **Graceful Shutdown**: SIGINT/SIGTERM with a 5-second context timeout; `godotenv` env
  loading; multi-tier RBAC (ADMIN / MODERATOR / STUDENT); JWT secret required from the
  environment (aborts when unset).

### 📦 Offline Resilience
- **Sneakernet Sync (`syncExport.ts`)**: `.logsync` export/import for offline devices,
  uploaded via `POST /api/sync/bulk`.
- **Adaptive Learning Engine**: client-side rule-based guidance from IndexedDB without a
  network call (later superseded by the WP-4.1 parity engine).
- **Micro-Learning Architecture**: swipeable bite-sized content modules optimized for
  low bandwidth; offline moderator roster prefetch.

### 🐛 Bug Fixes
- TypeScript errors in `syncExport.test.ts` (fixed `ProgressEvent<FileReader>` mocking,
  removed unused `CACHE_STORE` import); `MicroModuleViewer.tsx` switched to Next.js
  `<Image unoptimized />` to eliminate the LCP build warning.

### 🧪 Testing
- Go tests for new-learner completion, computed `needs_attention` (cross-checked against
  the DB), malformed sync payload rejection; Jest suite covers the offline sync layer
  (113 tests); CI runs backend build/vet/lint/test + frontend typecheck/jest/budget +
  git hygiene (no tracked build artifacts or `.env`).