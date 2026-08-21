# LOG — Research Findings & Improvement Plan

Comprehensive audit of the LOG platform (backend, frontend, offline-sync layer, product, and engineering process) with evidence-based findings and a prioritized improvement plan.

**Status:** Research only — no code was changed as part of this audit.
**Method:** Deep sub-agent audits of the backend & frontend plus manual verification of the offline layer, infrastructure, and product surface. All claims verified against the codebase with `file:line` evidence.

---

## ⚡ Phase 0 Revalidation (2026-08-16)

Re-audited against the current code (auth moved to `backend/internal/`, per-learner model landed, several criticals fixed). Statuses below are the **live** verdicts with fix locations.

| # | Finding | Verdict | Evidence / fix location |
|---|---|---|---|
| 1.1 | GoogleAuth forgeable identity | ✅ FIXED | `idtoken.Validate` with `GOOGLE_CLIENT_ID` audience, server-side (`backend/internal/service/auth_service_impl.go:132-169`); bare `{email}` rejected |
| 1.2 | Committed JWT secret + fallback | ✅ FIXED | `docker-compose.yml:14` `${JWT_SECRET:?}` deploy-time guard; `main.go:30` fatal; no fallback constant; HMAC-type enforcement (`middleware.go:101-106`) |
| 1.3 | Global activity status | ✅ FIXED | `LearnerActivity{LearnerID, ActivityID, Status, CompletedAt, Score}` (`internal/domain/domain.go:49-55`); all readers use it |
| 1.4 | Logout redirect trap | ✅ FIXED | AuthContext logout clears the token cookie (`frontend/src/context/AuthContext.tsx`) + `api.ts` local cleanup |
| 1.5 | OTP brute-force + logging | ✅ FIXED | `OTPRecord.Attempts`; 5 fails → invalidate (`auth_service_impl.go`); `DeleteOTP` now removes all records for the phone (`auth_repo.go:31-34`); `SetTrustedProxies(nil)` (`main.go`); OTP never logged (phone only) |
| 1.6 | SW cross-account cache leak | ✅ FIXED | `next.config.mjs` NetworkFirst route filtered for cross-origin |
| 1.7 | Queue 401 deletion + non-idempotent completion | ✅ FIXED | queue preserved on 401 + token replay (`api.ts`); completion + SyncBulk idempotent (`completion_repo.go`, `sync_repo.go`) |
| 1.8 | Tracked build artifacts | ✅ FIXED | `sw.js`/`workbox-*.js` gitignored + untracked |
| 2.1 | JWT role not revalidated | ✅ FIXED | `AuthMiddleware` re-loads user, compares DB role, rejects demoted/deleted (`middleware.go`); test `TestAuthMiddlewareRejectsDemotedUser` |
| 2.2 | Fabricated data / no DailyActivity path | ✅ FIXED | goal ring from real `progress`; roster honest zeros; **DailyActivity written on every completion + sync** (`completion_repo.go`, `sync_repo.go`); Mon–Sun zero series kept by design (honest empty state, AGENTS.md §1) |
| 2.3 | Streak/score date-blind | 🔧 FIXED (see below) | `Progress.LastActivityDate`; same-day completions no longer double streak |
| 2.4 | SyncBulk swallows errors | 🔧 FIXED (partial) | per-item `failedCount` surfaced in response; invalid items skipped without 200-masking |
| 2.5 | Roster half-fabricated | ✅ FIXED | `class_name` derived from real first course; `last_active` real `UpdatedAt` (`—` when none); **N+1 eliminated** — progress batch-loaded in one `IN` query (`moderator_repo.go`), `RosterEntry` pairs user+progress so the service never queries per learner; query count proven constant at 25 vs 50 students (`roster_test.go`) |
| 2.6 | Rate limiter lockout tension | 🔶 CHANGED | trusted proxies disabled (XFF spoof closed); per-IP 5/min NAT lockout documented tradeoff |
| 2.7 | Incomplete frontend states | 🔶 PARTIAL | skeletons + honest offline/queued states landed; per-page retry polish remains |
| 2.8 | Dead code & dead UI | 🔶 PARTIAL | `adaptiveEngine.ts` deleted; mock Google login gone; `/moderator/classes` removed; dead buttons remain (roadmap) |
| 2.9 | Edge RBAC presence-only | 🔧 FIXED | `middleware.ts` decodes the JWT `role` claim at the edge |
| 2.10 | Last-admin demotion | ✅ FIXED | `UpdateUserRole` counts remaining ADMINS before any demotion and rejects the final one (400 "Cannot demote the last admin") — no recovery path otherwise; same-role no-ops and non-admin demotions unaffected; rejected changes never hit the audit trail (`admin_test.go` → `TestUpdateUserRoleLastAdminGuard`) |
| 2.11 | Offline queue dedup ignores body | ✅ FIXED | `queueRequest` (`src/lib/api.ts`) now merges payloads instead of dropping them: completions keep the best-scoring attempt (server best-score semantics), all other actions keep the newest body (last-write-wins) — one queued entry per action, replaced in place via `put` (keyPath `id`); 4 new Jest cases in `api.test.ts` |
| 2.12 | SQLite seam missing pragmas | ✅ FIXED | DSN now sets `_busy_timeout=5000` (concurrent teacher+student writes never surface SQLITE_BUSY) and `_foreign_keys=on` (referential integrity on enroll/assign) at the seam (`database/db.go`); WAL + NORMAL retained |
| 2.13 | Fat school handler (4 concepts, one file) | ✅ FIXED | `school_handler.go` slims to the shared `SchoolHandler` seam (struct/constructor/`actor`/`audit`); per-resource files: `class_handler.go`, `assignment_handler.go`, `announcement_handler.go`, `audit_handler.go`, `export_handler.go` — routes unchanged; tests split to match (`class_test.go`, `assignment_test.go`, `audit_test.go`, `export_test.go`) |
| 2.14 | Admin/moderator pages untested monoliths | ✅ FIXED | sections extracted into tested components owning their data through the `fetchWithCache` seam: `ClassManager`, `AnnouncementComposer` (shared by both dashboards), `AuditLogTable`, `RosterOverview`, `AssignmentManager` (`src/components/{admin,moderator}/`); pages shrink to composition; 20 new Jest cases prove loading/empty/error states (48 total), `@/` alias added to `jest.config.js` |
| A1 | admin.go outside `internal/` | ✅ FIXED | moved to `backend/internal/handler/admin.go`; `backend/api/` removed |
| A2 | `/moderator/classes` stub | ✅ FIXED | route removed (no fabricated response) |
| A3 | AGENTS.md drift | ✅ FIXED | paths/env steps/Google-auth note updated |
| A4 | ThreeBackground heavy 3D | ✅ FIXED | device-gated + `next/dynamic` split (`ThreeBackground.tsx`) |
| A5 | Thin test coverage | ✅ FIXED | 5 new Go security tests + 3 new Jest tests (14 total) |

Only the roadmap items (SMS gateway, teacher workflow, Nepali i18n, reconnect digest) remain as product work — none block a demo.

---

## ⚡ Phase 1 — Product Hardening (2026-08-20, in progress)

Phase 1 ships RC-01/02/03/09 from `docs/html/07-phased-implementation-plan.html` (WP-1.1 → WP-1.5). Live verdicts below.

