# LOG (Learning Observation Guidance) - Comprehensive System Documentation

## 1. Executive Summary & System Overview
The **LOG (Learning Observation Guidance)** platform is an advanced, resilient educational technology ecosystem engineered specifically to eliminate educational disparities in low-connectivity and infrastructure-constrained regions (such as rural Nepal). While modern digital learning platforms increasingly depend on continuous, high-bandwidth cloud connectivity, LOG is architected from the ground up as an offline-first, edge-ready smart learning companion.

LOG bridges the gap between self-directed learning and intelligent pedagogical oversight. By continuously capturing fine-grained learner telemetry, the system constructs reflective observations, diagnoses mastery gaps, and delivers actionable, positive guidance—even when completely disconnected from the central server.

---

## 2. Core Constraints & Guiding Principles

### 2.1 Low-Connectivity First
LOG is purpose-built for environments with intermittent or non-existent internet. Direct network calls that bypass the offline caching and queueing layer (`src/lib/api.ts`) are strictly prohibited.

### 2.2 Supportive & Constructive Phrasing
Observations, evaluations, and guidance messages must consistently employ positive, supportive phrasing. Negative phrasing (such as "You failed", "Poor score", or "Incorrect attempt") is banned. Affirmative recommendations (e.g. "This area could use more practice" or "You are making steady progress") are enforced.

### 2.3 Deterministic & Grounded Analytics
All student metrics, mastery scores, streak statistics, and guidance recommendations are directly computed from verified database records. External hallucinated generative LLM calls for grading or telemetry are strictly disallowed without verified data ground-truth.

---

## 3. The 5-Stage LOG Pedagogical Framework

| Stage | Core Action | System Function | Outcome |
| :--- | :--- | :--- | :--- |
| **1. Learn** | Interactive Modules | Presents bite-sized concepts and interactive checks | Acquisition of new theoretical knowledge |
| **2. Observe** | Habit & Telemetry | Captures response latency, quiz scores, and streak data | Transparent reflection of learner habits |
| **3. Understand** | Cognitive Diagnostic | Maps topic mastery and pinpoints specific difficulty zones | Awareness of strengths and growth opportunities |
| **4. Guide** | Targeted Action | Formulates concrete, actionable next steps | Clear pathway to targeted remediation |
| **5. Improve** | Deliberate Practice | Executes targeted practice modules and review sessions | Long-term knowledge retention and mastery |

---

## 4. High-Level Architecture & Tech Stack

```
+-------------------------------------------------------------------------+
|                         CLIENT LAYER (Next.js 14 PWA)                   |
|  +---------------------+  +----------------------+  +----------------+  |
|  |   Student Portal    |  |    Teacher Portal    |  | Admin Console  |  |
|  +---------------------+  +----------------------+  +----------------+  |
|                         |                          |                    |
|                         v                          v                    |
|  +-------------------------------------------------------------------+  |
|  |                 OFFLINE RESILIENCE INTERCEPTOR                    |  |
|  |                    (src/lib/api.ts fetchWithCache)                |  |
|  +-------------------------------------------------------------------+  |
|            |                                              |             |
|   [Offline / Network Err]                       [Online Network]        |
|            v                                              v             |
|  +-----------------------+                      +--------------------+  |
|  | IndexedDB: api-cache  |                      | REST API Request   |  |
|  | IndexedDB: sync-queue |                      | (Bearer JWT Token) |  |
|  +-----------------------+                      +--------------------+  |
+------------|----------------------------------------------|-------------+
             |                                              |
             | (Auto-flushed on 'online' event)             |
             +--------------------+                         |
                                  v                         v
+-------------------------------------------------------------------------+
|                         BACKEND LAYER (Go / Gin)                        |
|  +-------------------------------------------------------------------+  |
|  | Middleware: CORS, Security Headers, JWT HMAC Auth, Multi-Tier RBAC |  |
|  +-------------------------------------------------------------------+  |
|            |                          |                         |       |
|            v                          v                         v       |
|  +--------------------+      +--------------------+   +---------------+ |
|  | /api/auth/*        |      | /api/dashboard     |   | /api/admin/*  | |
|  | OTP, JWT, Session  |      | Progress, Guidance |   | Users, Roles  | |
|  +--------------------+      +--------------------+   +---------------+ |
|                                       |                                 |
|                                       v                                 |
|  +-------------------------------------------------------------------+  |
|  |                  GORM Object-Relational Mapping                   |  |
|  +-------------------------------------------------------------------+  |
|                                       |                                 |
|                                       v                                 |
|  +-------------------------------------------------------------------+  |
|  |            PERSISTENCE: SQLite (Local) / PostgreSQL (Prod)         |  |
|  +-------------------------------------------------------------------+  |
+-------------------------------------------------------------------------+
```

### Technology Matrix
- **Frontend Framework:** Next.js 14 (App Router), React 18, TypeScript
- **Styling & Motion:** Tailwind CSS, Framer Motion, Lucide Icons, `react-circular-progressbar`, `react-hot-toast`
- **Charts & Visualization:** Recharts
- **Offline & PWA Layer:** `next-pwa`, Workbox, `idb` (IndexedDB)
- **Backend API:** Go (Gin Framework)
- **Database & ORM:** GORM with SQLite (`log.db`) and PostgreSQL support via Docker Compose
- **Testing:** Jest with mocked IndexedDB & fetch environment

