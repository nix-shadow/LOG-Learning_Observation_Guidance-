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
* **Frontend:** Next.js 14, React, TypeScript, Tailwind CSS, Lucide Icons.
* **Offline Layer:** `next-pwa` for Service Workers, `idb` for IndexedDB caching.
* **Backend:** Go, Gin framework.
* **Database:** PostgreSQL (with a robust in-memory mock layer currently configured for local dev).

## Setup Instructions

### Environment Variables
For local development, create a `.env` file in the frontend directory:
`NEXT_PUBLIC_API_URL=http://localhost:8080/api`

## Offline Support
This application uses a custom API wrapper (`src/lib/api.ts`) that intercepts requests. If the network is available, it fetches from the Go backend and caches the result in IndexedDB. If the network fails or the user goes offline, it seamlessly serves the cached data to keep the app functional.
