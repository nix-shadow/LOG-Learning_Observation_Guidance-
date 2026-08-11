# LOG — Research Findings & Improvement Plan

Comprehensive audit of the LOG platform (backend, frontend, offline-sync layer, product, and engineering process) with evidence-based findings and a prioritized improvement plan.

**Status:** Research only — no code was changed as part of this audit.
**Method:** Deep sub-agent audits of the backend & frontend plus manual verification of the offline layer, infrastructure, and product surface. All claims verified against the codebase with `file:line` evidence.

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
- Real users/progress are queried, but `class_name: "Logic 101: Discrete Structures"` is hardcoded (`handlers.go:280`), `last_active` uses `user.updated_at` (never an activity timestamp, stale for everyone) (`handlers.go:272`), N+1 per-student queries (`252-253`), and the status is a binary streak==0 heuristic (`255-259`).
- **Fix:** Track a real `last_active_at` on learner events, batch-load progress, derive class from real data.

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
- **Redundant listeners/polling:** 3 separate online/offline listeners + a 5s IndexedDB poll mounted globally (`api.ts:18-27`, `OfflineBanner.tsx`, `Navigation.tsx:20-28`, `useSyncQueue.ts:19`) — battery/CPU cost on low-end Android.
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