---

## 5. Offline Syncing & Mutation Queueing (`src/lib/api.ts`)

### Dual-Store IndexedDB Model
1. `api-cache`: Key-value store mapping endpoint URLs (e.g. `/dashboard`) to cached JSON response payloads.
2. `sync-queue`: Auto-incrementing queue persisting offline mutations (`POST`, `PUT`, `DELETE`) with original payload, method, and authentication headers.

### Execution Flow
- **GET Requests:** When online, fetches from server and caches result. If offline or network fails, instantly returns cached payload.
- **Mutating Requests:** When offline or upon network failure, requests are enqueued in `sync-queue` with an optimistic `202 Accepted` response.
- **Automatic Reconnection Sync:** Listens for the `window.online` event, processes all queued actions sequentially (FIFO), deletes synced entries, and notifies the user via toast notification.

---

## 6. Multi-Tier RBAC & Security

### 3-Tier Hierarchy
1. **`ADMIN` (Principal / HOD):** Full oversight, user management, role modification, learning activity authoring.
2. **`MODERATOR` (Teacher):** Classroom roster tracking, student progress evaluation, attention flags.
3. **`STUDENT` (Learner):** Personal learning journey, interactive quizzes, progress analytics, and guidance.

### Security Hardening
- **Bcrypt:** Password hashing with cost factor 14 (`api/auth.go`).
- **JWT:** HMAC-SHA256 tokens with 72-hour expiration, verified in `AuthMiddleware`.
- **Validation:** Gin binding struct tags (e.g. `binding:"required,min=10,max=15"`).
- **HTTP Headers:** `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 1; mode=block`.

---

## 7. REST API Reference

| Method | Endpoint | Access | Description |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/auth/request-otp` | Public | Generates and sends a 6-digit verification OTP (bcrypt-hashed, `crypto/rand`) |
| `POST` | `/api/auth/verify-otp` | Public | Verifies OTP and returns user data + JWT token |
| `POST` | `/api/auth/forgot-password` | Public | Anti-enumeration password reset confirmation |
| `POST` | `/api/auth/google` | Public | Federated (mock OAuth) sign-in |
| `POST` | `/api/auth/logout` | Authenticated | Revokes the caller's JWT via `jti` blocklist |
| `GET` | `/api/ping` | Public | Health probe returning `{"message": "pong"}` |
| `GET` | `/api/dashboard` | Student+ | Returns student profile, progress, activities, observations, guidance |
| `GET` | `/api/learning-journey` | Student+ | Returns sequenced list of curriculum activities |
| `GET` | `/api/chart-data` | Student+ | Returns weekly progress and engagement durations |
| `GET` | `/api/courses` | Student+ | Paginated course catalog from the `courses` table |
| `GET` | `/api/activities/:id/modules` | Student+ | Bite-sized `MicroModule` content for the module viewer |
| `POST` | `/api/activities/:id/complete` | Student+ | Transactional completion: progress, observation, guidance |
| `POST` | `/api/sync/bulk` | Student+ | Bulk offline sync from `.logsync` files (transactional, scoped to caller) |
| `GET` | `/api/moderator/classes` | Moderator+ | Returns teacher class data |
| `GET` | `/api/moderator/roster` | Moderator+ | Live roster with computed completion % and streak data |
| `GET` | `/api/admin/dashboard` | Admin | System-wide user counts, active-daily, completions (from DB) |
| `GET` | `/api/admin/users` | Admin | Returns complete user list |
| `PUT` | `/api/admin/users/:id/role` | Admin | Updates user role (`STUDENT`, `MODERATOR`, `ADMIN` — validated) |
| `POST` | `/api/admin/activities` | Admin | Creates a new curriculum activity (strict DTO, server-managed fields) |

---

## 8. Frontend Navigation & Pages

- **`/` (Landing):** Features LOG cycle cards, hero banner, and quick start CTA.
- **`/login`:** Phone OTP login, mock Google OAuth, session state initialization.
- **`/forgot-password`:** Password reset form with validation.
- **`/dashboard`:** Daily streak counter, circular goal progress, current focus guidance, recent observations, `.logsync` export/import, logout.
- **`/learning`:** Sequenced learning path with module cards and status icons (live links into the player).
- **`/learning/[id]`:** Micro-module viewer (server content, cached for offline replay) with demo-lesson fallback, feedback, and score calculations.
- **`/courses`:** Searchable catalog served from the real `/api/courses` endpoint with dynamic category filters.
- **`/observation`:** KPI metrics, weekly performance area chart, daily engagement bar chart, detailed insights.
- **`/guidance`:** Action-oriented recommendations with direct action links.
- **`/moderator`:** Class roster with offline pre-fetch, progress bars, active student indicators, attention flags.
- **`/admin`:** Administrative control center with live system counters, user table, and quick actions.

---

## 9. Generating & Updating the Word Documentation (`.docx`)

The project includes an automated Python documentation generator that outputs a formatted `.docx` file complete with branding, tables, diagrams, callout blocks, and styling:

```bash
# Generate / Update the Word Document
python3 scripts/generate_docs.py
```

Output file:
- `LOG_Project_Documentation.docx` (in the project root)