| WP | What shipped | Verdict |
|---|---|---|
| WP-1.1 (RC-01) | Canonical per-learner status engine (`not-started`/`active`/`needs-practice`/`completed`); supportive vocabulary everywhere ("Review", "Practice again", "Continue", "Start") | ✅ DONE |
| WP-1.2 (RC-02) | Needs-practice rule (accuracy < 70% flags; improving re-attempt clears; replays idempotent) in both write paths; explanation shown for correct AND incorrect answers; SEE 9–12 pilot content (3 activities, 18 modules: Quadratic Equations, Electricity, Tenses); `DailyActivity.Attempts`/`Accuracy` with weighted running mean in both write paths; accuracy area + practice totals on the observation chart | ✅ DONE |
| WP-1.3 (i18n) | next-intl (client-side, no URL prefix — school-LAN SPA shell); EN/NP switcher in nav; Noto Sans Devanagari bundled; Bikram Sambat dates (AD⇄BS round-trip, Devanagari digits) in `src/lib/bikramSambat.ts` + 7 tests; NP copy for nav/status/quiz/chart/review/reminder | ✅ DONE |
| WP-1.4 (RC-09) | SM-2 review scheduler (local-first IndexedDB v5 `review-schedule`, synced because every review is a real completion replay); dashboard review-queue card with honest empty state; opt-in permission-gated daily reminders (tab-open limitation disclosed); streaks stay backend-derived from real completions only | ✅ DONE |
| WP-1.5 (RC-03) | Moderator onboarding + class wizard; invite-by-code; roster CSV import with one-time passwords; per-student progress (WP-1.1 engine) | ✅ DONE |

**WP-1.4 details (Spaced repetition & streaks):**
- `src/lib/spacedRepetition.ts` — pure SM-2: quality = round(accuracy × 5) from REAL `correct_count/total_count`; q < 3 resets repetition to a 1-day interval (ease factor survives — a lapse is scheduling noise, never a "hard item" verdict); ease clamped ≥ 1.3; due-date math UTC-date based. 9 unit tests.
- `src/lib/reviewStore.ts` + `api.ts`/`crypto.ts` DB bumped to version 5 (`review-schedule` store, `keyPath: activityId`, created idempotently). Never encrypts; scheduler state is device-local (like settings), the completions behind it flow through the AES-GCM queue.
- `ReviewQueueCard.tsx` — dashboard card listing genuinely due/overdue items; empty schedule renders an honest "Nothing is due right now." state (never invented items); 3-test honesty suite.
- `ReminderToggle.tsx` (settings) — `Notification.requestPermission()` only on explicit click; daily time picker; fires once per day while the tab is open; UI explicitly discloses "no push server on the school LAN" (no VAPID — honest local approximation of web-push).
- Streaks remain backend `Progress.CurrentStreak` — derived exclusively from genuinely new completions, date-aware in the learner's timezone (`completion_repo.go`); the scheduler never fabricates streak activity.

