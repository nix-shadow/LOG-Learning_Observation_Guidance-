# LOG (Learning Observation Guidance) - Developer & Agent Guide

This document outlines the core architecture, constraints, and engineering guidelines for the LOG platform. Any automated agent or developer contributing to this repository MUST adhere to these rules.

---

## 1. Core Principles & Constraints
- **Low Connectivity First:** LOG is built for regions in Nepal with spotty internet. You must never bypass the offline caching layer (`src/lib/api.ts`).
- **Supportive Language:** Guidance and Observations must use positive, supportive language. Do not use negative phrasing. Example: Use "This area could use more practice" instead of "You failed this module."
- **No Hallucinations:** Guidance, metrics, and analytics must be derived from actual backend data. Do not integrate external LLM APIs for content generation unless explicitly requested.
- **No Fabricated Fallbacks:** The frontend must NEVER render invented numbers or placeholder students when data is unavailable. Show honest `0`/empty/error states instead (e.g., dashboard goal ring, moderator roster). See `docs/ENHANCEMENT.md` §2.2.
- **No Committed Secrets:** `JWT_SECRET` and other secrets must come from the environment (`.env` / shell). Never commit real secrets to the repo — `.env.example` files are the only allowed templates.

---

## 2. Architecture & Tech Stack
- **Frontend:** Next.js 14 (App Router), TypeScript, Tailwind CSS, `framer-motion` (animations), `recharts` (data visualization), `react-hot-toast` (notifications).
- **Backend:** Go (Gin framework), `gorm` ORM.
- **Database:** SQLite (`log.db`) is currently used for local development and persistence. The schema is auto-migrated on startup.
- **Offline Layer:** `next-pwa` handles the Service Worker. `idb` handles IndexedDB.

### 2a. Brand & Theme — THE LOG PALETTE (source of truth, WP-4.5)
The design system is driven entirely by CSS custom properties in `frontend/src/app/globals.css`. Both themes use the same brand hues — dark mode is the native design (`.dark` class on `<html>`), light mode (`html:not(.dark)`) is the official palette. **Never introduce hardcoded off-brand colors** (cyan `#00B4D8`, magenta `#FF0070`, purple `#7000FF`, old amber `#FFB703`) — the legacy neon/black scheme was replaced (WP-4.5) and these are banned; use the tokens below.

| Role | Light mode (hex) | Dark mode (hex) |
|---|---|---|
| Primary Blue | `#2563EB` | `#60A5FA` (accents/glow) / `#3B82F6` (button fills) |
| Secondary Teal | `#0D9488` | `#2DD4BF` |
| Accent Amber | `#F59E0B` | `#FBBF24` |
| Background | `#F8FAFC` | `#0B1220` (brand navy — not black) |
| Surface / card | `#FFFFFF` | `#122036` |
| Primary text | `#0F172A` | `#E9F0FA` |
| Secondary text | `#64748B` | `#9DB0C9` |

Rules that keep it coherent:
- **Tokens:** `--brand-blue/teal/amber/white/gray/dark/darker/text/neon/muted/faint` are HSL triplets; components consume them via `hsl(var(--brand-*))`, so both modes re-skin from the token block alone. `--glow-rgb`/`--teal-rgb`/`--amber-rgb` hold the RGB triplets used by shadow/glow utilities.
- **Literal utilities** (`text-white`, `border-white/*`, `bg-white/*`, `bg-black/*`, `from-white`, gray/neutral scales) are remapped under `html:not(.dark)` so component markup stays theme-agnostic.
- **Text contrast (WCAG):** on light, amber/teal *text* uses darker shades (`#B45309`/`#0F766E`); solid brand fills keep white labels via `html:not(.dark) .btn-primary { color:#fff !important }` (Tailwind merges a bare `.btn-primary` selector into the `text-white` remap otherwise).
- **Never fabricate colors:** charts/graphs use the brand hexes (`#2563EB` score, `#0D9488` accuracy, `#F59E0B` engagement); the goal ring is a fixed navy badge (`#0B1220`) with `#60A5FA` path in both modes.
- **No glow-on-hover (WP-4.6):** hover/focus feedback is a border/background tint or a lift — never a colored bloom. `shadow-glow`/`shadow-glow-strong` are subtle elevation halos only (see `tailwind.config.ts`); do not attach them to `hover:` variants, and do not add `drop-shadow-[0_0_*]` glows to icons. Semantic state glows on quiz correct/incorrect chips are the only exception.
- **Navigation layout (WP-4.6, research-driven):** desktop is a `1fr auto 1fr` grid — logo far-left | truly-centered learner links (exactly 5: Dashboard, Learning, Catalog, Observation, Guidance) | utilities (language, theme, avatar). **No logout on the bar** — it lives as the LAST item of the avatar disclosure menu (`AccountMenu` in `Navigation.tsx`: identity header + role chip → Settings/Support → role-scoped Parent/Moderator/Admin → divider → Log out). Menus follow the APG disclosure pattern (`aria-expanded`, Escape-closes-and-refocuses, pointerdown-outside close, close on route change); active links carry `aria-current="page"` + brand-blue pill. The mobile bottom bar renders **outside `<nav>`** — `backdrop-filter` on the nav makes it the containing block for `position:fixed` and would pin the bar under the header; bar = 5 equal-width tabs (`flex-1 min-w-0`, ≥48px targets), utilities live inside the mobile account menu.

