# LOG — Known Bugs & Fix List

> Generated from a full codebase read on 2026-08-16. All items listed below have
> been **fixed** on 2026-08-16 (uncommitted — commit at your convenience). Kept
> as a record of what changed and why; file:line references point at the fixes.

---

## CRITICAL — user-visible breakage

### 1. ~~Courses catalog always renders empty (API key mismatch)~~ ✅ FIXED
- **Fix location:** `backend/internal/handler/learner_handler.go` (`GetCourses`) now returns `{"courses": [...], "pagination": {...}}`; the test `backend/api/handlers_test.go` `TestGetCourses` asserts the `courses` key. Frontend `courses/page.tsx` (`res.courses || []`) and `docs/API_SPECIFICATION.md` §4.6 were already on this contract.
- **Note:** if you have an old-format `/courses` payload cached in IndexedDB it still displays fine; the next successful fetch refreshes it.

---

## HIGH — correctness / security

### 2. ~~Hardcoded `user-123` fallback in handlers~~ ✅ FIXED
- **Fix location:** New `callerID(c)` helper in `backend/internal/handler/learner_handler.go` (returns `(string, bool)`); applied to `GetDashboard`, `GetLearningJourney`, `GetChartData`, `CompleteActivity`, and `SyncBulk` (`sync_handler.go`). Missing/empty `userID` now returns `401 Unauthorized` instead of silently serving demo data.

### 3. ~~`CompleteActivity` is no longer atomic~~ ✅ FIXED
- **Fix location:** New `domain.CompletionRepository` interface (`backend/internal/domain/repository.go`) + `backend/internal/repository/completion_repo.go`. `CompleteActivityTx` runs the learner-activity upsert, progress bump, observation, and guidance inside one `gorm.DB.Transaction` (mirrors `sync_repo.go`). `learnerService.CompleteActivity` delegates to it; wired in `backend/main.go`. Already-completed activities remain an idempotent no-op.
- **Wiring change:** `service.NewLearnerService(u, a, p, l, c)` now takes a 5th arg — update any future call sites.

### 4. ~~`VerifyOTP` response missing `user` object~~ ✅ FIXED
- **Fix location:** `AuthService.VerifyOTP` now returns `(*domain.User, string, error)` (`auth_service.go`, `auth_service_impl.go`); the handler (`auth_handler.go`) emits `{"token": ..., "user": {id, name, email, phone, role, is_verified}}`, matching Login/Register/GoogleAuth and `docs/API_SPECIFICATION.md` §3.2.

---

## MEDIUM — hygiene, tests, dead code

### 5. ~~SQLite database file tracked in git + WAL sidecars unignored~~ ✅ FIXED
- **Fix location:** `.gitignore` now covers `backend/data/log.db*`; `git rm --cached backend/data/log.db` run (file stays on disk). Stage the deletion with your next commit.

### 6. ~~Backend tests pollute the real dev database~~ ✅ FIXED
- **Fix location:** `backend/database/db.go` now honors a `DB_PATH` env var (default `data/log.db`; absolute paths skip the `data/` mkdir). `backend/api/handlers_test.go` `TestMain` sets `DB_PATH` to a fresh `os.MkdirTemp` file and removes it after — the real `backend/data/log.db` is never touched.

### 7. ~~`update_handlers.py` is dead code~~ ✅ FIXED
- **Fix location:** File deleted (referenced `backend/api/handlers.go`/`auth.go`, both gone). Recoverable from git history if ever needed.

### 8. ~~Documentation drift vs the refactored codebase~~ ✅ FIXED
- **Fix location:** Updated `docs/API_SPECIFICATION.md` (base URL `/api/v1`, all routes prefixed, Google OAuth now `{"token": ...}` with server-side id_token verification, added Login/Register/UpdatePassword sections), `docs/SECURITY_AND_RBAC.md` (JWT 24h, bcrypt cost 12, `/api/v1` paths, env table + `DB_PATH`/`GOOGLE_CLIENT_ID`, middleware file ref), `docs/DATABASE_SCHEMA.md` (models → `internal/domain/domain.go`, DB path `backend/data/log.db`), `docs/IMPLEMENTATION_GUIDE.md` (paths + handler workflow), `docs/FRONTEND_GUIDE.md` (email/password login, new components/pages in tree, `/api/v1` endpoints), `docs/ARCHITECTURE.md`, `docs/OFFLINE_SYNC.md`, `DOCUMENTATION.md` (security params, full API table, diagram, /login description), `README.md` (`/api/v1`).
- **Not touched:** `docs/ENHANCEMENT.md` is a historical audit record; `LOG_Implementation_Guide.docx` is never regenerated.

---

## LOW — polish / edge cases

### 9. ~~`/settings` not in edge-middleware protected routes~~ ✅ FIXED
- **Fix location:** `frontend/src/middleware.ts` — `'/settings'` added to `PROTECTED_ROUTES`.

### 10. ~~Moderator roster shows "Jan 01" for users with no activity~~ ✅ FIXED
- **Fix location:** `backend/internal/service/learner_service.go` `GetModeratorRoster` — zero `UpdatedAt` renders `"—"` instead of the year-1 date.

### 11. ~~SyncIsland visual is purely cosmetic~~ ✅ FIXED
- **Fix location:** `frontend/src/components/SyncIsland.tsx` rewritten — driven by `useSyncQueue()` real pending count: shows "Offline — N changes saved" while offline, "Syncing N changes..." while the queue drains, and "Sync Complete" only after the queue actually reaches zero.

---

## Verification (all green, 2026-08-16)
- `go build ./... && go vet ./... && go test ./...` — pass (tests now run on a temp DB)
- `npx tsc --noEmit` — clean; `npx jest` — 11/11 pass

## Already fixed (do not re-do) — verified in code
- GoogleAuth id_token server-side verification — `backend/internal/service/auth_service_impl.go` (ENHANCEMENT §1.1 implemented)
- SW cross-origin cache leak — `frontend/next.config.mjs` filters the NetworkFirst catch-all
- Queue never deleted on 401; re-login prompt + token replay — `frontend/src/lib/api.ts`
- Per-learner activity status — `LearnerActivity` model + `sync_repo.go` upsert
- Rate limiting, JWT blocklist, env-only secrets, RFC 9457 errors, request IDs
- Chart empty state returns honest zeros (`learner_service.go`) — keep this pattern