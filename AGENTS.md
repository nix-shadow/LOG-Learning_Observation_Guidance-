# LOG (Learning Observation Guidance) - Developer & Agent Guide

This document outlines the core architecture, constraints, and engineering guidelines for the LOG platform. Any automated agent or developer contributing to this repository MUST adhere to these rules.

---

## 1. Core Principles & Constraints
- **Low Connectivity First:** LOG is built for regions in Nepal with spotty internet. You must never bypass the offline caching layer (`src/lib/api.ts`).
- **Supportive Language:** Guidance and Observations must use positive, supportive language. Do not use negative phrasing. Example: Use "This area could use more practice" instead of "You failed this module."
- **No Hallucinations:** Guidance, metrics, and analytics must be derived from actual backend data. Do not integrate external LLM APIs for content generation unless explicitly requested.

---

## 2. Architecture & Tech Stack
- **Frontend:** Next.js 14 (App Router), TypeScript, Tailwind CSS, `framer-motion` (animations), `recharts` (data visualization), `react-hot-toast` (notifications).
- **Backend:** Go (Gin framework), `gorm` ORM.
- **Database:** SQLite (`log.db`) is currently used for local development and persistence. The schema is auto-migrated on startup.
- **Offline Layer:** `next-pwa` handles the Service Worker. `idb` handles IndexedDB.

---

## 3. Advanced Offline Syncing Strategy
All frontend API calls MUST go through `fetchWithCache` in `src/lib/api.ts`.
- **GET Requests:** Cached in the `api-cache` IndexedDB store. If the network fails, the cached payload is returned.
- **Mutating Requests (POST/PUT/DELETE):** If the network is unavailable, these requests are intercepted and stored in the `sync-queue` IndexedDB store. The function returns an optimistic `202 Accepted` response. When the `online` window event fires, the queue is automatically flushed to the backend.

---

## 4. Multi-Tier RBAC & Security Hardening
The platform uses a strict 3-tier Role-Based Access Control (RBAC) system:
1. `ADMIN`: Principal/HOD level. Highest privilege. Access to `/admin`.
2. `MODERATOR`: Teacher level. Can manage classes and view student progress. Access to `/moderator`.
3. `STUDENT`: Normal user level. Access to dashboard, learning journey, and catalog.

**Security Requirements for Backend Changes:**
- **Passwords:** Must be hashed using `bcrypt` (see `HashPassword` in `api/auth.go`).
- **JWT:** Tokens must be validated strictly using HMAC. Check the `Bearer ` prefix.
- **Input Validation:** Use Gin's binding validation (e.g., `binding:"required,min=10"`). Never trust client input.
- **Headers:** Security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection) are enforced globally in `main.go`.

---

## 5. Running the Application

### Backend
The backend auto-seeds initial data on the first run.
```bash
cd backend
go build -o server main.go
./server &
```

### Frontend
Ensure the `.env` file contains `NEXT_PUBLIC_API_URL=http://localhost:8080/api`.
```bash
cd frontend
npm install
npm run build
npm start &
```

### Testing
Frontend tests (specifically covering the offline sync layer) are written in Jest.
```bash
cd frontend
npx jest
```
