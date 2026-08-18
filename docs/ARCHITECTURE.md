# System Architecture & Technical Specification

## 1. System Overview
The **LOG (Learning Observation Guidance)** platform is an edge-ready educational technology system designed to provide uninterrupted learning experiences in low-connectivity and infrastructure-constrained regions (such as rural Nepal). 

Unlike conventional learning management systems (LMS) that rely on uninterrupted high-speed internet, LOG is architected with an **offline-first paradigm**. It enables learners to study modular curriculum materials, complete interactive quizzes, track their progress, and receive targeted guidance without requiring an active connection. When connectivity is restored, all offline progress is synchronized with the backend.

---

## 2. The 5-Stage LOG Pedagogical Framework

The platform is structured around a continuous learning and feedback loop:

```
      +------------+
      |  1. LEARN  | <---+
      +------------+     |
            |            |
            v            |
      +------------+     |
      | 2. OBSERVE |     |
      +------------+     |
            |            |
            v            |
    +---------------+    |
    | 3. UNDERSTAND |    |
    +---------------+    |
            |            |
            v            |
      +------------+     |
      |  4. GUIDE  |     |
      +------------+     |
            |            |
            v            |
      +------------+     |
      | 5. IMPROVE | ----+
      +------------+
```

1. **Learn:** Learners engage with bite-sized, interactive modular concepts and knowledge checks.
2. **Observe:** The system logs real-time telemetry (accuracy, latency, study streaks, and activity completions).
3. **Understand:** Telemetry is analyzed to diagnose mastery levels and identify specific topics requiring attention.
4. **Guide:** The engine provides actionable, positive recommendations and direct links to relevant learning resources.
5. **Improve:** Learners apply the guidance through focused review and targeted practice sessions.

---

## 3. High-Level Component Architecture

```
+-----------------------------------------------------------------------------------+
|                           CLIENT TIER (Next.js 14 PWA)                            |
|                                                                                   |
|  +-----------------------+  +-----------------------+  +-----------------------+  |
|  |     Student App       |  |      Teacher App      |  |       Admin App       |  |
|  | (/dashboard, /courses,|  |     (/moderator)      |  |       (/admin)        |  |
|  |  /learning, /guidance)|  |                       |  |                       |  |
|  +-----------------------+  +-----------------------+  +-----------------------+  |
|              |                          |                          |              |
|              +--------------------------+--------------------------+              |
|                                         |                                         |
|                                         v                                         |
|  +-----------------------------------------------------------------------------+  |
|  |                  OFFLINE INTERCEPTOR (src/lib/api.ts)                       |  |
|  |                         fetchWithCache Layer                                |  |
|  +-----------------------------------------------------------------------------+  |
|             |                                                      |              |
|     [Network Offline]                                      [Network Online]       |
|             v                                                      v              |
|  +----------------------+                               +---------------------+   |
|  | IndexedDB (Dual Store|                               | HTTPS Request       |   |
|  |  - api-cache         |                               | (Bearer JWT Token)  |   |
|  |  - sync-queue)       |                               +---------------------+   |
|  +----------------------+                                          |              |
+-------------|------------------------------------------------------|--------------+
              |                                                      |
              | (Auto-flushed on 'online' event)                     |
              +----------------------+                               |
                                     v                               v
+-----------------------------------------------------------------------------------+
|                             BACKEND TIER (Go / Gin)                               |
|                                                                                   |
|  +-----------------------------------------------------------------------------+  |
|  |                           HTTP Security & CORS                              |  |
|  | (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection, Origin Header)  |  |
|  +-----------------------------------------------------------------------------+  |
|                                         |                                         |
|                                         v                                         |
|  +-----------------------------------------------------------------------------+  |
|  |                 AuthMiddleware & Multi-Tier RBAC Engine                     |  |
|  |                 (ADMIN -> MODERATOR -> STUDENT Privilege Hierarchy)        |  |
|  +-----------------------------------------------------------------------------+  |
|             |                           |                           |             |
|             v                           v                           v             |
|  +---------------------+   +-------------------------+   +---------------------+  |
|  |   Auth Controller   |   |   Student Controller    |   |   Admin Controller  |  |
|  | (/api/v1/auth/req-otp, |   | (/api/v1/dashboard,      |   | (/api/v1/admin/users, |  |
|  |  /api/v1/auth/ver-otp) |   |  /api/v1/learning-journey) |   |  /api/v1/admin/acts)   |  |
|  +---------------------+   +-------------------------+   +---------------------+  |
|                                         |                                         |
|                                         v                                         |
|  +-----------------------------------------------------------------------------+  |
|  |                         GORM (Object-Relational Mapping)                    |  |
|  +-----------------------------------------------------------------------------+  |
|                                         |                                         |
|                                         v                                         |
|  +-----------------------------------------------------------------------------+  |
|  |                  PERSISTENCE TIER: SQLite (Local) / Postgres (Prod)         |  |
|  +-----------------------------------------------------------------------------+  |
+-----------------------------------------------------------------------------------+
```

