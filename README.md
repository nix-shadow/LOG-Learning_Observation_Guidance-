# LOG (Learning Observation Guidance)

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
- **Security:** Strict JWT HMAC validation, Bcrypt password hashing, Input validation via Gin binding, and comprehensive HTTP security headers.

## Setup Instructions

### Environment Variables
For local development, create a `.env` file in the frontend directory:
```
NEXT_PUBLIC_API_URL=http://localhost:8080/api
```

### Running the Application
Read the `AGENTS.md` file for detailed instructions on running, testing, and developing the application.
