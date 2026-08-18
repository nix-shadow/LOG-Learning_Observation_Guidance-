# LOG — What To Do & Which Skills To Use

Project playbook for the **LOG (Learning Observation Guidance)** platform.
Every task below names the skill(s) to invoke and a ready-to-paste prompt. Skills auto-load — just type the prompt into opencode.

> **How skills work:** say the magic words → the skill loads and drives the work. Force any skill with: *"Use the `<name>` skill to..."*
> Full trigger-word list: `/home/shinra/Windows backups/Work/cheetsheet.md`

---

## Phase 0 — Re-verify the audit (DO FIRST)

`docs/ENHANCEMENT.md` lists 8 CRITICAL + several HIGH findings, but it cites **old file paths** (`backend/api/auth.go`, `backend/models/models.go`, `backend/api/handlers.go`) that no longer exist — auth moved to `backend/internal/`, and the CHANGELOG shows some criticals already fixed (unified logout, computed admin analytics, real-data endpoints). **Do not trust the list blindly.**

| Task | Skills | Prompt |
|---|---|---|
| Re-audit the current code against every finding | `vuln-analysis`, `security-audit`, `architecture` | "Re-run the ENHANCEMENT.md audit against the CURRENT code in backend/internal/. For each of 1.1–1.8 and 2.x, say: STILL BROKEN / FIXED / CHANGED, with file:line. Update the report." |
| Check nothing new regressed | `code-review` | "Review changes since the last commit for security and correctness regressions." |

**Output:** a revalidated findings list. Only work on items still confirmed broken.

---

## Phase 1 — Security CRITICALs (highest priority)

| # | Finding | Skills | Prompt |
|---|---|---|---|
| 1.1 | GoogleAuth forgeable identity (`POST /auth/google {email}` → ADMIN token) | `vuln-analysis`, `security-and-hardening` | "Fix the Google auth hole: require a verified Google id_token (JWKS + issuer + audience + hd) exchanged server-side; never trust client email. Or disable the endpoint until the SDK flow exists. Write a test proving forged emails are rejected." |
| 1.2 | Committed JWT secret (docker-compose.yml) + dev fallback | `security-and-hardening`, `ci-cd-and-automation` | "Remove the committed JWT_SECRET from docker-compose.yml, make startup fatal when unset (verify it already is in main.go), generate at deploy time, add HS256-only validation, document rotation." |
| 1.3 | Activity status is global, not per-learner (core data-model flaw) | `architecture`, `spec-driven-development`, `implement` | "Design the per-learner activity model: LearnerActivity{LearnerID, ActivityID, Status, CompletedAt, Score} with unique (LearnerID, ActivityID). Write an ADR, then implement and migrate readers (journey/dashboard/roster/modules)." |
| 1.5 | OTP: no per-phone brute-force protection, OTP logged in plaintext | `vuln-analysis`, `security-and-hardening` | "Add per-phone attempt limiting with invalidation (5 fails → expire), keyed IP+phone rate limit per route, stop logging OTPs, bind ClientIP to RemoteAddr, add re-request cooldown." |
| 1.6 | Service Worker caches per-user auth'd API responses (cross-account leak) | `bug-fixing`, `security-and-hardening`, `frontend-ui-engineering` | "Stop the next-pwa runtime cache from caching cross-origin authenticated API GETs. Use cache:'no-store' for API GETs or exclude the API origin in runtimeCaching. Keep IndexedDB as the offline layer. Add a test." |
| 1.7 | Sync queue deletes records on 401; completions not idempotent | `bug-fixing`, `test-driven-development` | "On 401 during flush: pause the queue, prompt re-login, preserve records (re-attach fresh token). Make CompleteActivity/SyncBulk idempotent (no double streak/score/observation on retry). Add Jest + Go tests." |
| 1.8 | Tracked next-pwa build artifacts (sw.js, workbox-*.js) | `git-workflow-and-versioning` | "Set next-pwa dest:'.next', gitignore frontend/public/sw.js + workbox-*.js, untrack them, verify precache consistency in CI." |

---

## Phase 2 — HIGH items

| # | Finding | Skills | Prompt |
|---|---|---|---|
| 2.1 | JWT role not revalidated against DB (demoted users keep 72h tokens) | `vuln-analysis`, `architecture` | "Load the user in AuthMiddleware, compare DB role vs token role, drop the token on mismatch. Add a test for a demoted user." |
| 2.2 | Fabricated data remains (dailyGoalPercentage=75, moderator `?? 124` fallbacks, fake student rows, zero-series chart, no DailyActivity write path) | `architecture`, `frontend-design`, `implement` | "Remove all fake fallbacks → honest 0/empty states. Write a DailyActivity row on every completion/score. Compute the dashboard goal from real data or drop it. Verify no fabricated numbers remain in frontend/src/app." |
| 2.x | Read the rest of ENHANCEMENT.md (lines 80–167) | — | "Summarize remaining HIGH/MEDIUM items and turn them into tickets." |