---

## 4. Technology Stack & Rationale

| Layer | Technology | Key Dependencies | Architectural Justification |
| :--- | :--- | :--- | :--- |
| **Frontend UI** | Next.js 14 (App Router) | React 18, TypeScript | Modern server/client component model, optimized bundle splitting, and client routing. |
| **Styling & Motion** | Tailwind CSS & Framer Motion | Lucide Icons, `clsx`, `tailwind-merge` | Utility-first responsive design tailored for mobile screens with hardware-accelerated animations. |
| **Data Visualization**| Recharts & CircularProgressbar | `recharts`, `react-circular-progressbar` | Lightweight, responsive SVG-based charting for performance trends and progress tracking. |
| **Offline Tier** | `next-pwa` & `idb` | Workbox, IndexedDB API | Service worker caching of app shell combined with structured IndexedDB request caching. |
| **Backend API** | Go (Gin Web Framework) | `gin-gonic/gin`, `golang-jwt/jwt/v5` | High-throughput, low-latency compiled backend with minimal memory footprint. |
| **Data Access & ORM** | GORM | `gorm.io/gorm`, `gorm.io/driver/sqlite` | Schema auto-migration, seed management, and database-agnostic queries. |
| **Database** | SQLite (Local) / PostgreSQL | `backend/data/log.db`, `postgres:15-alpine` | Embedded zero-setup SQLite for rapid deployment with seamless PostgreSQL scaling. |
| **Quality Assurance** | Jest & Testing Library | `ts-jest`, `jest-environment-jsdom` | Unit and integration testing for offline caching, queueing, and fallback logic. |

---

## 5. Core Engineering Constraints

### 5.1 Low-Connectivity First
Developers and agents must never introduce direct `fetch()` calls that bypass `src/lib/api.ts`. All network access must be wrapped by `fetchWithCache` to ensure offline availability.

### 5.2 Supportive Language Mandate
Guidance and observation strings must always use encouraging, growth-oriented phrasing. Negative words such as "failed", "incorrect", or "poor" are prohibited in user-facing content.

### 5.3 Deterministic Metrics Guarantee
Analytics, streak counters, and mastery scores must be calculated from database records. No external LLMs or ungrounded generative processes may be used for student evaluation data.

---

## 6. School Operations Module (Classes, Assignments, Announcements)

Added in the production-grade expansion, the school-operations module mirrors the existing layered pattern:

```
SchoolHandler (internal/handler/school_handler.go)
        │  HTTP + RBAC (admin / moderator / student routes)
        v
SchoolService (internal/service/school_service*.go)
        │  business rules:
        │   • only the class teacher may create assignments
        │   • only enrolled learners may submit
        │   • only STUDENT users can be enrolled
        v
SchoolRepository (internal/repository/school_repo.go)
        │  GORM + transactions + idempotent upserts
        v
tables: classes, class_members, announcements, assignments,
        submissions, audit_logs, user_revocations
```

Key properties:
- **Idempotent submissions** — `(assignment_id, learner_id)` unique pair; offline queue replays cannot duplicate work.
- **Honest exports** — the admin CSV export streams only real enrollment rows (header-only when empty).
- **Append-only audit trail** — sensitive ops (`user.role_change`, `class.*`, `assignment.create`, `announcement.create`, `export.*`, `auth.logout_all`) are recorded with actor + IP.
- **Global session revocation** — `POST /auth/logout-all` writes a `UserRevocation` row; `AuthMiddleware` rejects any token issued before it (uses the `iat` claim, which `GenerateJWT` now sets).
- **Class-scoped authorization** — class ownership and membership are enforced in the service layer, never trusted from the client.
