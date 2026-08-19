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
The backend auto-seeds initial data on the first run.
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

### Known Limitations & Roadmap
See `docs/ENHANCEMENT.md` for the audited issue list and phased improvement plan (GoogleAuth verification, per-learner activity status, queue idempotency, CI/CD, etc.).
