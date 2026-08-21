# LOG (Learning Observation Guidance)

[![CI](https://github.com/nix-shadow/LOG-Learning_Observation_Guidance-/actions/workflows/ci.yml/badge.svg)](https://github.com/nix-shadow/LOG-Learning_Observation_Guidance-/actions/workflows/ci.yml)

LOG is an educational technology platform built for learners in low-connectivity regions. It functions as a smart learning companion that helps users Understand their progress and receive actionable Guidance.

## Core Cycle
The application revolves around the LOG cycle:
1. **Learn:** Engage with modules.
2. **Observe:** See clear reflections of progress and habits.
3. **Understand:** Identify strengths and areas needing attention.
4. **Guide:** Receive actionable recommendations.
5. **Improve:** Apply guidance to build knowledge.

## Tech Stack
* **Frontend:** Next.js 14, React, TypeScript, Tailwind CSS, Lucide Icons, Recharts, Framer Motion.
* **Offline Layer:** `next-pwa` for Service Workers, `idb` for IndexedDB caching and mutation queues.
* **Backend:** Go, Gin framework.
* **Database:** SQLite via GORM (auto-migrated and seeded).

## Advanced Architecture
- **Multi-tier RBAC:** Secure Role-Based Access Control supporting `ADMIN`, `MODERATOR`, and `STUDENT` roles.
- **Offline Mutating:** The `src/lib/api.ts` wrapper queues POST/PUT/DELETE requests when offline and syncs them automatically when the connection is restored.
- **Sneakernet Sync:** Export `.logsync` files on an unconnected device and import them anywhere for `POST /api/sync/bulk` upload.
- **Real Data Everywhere:** Dashboards, catalogs, rosters, and analytics are computed from database records — no hardcoded mock telemetry.
- **Security:** Strict JWT HMAC validation with `jti` revocation, Bcrypt password & OTP hashing, rate limiting, Input validation via Gin binding, and comprehensive HTTP security headers.

## Setup Instructions

### Environment Variables
For local development, create a `.env` file in the frontend directory:
```
NEXT_PUBLIC_API_URL=http://localhost:6101/api/v1
```

### Running the Application
Read the `AGENTS.md` file for detailed instructions on running, testing, and developing the application.

Public health probes (never moved behind auth — monitoring needs them unauthenticated):
```
GET /api/ping   liveness
GET /healthz    real SQLite ping (200 ok / 503 unhealthy)
GET /readyz     readiness
```

## Documentation & Architecture Specifications

### 📚 Modular Markdown Documentation Suite (`docs/`)
- [System Architecture & Overview](./docs/ARCHITECTURE.md)
- [Offline Sync & PWA Engine](./docs/OFFLINE_SYNC.md)
- [REST API Specification](./docs/API_SPECIFICATION.md)
- [Security & Multi-Tier RBAC](./docs/SECURITY_AND_RBAC.md)
- [Database Schema & Data Models](./docs/DATABASE_SCHEMA.md)
- [Frontend Architecture & UI Guide](./docs/FRONTEND_GUIDE.md)
- [Developer Implementation Guide](./docs/IMPLEMENTATION_GUIDE.md)
- [Research Findings & Improvement Plan](./docs/ENHANCEMENT.md)
- [Full Monolithic Specification](./DOCUMENTATION.md)

### 📄 Professional Word Documents (.docx)
- **Master Project Documentation:** [LOG_Project_Documentation.docx](./LOG_Project_Documentation.docx)
- **Technical Implementation Guide:** [LOG_Implementation_Guide.docx](./LOG_Implementation_Guide.docx)

### 🔄 Regenerate / Update All DOCX Files
```bash
# Update Master Project Documentation DOCX
python3 scripts/generate_docs.py

# Update Implementation Guide DOCX
python3 scripts/generate_implementation_docx.py
```