---

## 3. Advanced Offline Syncing Strategy
All frontend API calls MUST go through `fetchWithCache` in `src/lib/api.ts`.
- **GET Requests:** Cached in the `api-cache` IndexedDB store with a 24h TTL. If the network fails, the cached payload is returned.
- **Mutating Requests (POST/PUT/DELETE):** If the network is unavailable, these requests are intercepted and stored in the `sync-queue` IndexedDB store. The function returns an optimistic `202 Accepted` response. When the `online` window event fires, the queue is automatically flushed to the backend.
- **Queue Integrity:** Queued records are NEVER deleted on 401 — the flush stops and asks the user to re-login instead (deleting would lose learner work). Replays re-attach the current token from localStorage at flush time, so records queued under an expired session sync after re-login.
- **Cache Invalidation:** After a successful sync of `/activities/:id/complete` or `/sync/bulk`, the `api-cache` entries for `/dashboard`, `/learning-journey`, and `/chart-data` are invalidated.
- **Sneakernet:** `downloadSyncFile()`/`importSyncFile()` (`src/lib/syncExport.ts`) export/import the queue as `.logsync` files for offline devices; imported actions upload via `POST /api/sync/bulk`. Exported files are **plaintext by design** (the user's own data, meant for another device that lacks the queue key) — treat `.logsync` files like passwords.
- **Queue Encryption at Rest (WP-0.1):** Queue payloads (headers + body) are encrypted with AES-GCM-256 (`src/lib/crypto.ts`) before entering IndexedDB; endpoint/method/timestamp stay plaintext for routing. The non-extractable CryptoKey lives in the `sync-keys` IDB store (DB version 4) and survives logout so records queued under an expired session still sync after re-login. Each record carries a fresh 12-byte IV and an SHA-256 fingerprint of `method|endpoint|body` for dedup. `decryptQueuePayload` returns `null` on failure and the record is **preserved** — never delete learner work on a key hiccup. Records without `enc` (legacy v3 or non-secure-context fallback) flush as-is.

---

## 3b. Privacy & Data Rights (WP-0.1)
Policy version: **`2026-08-v1`** (`backend/internal/domain/privacy.go`). Retention: learner data ≤ 2 years after last activity; audit logs ≤ 3 years; offline queue is user-controlled (90-day guidance).
- **Consent evidence:** `consent_records` table — one active row per (user, type); types `guardian`/`terms`/`privacy`; re-grants upsert (never duplicate), withdrawal flips status. Guardian consent is required at every registration path (frontend gates register + Google; the backend records consent via `POST /api/v1/me/consent`). **Server-side gate (enforcement round):** learner mutation routes (`enroll`, `unenroll`, `complete`, `sync/bulk`, `submit`, `password`) sit behind `RequireConsent` (`middleware.go`) — a raw API client cannot write learner data without an active, evidenced guardian grant; staff roles are exempt; repo errors 503 rather than guessing. The gate's 403 carries `code:"consent_required"`, and the offline queue **preserves** the record and stops the flush on it (never a terminal delete). Each guardian grant must carry `disclosure_hash` — the sha256 hex of the exact bilingual notice text the guardian saw (required; 400 otherwise), computed in `frontend/src/app/login/page.tsx` from the `CONSENT_NOTICE_*` constants the checkbox renders from (drift = invalid evidence). Non-secure contexts (plain-HTTP school LANs) fall back to an honestly-labeled `djb2-<hex>` hash, accepted by the backend — weaker by design, like the queue's `enc:null`. `ConsentRecord.IP` is `gorm:"-"` — client IPs are deliberately **never persisted**.
- **Retention purge (enforcement round):** `PurgeExpiredData` (`privacy_service.go`) erases learner accounts with no activity for `InactiveAccountRetentionYears` via the **full erasure map** (school context survives, per-erasure anonymized audit row written) and deletes audit rows past `AuditLogRetentionYears`. Runs at startup + a 24h ticker in `main.go`, plus manual `POST /api/v1/admin/maintenance/purge` (rate limited, returns the purge report). Staff are never purge candidates.
- **Forensic erasure (research-hardened):** `DELETE /me` runs `DeleteAccountTx`, then `ScrubDeletedData` — `PRAGMA wal_checkpoint(TRUNCATE)` + `VACUUM` (best-effort, logged) — because a plain DELETE leaves recoverable rows in freelist pages and the WAL file. `_secure_delete=ON` is in the DSN (zeroes b-tree cells). **Backups taken before an erasure still contain the erased rows** — rotate/destroy them (runbook §5).
- **Endpoints** (rate limited per IP, same hardening as auth): `POST /api/v1/me/consent`, `GET /api/v1/me/consent` (includes policy + retention), `GET /api/v1/me/export` (self-describing JSON envelope; password hash stripped; never cached client-side), `DELETE /api/v1/me` (requires body `{"confirm":"DELETE"}`). Every privacy/auth response also carries `X-RateLimit-Limit` + `X-RateLimit-Remaining` (RFC 9110 draft-7 headers), and 429s carry `Retry-After`.
- **Shared-device hygiene:** `POST /api/v1/auth/logout` and `POST /api/v1/auth/logout-all` both send `Clear-Site-Data: "cache", "cookies", "storage"` — a school-LAN shared browser cannot serve cached authenticated pages or a cached `/me/export` after the session is revoked.
- **Freshness honesty:** `GET /dashboard` and `GET /chart-data` include an `as_of` server timestamp; the dashboard hero and Observation charts render "Updated <date> at <time>" from it so a cached payload never masquerades as live data.
- **Erasure data map** (`privacy_repo.go` `DeleteAccountTx`, one transaction): learner-private rows deleted; audit logs, announcements, assignments, and class rows **anonymized** (`UserID`/`AuthorID`/`CreatedBy`/`TeacherID` → `""`) so the school's context survives; the erasure audit entry is written atomically with `UserID=""` and `Detail="erasure_hash=<sha256(userID)[:16]>"` — the only joinable trace, no personal data.
- **No fabricated privacy state:** the admin users list shows `consent: null` when absent; the settings page shows "Pending" — never an invented value.
- **Incident response:** `docs/PRIVACY_RUNBOOK.md` is the S1/S2/S3 playbook, with a legal annex (§8): the Nepal Privacy Act 2075 has **no statutory breach-notification duty** (District Court civil remedy, 3-month limitation); DCCS Directives 2081/NCSC apply only to licensed data centers; the IT & Cybersecurity Bill 2082 (in amendment) would impose a 35-day purpose-based destruction clock — track it for the next policy version.

---

## 4. Multi-Tier RBAC & Security Hardening
The platform uses a strict 3-tier Role-Based Access Control (RBAC) system:
1. `ADMIN`: Principal/HOD level. Highest privilege. Access to `/admin`.
2. `MODERATOR`: Teacher level. Can manage classes and view student progress. Access to `/moderator`.
3. `STUDENT`: Normal user level. Access to dashboard, learning journey, and catalog.

**Security Requirements for Backend Changes:**
- **Passwords & OTPs:** Must be hashed using `bcrypt` (see `HashPassword` in `backend/internal/service/auth_utils.go`). Never log plaintext OTPs or passwords.
- **JWT:** Tokens must be validated strictly using HMAC. Check the `Bearer ` prefix. `JWT_SECRET` must come from the environment — never commit it or rely on a dev fallback (`main.go` and `docker-compose.yml` both abort when it is unset). The auth middleware re-loads the user from the DB and rejects tokens whose role no longer matches (demotion/soft-delete).
- **Identity:** Never trust client-supplied identity (email/name) for account access without verifying an external token server-side. `POST /api/v1/auth/google` exchanges a Google `id_token` that is verified server-side (`internal/service/auth_service_impl.go`) — a bare `{email}` body is rejected.
- **OTP:** Per-phone brute-force protection is enforced — 5 failed verify attempts invalidate the OTP; the `/api/v1/auth/*` routes are rate limited per client IP, and gin is configured to trust no proxies so `X-Forwarded-For` cannot spoof the key.
- **Input Validation:** Use Gin's binding validation (e.g., `binding:"required,min=10"`). Never trust client input.
- **Headers:** Security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection) are enforced globally in `main.go`.
- **Health Probes:** `/api/ping` (liveness) and `/healthz` (real SQLite ping) are public; never move them behind auth.