**WP-1.3 details (Bilingual shell):**
- Locale via `src/i18n/LocaleProvider.tsx`: localStorage `log-locale`, fallback `navigator.language` (`ne` → `np`); NO URL-prefix routing (per-device language, not per-URL — right call for offline school LANs).
- `@sbmdkl/nepali-date-converter` (BS 1978–2099; anchor 1921-04-13 = BS 1978-01-01; round-trip 2026-08-20 ⇄ 2083-05-04 — note this dataset's Baisakh 1 = Apr 13, so Aug 20, 2026 = **Bhadra 4, 2083**). TS shim at `src/types/nepali-date-converter.d.ts` (package exports omit the "types" condition).
- Jest: next-intl's untranspiled ESM is mocked against the real `en.json` (`jest.setup.ts`) so component tests keep asserting actual user-facing English copy.

**WP-1.5 details (Moderator day-one experience):**
- Backend: `Class.InviteCode` (uniqueness enforced in service via re-roll loop — no DB unique index, keeps legacy empty rows migration-safe); `POST /moderator/classes`, `POST /classes/join` (consent-gated learner mutation), `POST /moderator/classes/:id/roster/import` (multipart CSV: `name,email[,phone,password]`; per-row 1-based honest errors; generated temp passwords returned exactly once in `report.passwords`, never logged), `GET /moderator/classes` now exposes `invite_code`; `GET /moderator/students/:id` reuses `GetDashboardData` (WP-1.1 status engine, hard-404 scope gate via `StudentInTeacherClasses`). Seed class `cls-1` code `LOG101`. 5 new tests (`wp15_test.go`) + 84→90 jest.
- Frontend: `ClassWizard.tsx` (3 steps: create → show/copy invite code + CSV import with report incl. one-time password table and row errors → done; honest failure toasts); onboarding banner on the moderator portal when `classes.length === 0` (3-step guide); roster rows clickable → `StudentProgressModal` (canonical status chips, translated supportive labels — "Review"/"Practice again"/"In progress"/"Not started" in EN+NP); `JoinClassCard` on the student dashboard (uppercases + trims code, honest "No class found" toast on 404). New i18n namespaces: `onboard`, `wizard`, `studentProgress`, `joinClass` (EN+NP).

---

## ⚡ Phase 2 — Teacher & Parent (2026-08-20)

Phase 2 ships RC-04/06/08 from `docs/html/07-phased-implementation-plan.html` (WP-2.1 → WP-2.4): the parent portal, the who-to-call support funnel, an honest gradebook, and a reconnect digest that makes offline sync visible. Backend green (`go build`, `go test ./...`, golangci-lint 0 issues), frontend green (`tsc --noEmit`, `next lint`, 101 jest tests, `next build`).

| WP | What shipped | Verdict |
|---|---|---|
| WP-2.1 (RC-04) | Parent portal: teacher-issued one-time invite codes (`ParentLink` + atomic `ClaimParentLinkTx`), `POST /api/v1/auth/parent-signup` (disclosure_hash validated), read-only digest (`/api/v1/parents/children`, `/children/:id/digest` — id+name+opt-in only, no OTPs/contacts/observations), opt-in toggle (`POST /children/:id/opt-in`, default off); `/parent` page + middleware PARENT gate + login tab "Parent · अभिभावक" | ✅ DONE |
| WP-2.2 (RC-06) | Support funnel: `/support` wizard (category → bilingual guidance → helped? → escalate → done) + `POST /api/v1/support/issue` (rate-limited); moderator/admin inbox (`GET /support/inbox`, `PUT /support/issue/:id` resolve with note), `SupportInbox` on /moderator + /admin; `GET /support/my-issues` for the reporter; actions audit-logged | ✅ DONE |
| WP-2.3 (RC-08) | Honest gradebook: `GET /moderator/gradebook[.csv]` — real accuracy/attempts per learner×activity, attempts=0 → "Not yet assessed", CSV sanitized + BOM; `GradebookOverview` on /moderator; per-learner teacher notes (`GET/PUT /moderator/students/:id/note`, `LearnerNote` table) with supportive-language guidance | ✅ DONE |
| WP-2.4 (offline) | Reconnect digest: `syncQueue` writes `{synced, failed, at}` to `log_reconnect_digest` + dispatches `log:digest-ready` (both window-online and manual flush paths; written only when synced>0 \|\| failed>0); `ReconnectDigest` card on the dashboard (dismiss clears); SyncIsland (WP-0.3) remains the persistent offline/syncing indicator with live queue count | ✅ DONE |

**WP-2.1 details (Parent portal):**
- Backend: `ParentLink` model (`domain.go`); `CreateParentLink` (teacher, 6-char code, `expires_at` 7 days) + `ClaimParentLinkTx` (single tx: PARENT user + pending→linked transition + `parent_access` consent with `disclosure_hash`); routes `/moderator/students/:id/parent-invite`, `/auth/parent-signup`, `/parents/*` (scoped to the caller's linked children only); `disclosure_hash` accepts 64-hex sha256 OR `djb2-<hex>` (non-secure-context school LANs — weaker by design, like the queue's `enc:null`); `ConsentRecord.IP` never persisted.
- Frontend: login page third tab with bilingual `PARENT_NOTICE_EN/NP` constants (the checkbox renders exactly the text the hash commits to — drift = invalid evidence); `/parent` page (children list, per-child digest expand, opt-in toggle); `AuthContext.login` lands PARENT → `/parent`; `middleware.ts` gates `/parent` to the PARENT role; nav item (HeartHandshake) for parents only.

**WP-2.2 details (Support funnel):**
- Backend: `SupportIssue` (`category` ∈ device/connectivity/account/content/other, `escalated`, `status`, `resolution_note`, timestamps); `POST /support/issue` rate-limited (10/IP); `GET /support/inbox` returns open escalated issues only; `PUT /support/issue/:id` resolves with required note; `GET /support/my-issues` scoped to the reporter; `support.*` events audit-logged.
- Frontend: `/support` wizard walks category → bilingual guidance (real, local copy — never invented help) → "did this help?" → describe + escalate → done; `SupportInbox` on /moderator + /admin with inline resolve + note; `GET /support/my-issues` list; nav item (LifeBuoy) for all roles.

**WP-2.3 details (Honest gradebook):**
- Backend: `GET /moderator/gradebook?class_id=` returns per-student rows over the class's activities with REAL `accuracy`/`attempts` from the completion engine; `GET /moderator/gradebook.csv?class_id=` (UTF-8 BOM, `sanitizeCSVCell`); `LearnerNote` table + GET/PUT note routes (teacher-scoped).
- Frontend: `GradebookOverview` — student × activity matrix, "Not yet assessed" when `attempts === 0` (never an invented 0% or dash), CSV export via direct authenticated fetch + blob download, inline per-student note editor; cache-invalidation rule: `/note` mutations clear the whole API cache; tests: `GradebookOverview.test.tsx` (3), `wp23_test.go` (gradebook honesty + CSV).

**WP-2.4 details (Reconnect digest):**
- `src/lib/api.ts`: digest written inside `syncQueue` (single choke point covering the `online` listener AND `flushSyncQueue`), key `log_reconnect_digest`, event `log:digest-ready`; helpers `getReconnectDigest()`/`clearReconnectDigest()`.
- `ReconnectDigest.tsx` (dashboard sidebar, above ReviewQueueCard): listens for the event, renders "Back online — changes synced: N · failed: M" with the `at` timestamp, dismiss clears; renders nothing on honest zero (no event → no card). Tests: 5 (event fire, pre-mount read, dismiss, zero state, helper round-trip).
- Sync status indicator remains `SyncIsland` (WP-0.3): persistent offline/syncing pill, live queue count, manual "Sync now" — the digest explains what that sync actually did.

**New tests:** backend `wp21_test.go` (parent invite/claim, sanitized digest, opt-in), `wp22_test.go` (issue create, escalation, inbox scoping, resolve), `wp23_test.go` (gradebook honesty, CSV, notes) — all in `internal/handler/`; frontend `ReconnectDigest.test.tsx` (5), `SupportInbox.test.tsx` (3), `GradebookOverview.test.tsx` (3) — 101 total. Frontend CI chain (`tsc --noEmit` + `next build` + jest) green; a pre-existing TS2540 in `crypto.test.ts` was fixed with `@ts-expect-error`.

---

## ⚡ Phase 3 — Content & Ecosystem (2026-08-20)

Phase 3 ships RC-07/10/11/12 from `docs/html/07-phased-implementation-plan.html` (WP-3.1 → WP-3.5): OER metadata + import pipeline, full SEE 9–12 content, the QR poster pilot, NSL captions + accessibility packs, and formal partnership lanes. Backend green (`go build`, `go vet`, `go test ./...`, golangci-lint 0 issues), frontend green (`tsc --noEmit`, `next lint`, 113 jest tests, `next build` incl. the new `/qr/[activityId]` route).

| WP | What shipped | Verdict |
|---|---|---|
| WP-3.1 (RC-07) | OER metadata: `Activity` carries `license`/`license_url`/`attribution`/`source_url` with a strict `OERAllowedLicenses` allowlist; catalog cards + lesson pages render license + attribution honestly (nothing shown when absent). Import pipeline: `POST /api/v1/admin/oer/import` — per-row license checks, attribution required for third-party rows, existing IDs skipped (progress never orphaned), audit-logged; seeded OER packs act-11/act-12 (original LOG content under CC BY-SA 4.0) | ✅ DONE |
| WP-3.2 (content) | SEE 9–12 units act-6..act-10: Statistics, Chemical Reactions, Social Studies (federalism), Computer Science (number systems), Nepali व्याकरण (bilingual NP) — 24 new modules, each a practice-first quiz bank with a supportive explanation; per-activity seed gating so existing databases never duplicate | ✅ DONE |
| WP-3.3 (RC-10) | QR poster program: admin `PilotPosters` panel renders a printable QR per activity (qrcode lib, offline); `/qr/<activityId>` landing records the scan, warms the offline cache (fetchWithCache), and marks started on click-through. Pilot measurement: `PilotScan` table (no IP/device data by design), `POST /api/v1/pilot/scans` + `/scans/:id/start` (public, rate-limited), `GET /api/v1/admin/pilot/stats` — scans/starts/start-rate derived from real rows, honest zeros | ✅ DONE |
| WP-3.4 (RC-12) | NSL captions: `Activity.caption_text` + caption block on the lesson page (badge only when a real caption exists; act-4 seeded with an NSL caption; consent sourcing documented in PARTNERSHIP_LANES.md). Accessibility packs: Settings card with A/A/A font scale + high-contrast switch — `lib/a11y.ts` + `A11yProvider` apply `data-font-scale`/`data-contrast` on `<html>`, persisted in localStorage, works fully offline | ✅ DONE |
| WP-3.5 (RC-11) | `docs/PARTNERSHIP_LANES.md`: school/MoE MoU template (10 clauses: consent-first, retention/erasure, no re-sharing, incident response, Nepal Privacy Act 2075 alignment), attribution & remix rules (allowlist, SA/NC), shared-library model, contributor credits — the rules are enforced in code by the OER pipeline + UI credit lines | ✅ DONE |

**WP-3.1 details (OER metadata & import):**
- `domain.go`: `OERAllowedLicenses` (CC BY 4.0 / BY-SA / BY-NC / BY-NC-SA / CC0 / "Own work (LOG team)") + `OERLicenseURLs` + `IsAllowedOERLicense`; `OERPack`/`OERImportReport` (honest imported/skipped/rejected counts).
- `oer_service_impl.go`: rejects missing-id, unknown/empty license, and un-attributed third-party rows — each with a per-row reason; normalizes the canonical license URL; `activityRepo.CreateMany` inserts in one tx skipping existing IDs.
- `oer_handler.go` → `POST /admin/oer/import` (rate-limited 5/IP, audit `oer.import` with counts); tests `wp31_test.go` (valid pack, re-import skip, unknown license + missing attribution rejection, nameless pack 400).
- Seed backfills honest metadata on act-1..act-5 ("Own work (LOG team)") and imports two CC BY-SA 4.0 packs (act-11/act-12) with real attribution lines.
- Frontend: license badge + attribution + NSL badge on catalog cards; "Licensed under …" credit line + caption block on lesson pages; types updated (`Activity.license*`, `caption_text`).

**WP-3.3 details (QR pilot):**
- `PilotScan {poster_id, source, started, created_at}` — deliberately NO IP/device/user columns (privacy by design, AGENTS.md §3b).
- Public routes rate-limited 30/IP (`RateLimitPilotScan`); unknown poster → 404 (never a fabricated scan); stats route admin-only; `TestPilotStatsHonestZeros` pins the no-data state to real zeros.
- `/qr/<activityId>` landing: records scan (fire-and-forget; offline → honest "not counted" copy), warms `/activities/:id/modules` + `/learning-journey` into the api-cache (offline demo kit), marks `started` on click-through, then navigates.
- Admin `PilotPosters` panel: QR per activity (pointing at `window.location.origin`, so school-LAN deployments print correct URLs), honest stats grid (total/24h/starts/start-rate/posters seen) + refresh; tests `PilotPosters.test.tsx` (4) + `QRLanding.test.tsx` (3).

**WP-3.4 details (NSL & accessibility):**
- Caption track: `caption_text` on the Activity model; lesson page shows the caption block only when the field is non-empty — honest, nothing invented; act-4's caption is written bilingual (NSL description + EN gloss).
- `lib/a11y.ts`: `FontScale` (normal 1 / large 1.18 / xlarge 1.35), `loadA11yPrefs` rejects unknown stored values (falls back to defaults), `applyA11yPrefs` sets `data-font-scale`/`data-contrast` on `<html>`; `A11yProvider` in the root layout applies at mount (no flash); `globals.css` overrides lift muted text/border opacity under high contrast; Settings card exposes both controls; tests `a11y.test.ts` (5).

**New tests:** backend `wp31_test.go` (3), `wp33_test.go` (4) — all in `internal/handler/`; frontend `a11y.test.ts` (5), `PilotPosters.test.tsx` (4), `QRLanding.test.tsx` (3) — 113 total (was 101). Tracker: 63/72 (Phase 3 10/10).

---

## ⚡ Phase 4 — Scale & Trust at Depth (2026-08-20)

Phase 4 retires the architecture-review debt (C2/C3/C4), proves the performance budgets, and adds honest monitoring — tracker 72/72. Backend green (`go build`, `go vet`, `go test ./...` all packages), frontend green (`tsc --noEmit`, `next lint` 0 warnings, 113 jest tests, `node scripts/check-budget.mjs` build + budget gate).

| WP | What shipped | Verdict |
|---|---|---|
| WP-4.1 (C2) | Completion parity: `applyCompletion` seam (`backend/internal/repository/completion_engine.go`) shared by online complete + bulk sync — same status/score/guidance on both paths; client-side-only engine deleted (`frontend/src/lib/adaptiveEngine.ts`); lesson page shows success only when the backend recorded the attempt or the offline queue accepted it (`recorded` gate in `learning/[id]/page.tsx`) | ✅ DONE |
| WP-4.1 (C3) | Admin seam: `AdminHandler` → `AdminService` (role validation, sentinel errors `ErrInvalidRole`/`ErrLastAdmin`/`ErrUserNotFound`) → `AdminRepository` (transactions own order assignment + audit writes); handler maps sentinels to 400/404; unit tests pin role checks without a DB | ✅ DONE |
| WP-4.1 (C4) | Real FKs: `backend/database/migrate_fks.go` — idempotent rebuild migration, 20 FK columns across 14 child tables, `ON DELETE CASCADE`, existing FKs preserved, orphan rows skipped (data preserved); ran against dev `log.db` (11 ghost learner-activity rows cleaned first); erasure map extended to parent links / support issues / learner notes | ✅ DONE |
| WP-4.2 (pool) | `db.go`: `SetMaxOpenConns(1)`/`SetMaxIdleConns(1)`/`SetConnMaxLifetime(0)` — single-writer SQLite, zero `SQLITE_BUSY` churn | ✅ DONE |
| WP-4.2 (indexes) | `daily_activities (learner_id, date)` composite + `announcements (created_at)`; a `completed_at` index was benchmarked and rejected (no plan/runtime change to 200k rows — honest rejection documented); real `EXPLAIN QUERY PLAN` for 10 hot queries in `docs/QUERY_PLANS.md` | ✅ DONE |
| WP-4.2 (budget) | `frontend/scripts/check-budget.mjs` parses real `next build` output (all 15 routes ≤ 174 kB vs 500 kB budget); wired into CI + `make budget`; manual real-device TTI < 5s step in `docs/RELEASE.md` (CI cannot measure it) | ✅ DONE |
| WP-4.3 (metrics) | `backend/internal/metrics`: per-route-pattern counters (`c.FullPath()`, PII-free by construction), public `GET /metrics` (text/plain) + admin `GET /metrics` (JSON), 5xx spike alarm ≥5/60s with last-alert state; `wp43_test.go` proves pattern recording, spike fire/re-arm, PII-free render | ✅ DONE |
| WP-4.3 (analytics) | `analytics` consent type + real withdrawal (`status: granted|withdrawn` on `POST /me/consent`, `privacy.consent_withdrawn` audit); Settings three-state toggle; `GET /api/v1/admin/analytics/summary` aggregate-only over opted-in learners (`avg_score` null when none — never a fabricated 0); `TestAnalyticsSummaryOptInGate`; privacy pack §10.7 | ✅ DONE |
| WP-4.4 (release) | `VERSION` (0.5.0) + `CHANGELOG.md` rewritten: phase sections 0.1.0 → 0.5.0 tied to WP ids; `docs/RELEASE.md` procedure incl. the mandatory manual TTI check and rollback notes | ✅ DONE |
| WP-4.4 (docs) | `docs/README.md` docs hub (architecture, operations, constraints); `CONTRIBUTING.md` for humans and agents | ✅ DONE |

**Evidence for the C4 migration run:** `migrate_fks_test.go` (cascade enforcement, idempotency, anonymized columns stay unconstrained, orphan skip), database suite green on a copy of the real `log.db`. The dev DB itself was migrated at startup (FKs verified via `PRAGMA foreign_key_list`), after removing 11 orphan learner-activity rows that earlier tests left behind (ghost users with empty emails — the unique `users.email` index is now exposed, so future leaks fail loudly).

**Backlog (carried forward):** `Course.Enrolled` seed field · recharts dynamic import · per-route budget tuning beyond 500 kB if a real device misses TTI.

## ⚡ Phase 4.5 — Live-Stack Verification & Dark Mode (2026-08-20)

The Phase 4 line items were proven against the **running Docker stack** (backend `:6101`, frontend `:6100`) instead of unit tests alone — `scripts/live_stack_test.py` (L1–L13 API + W1 frontend routes + W2 real-browser Playwright run, `executable_path=/usr/bin/chromium`, venv `/tmp/logvenv`). 79/79 checks pass. The live tests found and fixed two real bugs:

| WP | What shipped | Verdict |
|---|---|---|
| WP-4.5 (rate limiter) | `rateLimiter.allow()` used the global `rateLimitMax`/`rateLimitWindow` constants instead of `rl.limit`/`rl.window` — every per-route budget silently collapsed to 4 requests/min (`RateLimitLogin=10`, `RateLimitRequestOTP=5`, etc. were dead config). Fixed in `backend/internal/handler/middleware.go`; regression tests `TestRateLimiterHonorsPerRouteBudget` + `TestRateLimiterWindowResets`; live check proves the 10/min login budget resets across windows | ✅ DONE |
| WP-4.5 (register) | `POST /api/v1/auth/register` answered 404 — the email/password register route was dropped in the backend rewrite while the frontend Register tab kept calling it. Restored through the service seam: `AuthService.Register` (validate → `ErrEmailTaken` 409 → bcrypt → STUDENT account, no invented phone) + `AuthHandler.Register` (audit `auth.register` with empty user id on public routes — `audit()` helper hardened against the nil assertion) + rate-limited route in `main.go`. Tests: 201+token / 409 / 400; live suite registers a fresh account and proves the consent gate on it (403 `consent_required` → grant → enroll/complete) | ✅ DONE |
| WP-4.5 (dark mode) | App re-skinned to the LOG palette in **both** themes (AGENTS.md §2a is the source of truth): dark = brand navy `#0B1220` canvas (no more OLED black), `#60A5FA`/`#3B82F6` blue, `#2DD4BF` teal, `#FBBF24` amber; light = official palette (`#F8FAFC` bg, `#0F172A` navy text, `#2563EB`/`#0D9488`/`#F59E0B`). Done via token remap + literal-utility override layer in `globals.css` (`html:not(.dark)`) plus `--glow-rgb`/`--teal-rgb`/`--amber-rgb` vars — the legacy cyan/magenta glow (`0,240,255`, `#00B4D8`, `#FF0070`, `#7000FF`, `#FFB703`) was removed from every component and chart (score `#2563EB`, accuracy `#0D9488`, engagement `#F59E0B`; goal ring = fixed navy badge `#0B1220` + `#60A5FA`). `ThemeToggle` (was login-only) added to `Navigation`; Settings gained an Appearance card (`role="switch"`, `aria-label="Toggle dark mode"`). W2 browser checks: default dark → toggle light (class + localStorage `theme=light`, offline-ready) → toggle back; Settings switch reflects and persists across navigation; computed-style probes verify LOG hexes per mode (btn `#2563EB`/white label on light, `#3B82F6`/white on dark, navy text on light) | ✅ DONE |

| WP-4.5 (nav + de-glow) | Navigation rebuilt after user feedback ("items clustered, blue hover glow"): desktop is a `1fr auto 1fr` grid — logo \| truly-centered pill group (labels `2xl`+, tooltips below) \| utilities (lang/theme/logout); phone gets an evenly-distributed 6-target bottom bar (`flex-1` items, no scroll clustering) plus a settings gear in the top row. Every glow-on-hover was stripped app-wide (~35 sites): `hover:shadow-glow`, icon `drop-shadow-[0_0_*]`, ambient card blooms; `glow`/`glow-strong` shadows redefined as subtle elevation halos and `.card-glow:hover` capped at 0.55 opacity — feedback is now border/background tints and lifts. Off-brand `#FF003C` insight icon → brand teal. Verified: computed probes show nav/link `boxShadow: none`, pill group center offset **0.0px** in both modes, mobile bar = 6 equal-width targets; live suite 79/79 | ✅ DONE |

| WP-4.6 (nav v2) | Second nav rebuild after user feedback, this time research-driven (8 parallel agents: Canvas/Moodle/Coursera/Udemy/Open edX anatomy, MD3/HIG mobile bars, WCAG 2.2 targets+contrast, APG disclosure pattern, Next.js App Router menu patterns). Desktop = `1fr auto 1fr` grid: logo far-left \| exactly 5 learner links dead-centered (probes: 0.0px offset) \| lang + theme + avatar; **logout removed from the bar** and pinned LAST in the new `AccountMenu` disclosure (identity header w/ initials avatar + role chip → Settings/Support → role-gated Parent/Moderator/Admin → Log out), with `aria-expanded`, Escape-closes-and-refocuses, pointerdown-outside close, route-change close, 44px targets. Mobile = logo + avatar top row (utilities inside the menu) and a 5-tab bottom bar (`flex-1 min-w-0`, equal 76px widths, ≥48px targets). Fixed a latent bug: `backdrop-filter` on `<nav>` was the containing block for the `position:fixed` bottom bar, pinning it under the header — bar now renders outside `<nav>`. Verified by an 18-check Playwright probe (`/tmp/opencode/nav_probe2.py`): all pass; live suite 79/79 | ✅ DONE |

**Verification:** backend `go build`/`go vet`/`go test ./...` green + `golangci-lint` 0 issues; frontend `tsc --noEmit` clean, `next lint` 0 warnings, 113 jest tests, budget gate PASS. Live suite: 79/79 (`RESULT: 79/79 checks passed, 0 failed`). Nothing committed — working tree carries the WP-4.5 delta for review.

**Backlog (carried forward):** same list as Phase 4 + per-route limiter budgets validated against real device traffic.

---

## ⚡ Phase 2 Revalidation (2026-08-19)

Second hardening pass — auth/rate-limit seam, OTP lifecycle, JWT claims, export sanitization, read-scoping, and the offline-sync UX. Full evidence + before/after diagrams in `docs/architecture-review-2-20260818.html`. Statuses below are live verdicts.

| # | Finding | Verdict | Evidence / fix location |
|---|---|---|---|
| B1 | Shared auth rate-limit bucket across all `/auth/*` routes | ✅ FIXED | per-route limiters: `RateLimitLogin=10`, `RateLimitRequestOTP=5`, `RateLimitVerifyOTP=20`, `RateLimitPassword=10` + `NewLimiter`/`RateLimitMiddlewareWith` (`backend/internal/handler/middleware.go`, `main.go`) |
| B2 | RequestOTP deleted the live OTP; no cooldown | ✅ FIXED | `ErrOTPCooldown` sentinel → 429; only expired/near-expiry records deleted (`auth_service_impl.go`, `auth_handler.go`); test `TestRequestOTPCooldownReturns429` |
| B3 | JWT accepted without `exp` / `jti` | ✅ FIXED | `jwt.WithExpirationRequired()` + non-empty jti (401 "Invalid token claims") (`middleware.go`); LogoutHandler exp float64 guard + `auth.logout` audit records `jti=` |
| B4 | CSV export formula injection | ✅ FIXED | `sanitizeCSVCell` (`'` prefix on `= + - @` and tab/CR) + UTF-8 BOM (`export_handler.go`); test `TestStudentsCSVExportSanitizesFormulaCells` |
| B5 | Teacher reads not scoped to own classes | ✅ FIXED | reads scoped by caller `teacher_id` (`learner_service.go`); unenroll non-member → 404 `ErrNotEnrolled` (`class_handler.go`); tests `TestAssignmentReadsScopedToTeacher`, `TestUnenrollNonMemberReturns404` |
| B6 | Google token error leaked provider detail | ✅ FIXED | generic `"Invalid Google token"`; detail via `slog.Warn` server-side only |
| B7 | OTP restore kept the old role | ✅ FIXED | role reset to `RoleStudent` on soft-delete restore (`auth_service_impl.go`) |
| B8 | CreateActivity error shape + hygiene | ✅ FIXED | `RespondError` conversion, import cleanup (`admin.go`); gofmt (`moderator_repo.go`) |
| B9 | db.go over-claimed FK enforcement | ✅ FIXED | comment now honest: `_foreign_keys=on` honored but AutoMigrate creates no FK constraints — service layer enforces |
| B10 | Roster tests asserted the pre-fix fabricated roster | ✅ FIXED | `c.Set("userID","mod-1")` stubs; needs-attention scoped to teacher classes; seed students enroll into `cls-1` (`handlers_test.go`, `roster_test.go`) |
| F1 | Sync queue only flushed on `online` event | ✅ FIXED | flush on window `load`, login/register/Google auth, `.logsync` import, SyncIsland tap, CommandPalette force (`api.ts`, `login/page.tsx`, `dashboard/page.tsx`, `SyncIsland.tsx`, `CommandPalette.tsx`) |
| F2 | Auth requests queued to IndexedDB in plaintext | ✅ FIXED | `queueRequest` rejects `/auth/*`; legacy queued auth entries dropped at flush (`api.ts`) |
| F4 | Mutations never invalidated the GET cache | ✅ FIXED | `invalidateRelatedCache` + exported `clearApiCache` for class/enroll/announcement/assignment/user/role mutations (`api.ts`) |
| F5 | Logout-all left the local cache | ✅ FIXED | `settings/page.tsx` awaits `clearApiCache()` |
| F6 | 401s left the user half-logged-in | ✅ FIXED | `clearCredentialsAndRedirect()` on 401 + expired token (`api.ts`) |
| F7 | No submitting state on create/enroll/publish | ✅ FIXED | disabled buttons + `Loader2` in `AssignmentManager`, `ClassManager`, `AnnouncementComposer` |
| F8 | Queued offline writes claimed committed success | ✅ FIXED | queued-aware toasts ("saved offline · will sync") across all write forms + dashboard submit |
| F9 | Command palette dead route + fake sync | ✅ FIXED | `/learning-journey` → `/learning`; Force Data Sync flushes with toasts (`CommandPalette.tsx`) |
| F10 | AssignmentManager stale-response race | ✅ FIXED | `requestSeq` ref discards out-of-order responses |
| F11/F14 | Lesson page fabricated demo + fake completion | ✅ FIXED | demo/confetti removed; honest load-error vs empty states (`learning/[id]/page.tsx`) |
| F12 | Unused deps | ✅ FIXED | removed `lottie-react`, `clsx`, `tailwind-merge`, `canvas-confetti` (+types); `jest.setup.ts` confetti mock deleted |
| F13 | A11y labels | 🔶 PARTIAL | aria-labels on ClassManager/AnnouncementComposer/AssignmentManager/SyncIsland; contrast (teal/amber ≈ 2:1) deferred to token-level backlog (C6) |
| F14 | Error states masquerading as empty (ClassManager, AuditLogTable) | ✅ FIXED | distinct error banners + "unavailable" table text; jest suites updated |
| F15 | Dead buttons (RosterOverview Message, admin Create Activity/Manage Classes) | ✅ FIXED | removed or wired to real scroll targets |
| F16 | CSV download revoke raced the download | ✅ FIXED | anchor append + deferred `revokeObjectURL` (`admin/page.tsx`) |
| F17 | Observation page coupled two independent fetches | ✅ FIXED | `Promise.all` removed; `activity_data` array-guarded |
| M8 | Zero-value dates rendered as "1/1/1" | ✅ FIXED | `formatDueDate` guards `NaN` + year < 2000 → "No deadline" (`AssignmentManager`, dashboard) |
| — | Courses page fabricated enrollment count | ✅ FIXED | `enrolled` removed from UI (`courses/page.tsx`); backend seed field → backlog C5 |

**New tests:** `TestRequestOTPCooldownReturns429`, `TestAssignmentReadsScopedToTeacher`, `TestUnenrollNonMemberReturns404`, `TestStudentsCSVExportSanitizesFormulaCells`, `TestSyncBulkImprovingReplayKeepsBestScore` (backend); AuditLogTable/ClassManager error-vs-empty suites updated (frontend 48 total).

**Backlog (see HTML doc §candidates):** audit-log pagination · unify completion_repo/sync_repo completion logic · admin.go service seam · real FK constraints · backend `Course.Enrolled` seed · brand contrast tokens · recharts dynamic import.

---

## 0. What's Already Strong (Keep as the Pattern)

- **Security posture:** Comprehensive HTTP security headers, 4 MB request body cap (`backend/main.go:37-78`), custom recovery middleware that never leaks stack traces, graceful shutdown, HMAC-signing-method enforcement, `jti` revocation blocklist, bcrypt-14 OTP hashing, per-IP rate limiter with background entry sweeper (`backend/api/auth.go`).
- **Guidance integrity:** Observations/guidance are generated deterministically server-side from real parameters — no external LLM integration, respecting the "No Hallucinations" principle in AGENTS.md.
- **Observability:** Per-request IDs + structured audit logging with duration, user ID, and IP (`backend/api/auth.go:442-469`).
- **Offline-first foundations:** IndexedDB `api-cache` with TTL + stale handling, `sync-queue` with optimistic 202 responses, exponential backoff, legacy cache-shape migration path, `.logsync` sneakernet export/import, PWA manifest, offline banner (`frontend/src/lib/api.ts`, `frontend/src/lib/syncExport.ts`).
- **Real-data refactor:** Courses, micro-modules, learning journey, admin analytics, and moderator roster now derive from live DB queries (most endpoints).
- **Docs:** 7-file markdown suite in `docs/` plus generated DOCX deliverables, updated README, CHANGELOG.
- **Builds:** Multi-stage frontend Dockerfile (non-root `nextjs` user), standalone Next output, tests green on both stacks.

Two audit premises were disproven against the code and are therefore NOT in this report:
- There is no "ngram JSON-in-column" time-series parsing — `GetChartData` reads a proper `DailyActivity` table (`backend/api/handlers.go:299-324`).
- The root `README.md` is fully updated (not create-next-app boilerplate).

---

## 1. CRITICAL (Fix Before Any Demo or Production Use)

### 1.1 GoogleAuth — trivially forgeable identity, full account takeover
- **Evidence:** `backend/api/auth.go:246-276`. DTO accepts only `{email, name}` with no Google `id_token`, no issuer/audience/JWKS verification. `DB.First(&user, "email = ?", req.Email)` then mints a JWT with the **stored user's role**.
- **Impact:** `POST /api/auth/google {"email":"admin@log.edu"}` returns an ADMIN token. The seeded principal account (`backend/database/db.go:56`) is hijackable, granting full `/api/admin` access including `UpdateUserRole` self-escalation.
- **Fix:** Require a verified Google-signed `id_token` (JWKS + issuer + audience + `hd` allowlist), exchanged server-side for the email. Never trust client-sent identity. Alternatively remove the endpoint until the SDK flow is implemented.

### 1.2 JWT signing secret is publicly committed
- **Evidence:** `docker-compose.yml:13` commits `JWT_SECRET=this-is-a-secure-secret-for-compose-deploy-32-chars`; with no `.env` files in the repo, the dev fallback `"dev-only-change-me-in-production-min-32-chars!"` (`backend/api/auth.go:31`) is also a known constant.
- **Impact:** Anyone who can read the repo can mint `{role: "ADMIN"}` tokens — bypassing the entire multi-tier RBAC documented in `docs/SECURITY_AND_RBAC.md`.
- **Fix:** Generate the secret at deploy time (`openssl rand -base64 48`), remove the committed value, make startup fatal when unset, add `jwt.WithValidMethods(["HS256"])`, and document rotation.

### 1.3 Activity status is global, not per-learner — the core data-model flaw
- **Evidence:** `Activity` has a global `Status` column and no learner field (`backend/models/models.go:36-48`). `CompleteActivity` (and `SyncBulk`) unconditionally set `activity.Status = "Completed"` (`backend/api/handlers.go:115`, `:365`). Every reader surfaces the same global value: `GetLearningJourney` (`:58-62`), `GetDashboard` (`:34`), `GetMicroModules` (`:74-86`), moderator `assignments_due` (`:277`).
- **Impact:** Learner A completing act-1 changes what learners B and C see on their journeys/dashboards; completion states and roster percentages become meaningless. Violates the "metrics derived from actual learner data" principle.
- **Fix:** Introduce a per-learner join table, e.g. `LearnerActivity {LearnerID, ActivityID (PK), Status, CompletedAt, Score}` with `Activity` holding only catalog data; derive journey/dashboard/roster from the join. Unique constraint on `(LearnerID, ActivityID)` for idempotency.
- **✅ FIXED + WP-1.1 (RC-01) status model:** the join table landed earlier; Phase 1 adds the canonical per-learner vocabulary and needs-practice determination. Statuses: `not-started` / `active` / `needs-practice` / `completed` (`domain.Status*` constants, `backend/internal/domain/domain.go`), with `ResolveActivityStatus` normalizing legacy rows for reads. **Needs-practice determination rule** (documented in `completion_repo.go`): a completion with quiz data below `NeedsPracticeAccuracyThreshold` (70% accuracy) is flagged `needs-practice` — supportive framing, never "failed"; an improving re-attempt crossing the threshold clears the flag; equal/lower replays are idempotent (never double-bump progress); quiz-less completions stay `completed` (no accuracy signal to judge). Applied identically in the online completion path (`completion_repo.go`) and the offline bulk-sync path (`sync_repo.go`). Status-transition tests: `backend/internal/handler/status_transition_test.go` (start → completed, needs-practice flagging, flag clearing, below-threshold persistence, idempotent replays, API canonical-status exposure, legacy-row resolution).

### 1.4 Logout redirect trap — users can never reach /login again
- **Evidence:** `AuthContext.logout` calls `serverLogout()` which clears localStorage + IndexedDB but **never clears the `log_token` cookie** (`frontend/src/lib/api.ts:52-74`). `setTokenCookie(null)` exists (`frontend/src/context/AuthContext.tsx:59-68`) but is only ever called with a token. `frontend/src/middleware.ts:44-47` redirects `/login` → `/dashboard` whenever the cookie is present.
- **Impact:** After any logout or session-expiry (`AuthContext.tsx:91-95`, `api.ts:96-102`), the user is stuck in a redirect loop; only manually deleting the cookie escapes it.
- **Fix:** Call `setTokenCookie(null)` in `AuthContext.logout` (and in `api.ts`'s local-cleanup branch), or have middleware validate token presence + expiry instead of mere existence.

### 1.5 OTP flow: no per-phone brute-force protection; plaintext OTP in logs
- **Evidence:** Rate limiter is per-IP only, shared across the whole `/api/auth` group (`backend/main.go:86-93`, `auth.go:59-63`); gin 1.12 trusts all reverse proxies by default, so spoofed `X-Forwarded-For` bypasses it (`auth.go:107`). `VerifyOTP` has no per-phone failure counter and only invalidates on success (`auth.go:225`). The OTP is emitted in plaintext via `slog.Info("[DEMO] OTP generated", ..., "otp", otp)` (`auth.go:180`).
- **Impact:** The 6-digit space (1e6) has no effective online attempt cap, and a leaked `otp_records` table is offline-brute-forceable. Clear-and-present takeover of any phone number.
- **Fix:** Per-phone attempt counter with invalidation (e.g., 5 failures → expire), keyed limiter (IP+phone, per-route), bind `ClientIP` to `RemoteAddr` unless trusted-proxy middleware is added, stop logging OTPs, add per-minute re-request cooldown.

### 1.6 Service Worker caches per-user authorization responses
- **Evidence:** Default next-pwa workbox runtime caching includes a cross-origin `NetworkFirst` route (cache `"cross-origin"`, `maxAgeSeconds: 3600`). The app's API base is always cross-origin (`frontend/src/lib/api.ts:43`), so every authenticated GET (`/dashboard`, `/moderator/roster`, `/admin/dashboard`) is SW-cached **keyed by URL only, with no auth identity**, and survives logout (logout only clears IndexedDB, not the SW cache).
- **Impact:** On shared browsers (common in Nepali schools), the next user inherits the previous user's cached API data — a cross-account data leak that defeats the platform's own logout cleanup intent.
- **Fix:** Configure `runtimeCaching` in `frontend/next.config.mjs` to exclude the API origin (or have `fetchWithCache` use `cache: 'no-store'` for GETs so the SW never intercepts). The IndexedDB layer already provides the offline guarantee.

### 1.7 Sync queue silently deletes records on 401; completions not idempotent
- **Evidence:** Queued requests store the snapshot headers incl. the JWT (`api.ts:188-193`); on flush a 401/4xx → `db.delete(QUEUE_STORE, req.id)` and only a `console.error` (`api.ts:245-262`). An offline user with an expired (72h) token loses every queued record with no warning. Additionally `CompleteActivity` has no "already completed" guard — repeats double-bump streak/score and spawn duplicate observations/guidance (`backend/api/handlers.go:111-139, 149-173`), so a flush retried after a timeout-but-successful server write corrupts state.
- **Fix:** Re-attach a fresh token at flush time; on 401 **pause the queue and prompt re-login while preserving records**; make completion idempotent server-side (return 200 with no side effects if already completed, or rely on the 1.3 unique constraint).

### 1.8 Tracked build artifacts in git
- **Evidence:** `frontend/public/sw.js` and `frontend/public/workbox-4754cb34.js` are next-pwa **generated artifacts** (build-hash precache lists) and are tracked in git (confirmed via `git ls-files`). `frontend/next.config.mjs:3-8` sets `dest: 'public'` + `skipWaiting: true`.
- **Impact:** Every `next build` dirties VCS; a served stale `sw.js` with `skipWaiting` activates immediately and 404s the new build's chunks; dev builds (`disable: true`) leave the last production SW lying in `public/`.
- **Fix:** Set `dest: '.next'` and gitignore `frontend/public/sw.js` + `workbox-*.js`; never hand-edit the generated file; verify precache consistency in CI.

---

## 2. HIGH

### 2.1 JWT role not revalidated against the DB
- Middleware validates the signature and claims but never loads the user: demoted, soft-deleted, or disabled users keep working 72h tokens (`backend/api/auth.go:356-435`). Ghost users can still write observations/guidance/progress because `CompleteActivity`/`SyncBulk` never look up the user while `GetDashboard` 404s for them (`handlers.go:29` vs `:112`).
- **Fix:** Load the user in middleware (or lazily per handler), compare DB role vs token role, drop the token on mismatch.

### 2.2 Remaining fabricated data (violates "No Hallucinations")
- Dashboard `dailyGoalPercentage = 75` hardcoded (`frontend/src/app/dashboard/page.tsx:49`).
- Moderator fallbacks `?? 124`, `"↑ 12% from last week"`, `?? 8`, `?? 3` and 4 fabricated student rows (`frontend/src/app/moderator/page.tsx:82-94, 112-117`).
- Chart Mon–Sun zero series when empty (`backend/api/handlers.go:311-321`) and seeded `DailyActivity` rows only for `user-123` with **no live write path** — real completions never create rows, so trends can never reflect actual learning (`backend/database/db.go:127-138`).
- **Fix:** Remove fake fallbacks → explicit "no cached data" states; write a `DailyActivity` row on every completion/score event; compute the goal from real data or drop it.

### 2.3 Streak/score math is date-blind
- `Progress` has no `LastActivityDate`/timestamps (`backend/models/models.go:60-66`); `CurrentStreak++` and `OverallScore += 2.5` are unconditional (`handlers.go:131-138`, `:383`). Same-day double completion doubles a streak; consecutive-day logic doesn't exist. Seeded `TotalTopics: 10, Completed: 2, Streak: 3, Score: 85.5` is internally inconsistent with the 2-3 seeded activities (`backend/database/db.go:65`).
- **Fix:** Store `last_activity_date`; increment only when `today - last == 1 day`, reset to 1 otherwise; seed consistently.

### 2.4 `SyncBulk` swallows errors and discards learner data
- Transaction wrapper exists but per-item `Save`/`Create` errors are ignored; endpoints always return 200 (`backend/api/handlers.go:356-426`). No payload size/item cap beyond the global 4 MB, no checksum, no idempotency key. The learner's request `Body` is parsed but never persisted → quiz correctness never reaches the server.
- **Fix:** Validate item count/size, propagate per-item results into an aggregate response, add idempotency keys, extract and store real answer/score data.

### 2.5 Moderator roster is half real, half fabricated
- ~~Real users/progress are queried, but `class_name: "Logic 101: Discrete Structures"` is hardcoded, `last_active` uses `user.updated_at`, N+1 per-student queries, and the status is a binary streak==0 heuristic.~~ RESOLVED: `class_name` derives from the real first course; `last_active` uses real `UpdatedAt`; the N+1 is gone — `ModeratorRepository.GetRoster` returns `[]RosterEntry{User, Progress}` built from a roster-page query plus one batched `IN` progress lookup (`moderator_repo.go`), and the moderator service holds only the moderator repo (`NewModeratorService(m)`).
- **Fix (implemented):** batch-load progress inside the repository so the per-learner loop never crosses the seam; regression test `TestRosterQueryCountIsConstant` proves the query count is identical for 25 vs 50 students.

### 2.6 Rate limiter lockout tension
- 5 req/min per-IP bucket over the whole auth group locks out an entire school behind one NAT after ~2-3 students (UX), while an attacker rotating XFF has no limit (DoS). (`backend/main.go:86-93`)
- **Fix:** Require trusted proxies (or trust loopback only), move to a keyed limiter (IP+phone, per-route), raise the window.

### 2.7 Incomplete frontend states
- No empty/offline/retry states on guidance, observation, learning-journey, and admin pages; offline completion shows "Lesson marked as completed!" for a queued 202 (`learning/[id]/page.tsx:84-95`); no optimistic UI or pending-sync honesty; no `disabled` during POST → double-click double-POST risk.
- **Fix:** Honest queued/pending states, retry buttons, consistent skeletons, `getSyncQueueCount` badge surfaced on relevant pages.

### 2.8 Dead code & dead UI inventory
- `frontend/src/lib/adaptiveEngine.ts` (`evaluateLocalAdaptivity`, `LocalGuidance`) — zero importers.
- `ForgotPassword` stub returning canned success with no reset mechanism (`backend/api/auth.go:230-244`).
- Mock Google login posting hardcoded `'learner@gmail.com'` (`frontend/src/app/login/page.tsx:73`).
- Dead buttons: moderator "Create Assignment", admin "View All / Create Activity / Send Broadcast / Manage Roles", courses "Filters".
- Dead `useAuth()` with unused return (`frontend/src/app/courses/page.tsx:24`); dead `?redirect=` param (`frontend/src/middleware.ts:40`); dead `constantTimeEqual` (`backend/api/auth.go:134-136`); `/api/moderator/classes` placeholder returning a static message (`backend/main.go:123-125`).
- **Fix:** Wire, implement, or delete each item.

### 2.9 Edge RBAC guard checks token presence only
- `frontend/src/middleware.ts:34-42` only checks token existence; role enforcement happens client-side post-hydration (`admin/page.tsx:20-24`, `moderator/page.tsx:25-29`). A STUDENT token passes the edge guard and can read protected HTML en route.
- **Fix:** Decode the JWT `role` claim in middleware and redirect at the edge (the app already base64-decodes JWTs client-side elsewhere).

---

## 3. MEDIUM

- **SQLite pragmas:** `gorm.Open(sqlite.Open("log.db"))` — no `_journal_mode=WAL`, no `_busy_timeout`, no `_foreign_keys=on`; Info-level query logging in production (`backend/database/db.go:18-19`).
- **Blocklist cleanup** runs once at startup only; rows accumulate and every request hits the list (`backend/database/db.go:45`). → periodic sweeper goroutine like the auth limiter's.
- **Control-plane hardening:** `UpdateUserRole` has no last-ADMIN/self-demotion guard (`backend/api/admin.go:71-99`); `CreateActivity` `Order = count+1` is racy and unconstrained (`admin.go:120-132`); `ContentJSON` is an unvalidated free string — stored-XSS vector (`admin.go:109, 130`); `Prerequisites` unvalidated.
- **Input validation gaps:** `RequestOTP` phone has no E.164/Nepal format check (`auth.go:144`); `VerifyOTP` phone unvalidated beyond `required` (`auth.go:186`); OTP input has no client-side length validation (generic "Login failed." on short OTP, `login/page.tsx:61`).
- **Tests use the real seeded DB:** `TestMain` does `os.Chdir("..")` + `InitDB()` (`handlers_test.go:21-23`); a failing test pollutes `backend/log.db`. → in-memory/temp-file SQLite per test.
- **Coverage gaps:** auth middleware RBAC matrix, blocklist/expiry, GoogleAuth, OTP flow, admin handlers, rate limiter, empty-DB `GetDashboard`, sync idempotency/conflicts, component tests (0 today).
- **Accessibility:** white-on-brand teal `#00B4D8` ≈ 2.5:1 and white-on-amber `#FFB703` ≈ 1.9:1 (WCAG AA fail; `frontend/src/app/globals.css:12-14`, `learning/[id]/page.tsx:260-262`, `moderator/page.tsx:89`); no `prefers-reduced-motion` handling for framer-motion/skeleton/confetti; unlabeled InstallPrompt close button (`InstallPrompt.tsx:80-85`); unlabeled import file input (`dashboard/page.tsx:191-206`); no bottom-padding compensation for the fixed bottom nav (`layout.tsx:30`).
- **Bundle size:** recharts statically imported in the observation route chunk (~100-200KB, `observation/page.tsx:7`). → `next/dynamic`.
- **Manifest mismatch:** declared 192/512 maskable while the actual file is 1254×1254 and not maskable-safe; `start_url: "/dashboard"` means offline cold start for a logged-out user fails until one online visit (`frontend/public/manifest.json:6-21`).
- **Redundant listeners/polling:** 3 separate online/offline listeners + a 5s IndexedDB poll mounted globally (`api.ts:18-27`, `OfflineBanner.tsx`, `Navigation.tsx:20-28`, `useSyncQueue.ts:19`) — battery/CPU cost on low-end Android. ✅ FIXED (WP-0.4) — `OfflineBanner.tsx` deleted (unmounted dead code), Navigation consolidated to SyncIsland, `useSyncQueue` is now event-driven (`log:queue-changed` after enqueue/flush in `api.ts`, `online`, `visibilitychange`) — zero constant polling.
- **Queue dedup ignores body:** `queueRequest` collapses distinct payloads to the same endpoint+method (`api.ts:177-184`).
- **Route params:** `params?.id as string` can be `string[]`; hardcoded `'act-2'` fallback masks bad URLs (`learning/[id]/page.tsx:47`).
- **CORS/trusted proxies:** no `Vary: Origin`, no explicit `TrustedProxies` config (`backend/main.go:56-62`).
- **Rebuild artifacts:** `backend/log.db`, `sw.js`, `workbox-*.js`, and `backend/server` are all tracked in git — no root `.gitignore`.
- **Inconsistency:** dashboard logout button bypasses AuthContext state (`dashboard/page.tsx:68`) while nav logout uses context (`Navigation.tsx:89`).

---

## 4. Product Roadmap (Nepal Lens)

1. **Per-learner enrollment/progress model** — fixes 1.3 and unlocks real per-learner charts, roster drill-down, and teacher analytics. Highest single change.
2. **Capture real quiz correctness** — learner sends score/answers; guidance derives from *actual* attempt/accuracy/elapsed time. Eliminates the `+2.5` fabrication (2.3) while keeping deterministic, AGENTS.md-compliant guidance.
3. **Nepali localization** (`next-intl`) for student-facing strings — core adoption enabler for students with limited English literacy.
4. **Real teacher workflow** — `/api/moderator/classes` is a placeholder; add assignment creation, per-student guidance replies, class management, and the student↔teacher loop (currently one-directional).
5. **Reconnect digest** — "3 new guidance notes since your last visit," leveraging the existing offline-first cache to serve the low-bandwidth periodic-check-in pattern.
6. **SMS gateway for OTP** — today the only way to log in is reading server logs (demo OTP only, `auth.go:180`); field authentication is impossible without a provider.
7. **Device-sharing safety** — 1.6 fix plus "log out all devices" (revoke all jti for a user).

---

## 5. Engineering Process

- `.env.example` for `backend/` and `frontend/`; remove committed compose secret → generate at deploy time.
- CI (GitHub Actions): `go vet` + `go test` + `go build`, jest, `next build`, eslint, docker build on PR.
- Makefile: add `test-backend`, `lint`, `docs`, `build` targets (currently only dev/test-frontend/up/down/logs/clean exist).
- Health endpoints: `/healthz` (DB reachability — today `/api/ping` returns pong even with a dead DB) and `/readyz`; optional metrics endpoint.
- golangci-lint config; frontend `typecheck` script.
- Test isolation: in-memory SQLite per test; add auth-middleware matrix, admin handler, rate limiter, and sync-conflict tests.
- Git hygiene: gitignore `sw.js`, `workbox-*`, `log.db`, `.env*`, `backend/server`; keep docs committed.

---

## 6. Suggested Execution Order

| Phase | Scope | Items |
|---|---|---|
| **Phase 0 — Blockers** | Identity & auth security | 1.1 GoogleAuth, 1.2 JWT secret, 1.4 logout trap, 1.5 OTP hardening, 1.6 SW cache leak, 1.8 git-attifact hygiene |
| **Phase 1 — Data truth** | Model + offline correctness | 1.3 per-learner model, 1.7 idempotency + queue preservation, 2.2 remove fabrications, 2.3 date-safe streaks, 2.4 SyncBulk integrity |
| **Phase 2 — Offline resilience** | Sync UX + fallback states | token refresh at flush, 401-preserving queue, body-aware dedup, honest queued-state UI, 2.7 missing states |
| **Phase 3 — Product** | Value for Nepal | teacher workflows, Nepali i18n, reconnect digest, SMS OTP, device-sharing logout |
| **Phase 4 — Engineering** | Process & hardening | CI, env templates, health checks, test coverage, Docker/DB hardening, a11y, bundle size |