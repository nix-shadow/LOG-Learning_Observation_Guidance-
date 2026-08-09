# LOG (Learning Observation Guidance)

## Core Principles
1. **Offline-First:** This app is designed for regions in Nepal with spotty connectivity. Rely on `IndexedDB` (via `src/lib/api.ts -> fetchWithCache`) to serve data when offline. Do not bypass this layer.
2. **Supportive Language:** Guidance and Observations must use positive, supportive language. Avoid negative phrasing. Use "This area could use more practice."
3. **No AI Hallucinations:** Guidance and metrics must be derived from actual data/logic. Do not insert LLM/AI APIs unless explicitly requested.

## Architecture
- **Frontend:** Next.js 14 (App Router), TypeScript, Tailwind CSS, `next-pwa`, `idb`.
- **Backend:** Go (Gin framework), PostgreSQL.
- **Styling:** Colors defined in `tailwind.config.ts`.