**Git Hygiene:**
- `backend/log.db`, `backend/server`, `frontend/public/sw.js`, and `frontend/public/workbox-*.js` are generated/build artifacts — gitignored and untracked. Do not re-add them.
- Never commit `.env` files. Copy from `.env.example` (root, `backend/`, `frontend/`) instead.
- **No orphan submodules:** a gitlink (mode 160000) in the index without a matching `.gitmodules` entry makes every `actions/checkout` cleanup fail with "No url found for submodule path". Never `git add` a directory that contains its own `.git`; if one slips in, remove with `git rm --cached <path>`.

**CI stability rules (learned from failed runs 96666852249 / 96666852275):**
- Action major versions must match the tool major they wrap — `golangci-lint v2.x` requires `golangci/golangci-lint-action@v7` (v6 hard-rejects v2). When bumping either side in `.github/workflows/ci.yml`, bump the pair together.
- WebCrypto inputs must be **fresh copies**: pass `new Uint8Array(view).buffer`, never `view.buffer` or a sliced view of a Node Buffer/pooled ArrayBuffer — some WebCrypto implementations reject such backing stores and `crypto.subtle.decrypt` throws, which `decryptQueuePayload` converts to `null` (preserved records, failing tests). See `asBuffer` in `frontend/src/lib/crypto.ts`.