---

## Phase 3 — My architecture-review findings (verified against current code)

From the live architecture review — all confirmed with `file:line`:

| # | Finding | Skills | Prompt |
|---|---|---|---|
| A1 | `backend/api/admin.go` lives OUTSIDE `internal/` — admin handlers have a second home; main.go imports both `api` and `internal/handler` | `architecture`, `code-review` | "Move backend/api/admin.go into internal/handler (or rename the package) so there is one handler home and one import path." |
| A2 | `/api/v1/moderator/classes` returns a hardcoded stub `{"message":"Moderator classes data"}` (main.go:169-171) — violates the No-Fabricated-Fallbacks principle | `architecture`, `implement` | "Implement /moderator/classes against moderatorRepo, or remove the route until it is real." |
| A3 | AGENTS.md doc drift — §4 references `api/auth.go` (HashPassword is now `internal/service/auth_utils.go:35`); §5 env-copy steps stale | `documentation`, `spec-to-code-compliance` | "Update AGENTS.md to match the current internal/ layout and run instructions. Verify every path and command it references actually exists." |
| A4 | `ThreeBackground.tsx` (Three.js canvas) mounted in layout.tsx — heavy 3D + framer-motion conflicts with the low-connectivity/low-end-device constraint | `performance-optimization`, `frontend-design` | "Evaluate ThreeBackground cost. Either lazy-load it, gate it behind a pref/device check, or drop it. Measure bundle-size impact before/after." |
| A5 | Backend test coverage thin — only `backend/api/handlers_test.go` + `frontend/__tests__/api.test.ts` | `test-driven-development`, `property-based-testing` | "Add Go tests for auth middleware, role checks, sync idempotency, and OTP limiting. Add Jest tests for fetchWithCache TTL/invalidation and sync-queue flush." |

---

## Phase 4 — Polish & UX

| Task | Skills | Prompt |
|---|---|---|
| Make the app feel high-end | `impeccable`, `frontend-design`, `ui-ux-pro-max` | "Run impeccable on the dashboard and login pages. Improve visual hierarchy, spacing, and empty/offline states. Keep it lightweight for low-end devices." |
| Improve offline UX | `frontend-ui-engineering`, `webapp-testing` | "Review the offline banner, sync island, and InstallPrompt. Make offline state obvious and recoverable. Test the full offline → online flow." |
| Micro-interactions & motion | `animate`, `gsap-core`, `motion-design` | "Add tasteful motion to the dashboard cards and page transitions. Respect prefers-reduced-motion." |

---

## Phase 5 — Docs & process

| Task | Skills | Prompt |
|---|---|---|
| Sync the whole docs suite with current code | `documentation`, `doc-coauthoring`, `spec-to-code-compliance` | "Check docs/ and DOCUMENTATION.md against the current code. Update ARCHITECTURE, API_SPECIFICATION, SECURITY_AND_RBAC, DATABASE_SCHEMA, OFFLINE_SYNC to match the internal/ layout and per-learner model." |
| Regenerate the DOCX deliverables | `docx`, `documentation` | "Run scripts/generate_docs.py and scripts/generate_implementation_docx.py, then review the output for stale sections." |
| Turn the roadmap into tickets | `to-tickets`, `planning-and-task-breakdown` | "Break the revalidated findings into ordered, ticketed tasks with acceptance criteria." |

---

## Phase 6 — Release readiness

| Task | Skills | Prompt |
|---|---|---|
| CI/CD pipeline | `ci-cd-and-automation`, `setup-pre-commit` | "Add CI: go test + npm test + build + a security header/precache check. Add pre-commit hooks for gofmt/prettier/typecheck." |
| Pre-launch checklist | `shipping-and-launch`, `open-sourcing` | "Run the release checklist: secrets hygiene, health probes, logging, rollback plan, monitoring." |
| Final code health | `health` (gstack), `code-review-and-quality` | "Run a health check and quality review across the repo." |

---

## Golden rule

Do **Phase 0 first** — half the audit may already be fixed. Work top-down (CRITICAL → HIGH → polish). Every fix ends with a test that fails without the fix. When unsure which skill fits: *"Use the ask-matt skill"*.
