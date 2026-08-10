# Changelog

All notable changes to the LOG (Learning Observation Guidance) platform are documented here.

---

## [Unreleased] — 2026-08-10 (continued)

### 🔄 Real-Data Integrity & UX Wiring
- **Course Catalog from Real API**: `/courses` page now renders `GET /api/courses` (paginated, cached for offline) with dynamic category filters derived from data — removed the 24 fake `Math.random()` courses.
- **Live Micro-Modules in Lesson Player**: `/learning/[id]` fetches `GET /api/activities/:id/modules` and renders the `MicroModuleViewer`; the demo lesson remains only as an offline/catalog fallback.
- **Seeded Micro-Module Content**: `database/db.go` seeds 5 bite-sized modules for `act-1`/`act-2` independently of users, so existing databases receive them on startup.
- **Functional Journey Buttons**: "Continue"/"Review" on `/learning` now navigate to `/learning/[id]`.
- **Progress Upsert for New Learners**: `CompleteActivity` and `SyncBulk` create a `Progress` record on first completion instead of returning 500; progress defaults to the real activity count.
- **No Cross-User Data Leaks**: `GetDashboard` returns `404` when the authenticated learner doesn't exist instead of silently serving `user-123`'s data.
- **Computed Admin Analytics**: `GetAdminDashboard` now derives ActiveDaily (24h recency) and TotalCompletions (sum of progress) from the database instead of `users/2` and `activities*20`.
- **Computed Moderator Flags**: `needs_attention` counts zero-streak learners and `assignments_due` counts in-progress activities (were hardcoded `8`/`3`).
- **SyncBulk Mirrors Online Flow**: bulk sync now generates supportive observations + next-step guidance in the same transaction.
- **Public `/api/ping`**: health probe moved out of the authenticated route group.
- **Adaptive Engine Cache Bug**: fixed mismatched IndexedDB shape (`{data, cachedAt}`) that broke the dashboard UI after local guidance injection.
- **Unified Logout**: `AuthContext` now uses the `api.ts` logout lifecycle — JWT server-side revocation, cookie + storage cleanup, and IndexedDB cache purge before redirect.

### 🔒 Security
- **Register of fixes for docs**: API spec corrected (CORS is origin-restricted, not `*`; Create Activity DTO no longer documents client-injected `id`/`order`), offline docs updated to v3 cache format, security guide updated with `jti` revocation and DoS hardening.

### 🧪 Testing
- **Go tests added**: new-learner completion (progress created), computed `needs_attention` (cross-checked against DB), malformed sync payload rejection; test learners are cleaned up after each run.

---

## [Unreleased] — 2026-08-10

### 🔐 Security Hardening
- **OTP Hashing**: OTPs are now hashed with `bcrypt` before persisting. Plaintext OTPs are never stored in the database.
- **Cryptographically Secure OTPs**: Replaced the hardcoded `"123456"` demo OTP with `crypto/rand`-based 6-digit generation.
- **JWT Revocation**: Introduced a `TokenBlocklist` database model. The new `POST /api/auth/logout` endpoint adds the token's `jti` to the blocklist; `AuthMiddleware` rejects any revoked token.
- **JWT `jti` Claim**: All generated tokens now include a unique `jti` claim for per-token revocation.
- **Strict Input DTOs**: `CreateActivity` now uses a dedicated `CreateActivityRequest` DTO — clients can no longer inject server-managed fields like `id`, `created_at`, or `order`.
- **Role Enum Validation**: `UpdateUserRole` now validates that the role is one of `STUDENT`, `MODERATOR`, or `ADMIN`.
- **Rate Limiter Memory Leak Fix**: Added a background cleanup goroutine that prunes stale IP entries from the in-memory rate limiter every 5 minutes.
- **Request Body Size Limit**: Global 4 MB body limit added via `http.MaxBytesReader` to prevent denial-of-service via oversized payloads.
- **Server Timeouts**: `ReadTimeout: 15s`, `WriteTimeout: 30s`, `IdleTimeout: 120s` added to `http.Server`.
- **Panic Recovery Hardened**: Switched from `gin.Default()` to `gin.New()` + `gin.CustomRecovery()` — server panics now return a clean `500` without leaking internal stack traces.
- **Frontend JWT Expiry Check**: Client-side code now decodes the JWT `exp` claim before sending requests. Expired tokens are cleared and the user is redirected to `/login` automatically.
- **XSS Mitigation**: Token reads from `localStorage` are wrapped in `try/catch`; on any parse failure, credentials are cleared defensively.

### 🏗️ Backend Architecture (Production-Grade)
- **Real Database Models**: Added `Course` and `DailyActivity` GORM models to replace all hardcoded mock data.
- **Auto-Seeding**: `database/db.go` seeds `Course` and `DailyActivity` tables on first startup.
- **`GetCourses`**: Refactored to paginate from the `courses` table with real `COUNT`.
- **`GetChartData`**: Refactored to aggregate `DailyActivity` records per learner from the DB.
- **`GetModeratorRoster`**: Refactored to JOIN `users` with `progress` table — real completion % and streak data.
- **`SyncBulk` (Transactional)**: Now actually processes offline sync payloads in a DB transaction, scoped to the authenticated user's progress.
- **`GetMicroModules`**: New `GET /api/activities/:id/modules` endpoint serving `MicroModule` content for the viewer.
- **Graceful Shutdown**: Server now intercepts `SIGINT`/`SIGTERM` with a 5-second context timeout.
- **`godotenv`**: Environment variables loaded from `.env` file on startup.

### 📦 Offline Resilience (Research-Driven)
- **Sneakernet Sync (`syncExport.ts`)**: Students can export a `.logsync` JSON payload and import it on a connected device for bulk upload via `POST /api/sync/bulk`.
- **Adaptive Learning Engine (`adaptiveEngine.ts`)**: Client-side rule-based engine generates local guidance recommendations from IndexedDB without a network call.
- **Micro-Learning Architecture (`MicroModuleViewer.tsx`)**: New animated component for swipeable, bite-sized content modules optimized for low-bandwidth.
- **Offline Moderator Dashboard**: Teachers can pre-fetch the class roster to IndexedDB for use in disconnected environments.

### 🐛 Bug Fixes
- **TypeScript errors in `syncExport.test.ts`**: Fixed `ProgressEvent<FileReader>` mock casting and removed unused `CACHE_STORE` import.
- **`MicroModuleViewer.tsx`**: Replaced `<img>` with Next.js `<Image unoptimized />` to eliminate LCP build warning.

### ✨ UX Improvements
- **Logout Button**: Dashboard now displays a logout button that calls `POST /api/auth/logout`, clears IndexedDB cache and localStorage, then redirects to `/login`.
- **Frontend `logout()` helper**: Exported from `api.ts` — handles the full revocation and cleanup lifecycle.
