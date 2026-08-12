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
- **Sneakernet:** `downloadSyncFile()`/`importSyncFile()` (`src/lib/syncExport.ts`) export/import the queue as `.logsync` files for offline devices; imported actions upload via `POST /api/sync/bulk`.

---

## 4. Multi-Tier RBAC & Security Hardening
The platform uses a strict 3-tier Role-Based Access Control (RBAC) system:
1. `ADMIN`: Principal/HOD level. Highest privilege. Access to `/admin`.
2. `MODERATOR`: Teacher level. Can manage classes and view student progress. Access to `/moderator`.
3. `STUDENT`: Normal user level. Access to dashboard, learning journey, and catalog.

**Security Requirements for Backend Changes:**
- **Passwords & OTPs:** Must be hashed using `bcrypt` (see `HashPassword` in `api/auth.go`). Never log plaintext OTPs or passwords.
- **JWT:** Tokens must be validated strictly using HMAC. Check the `Bearer ` prefix. `JWT_SECRET` must come from the environment — never commit it or rely on the dev fallback in production.
- **Identity:** Never trust client-supplied identity (email/name) for account access without verifying an external token server-side. `POST /api/auth/google` currently accepts unverified emails — treat it as a known vulnerability until fixed (see `docs/ENHANCEMENT.md` §1.1).
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
cp backend/.env.example backend/.env  # backend: JWT_SECRET, CORS_ORIGIN, PORT
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
Ensure the `.env` file contains `NEXT_PUBLIC_API_URL=http://localhost:6001/api`.
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