---

## 5. Running the Application

### Environment
Copy the templates first — never commit real `.env` files:
```bash
cp .env.example .env                  # root: JWT_SECRET + CORS_ORIGIN (docker-compose)
cp backend/.env.example backend/.env  # backend: JWT_SECRET, CORS_ORIGIN, PORT, GOOGLE_CLIENT_ID (optional)
cp frontend/.env.example frontend/.env  # frontend: NEXT_PUBLIC_API_URL
```
Generate a strong secret with `openssl rand -base64 48`.

### Backend
The backend auto-seeds initial data on the first run. Seeded demo accounts (dev/testing only — bcrypt-hashed at startup, never in production):
- `admin@log.edu` / `Admin@123` (ADMIN)
- `teacher@log.edu` / `Teacher@123` (MODERATOR)
- `aisha@example.com` / `Student@123` (STUDENT)
```bash
cd backend
go build -o server main.go
./server &
```
Probes: `GET /api/ping` (liveness), `GET /healthz` (SQLite ping), `GET /readyz`.

### Frontend
Ensure the `.env` file contains `NEXT_PUBLIC_API_URL=http://localhost:6101/api/v1`.
```bash
cd frontend
npm install
npm run build
npm start &
```

### Docker
```bash
cp .env.example .env   # then set JWT_SECRET
docker compose up --build -d
```

### Testing
Frontend tests (specifically covering the offline sync layer) are written in Jest.
```bash
cd frontend
npx jest
```
Backend tests run against a real local `log.db` — wipe it before a demo run if you want pristine seed data.
```bash
cd backend
go test ./...
```
Linting (enforced in CI): `golangci-lint run ./...` from `backend/` (config: `backend/.golangci.yml` — gosec, staticcheck, errcheck; run `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6` to install) and `npm run lint` from `frontend/`. Everything is also reachable from the root `Makefile`: `make build`, `make test`, `make lint`.

### Known Limitations & Roadmap
See `docs/ENHANCEMENT.md` for the audited issue list and phased improvement plan (GoogleAuth verification, per-learner activity status, queue idempotency, CI/CD, etc.).
