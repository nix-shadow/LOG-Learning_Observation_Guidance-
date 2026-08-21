# Contributing to LOG

LOG (Learning Observation Guidance) is an offline-first learning platform for
low-connectivity schools in Nepal. Thank you for contributing. Please read
`AGENTS.md` first — it is the engineering contract for this repository and
applies to humans and automated agents alike.

## The non-negotiables

1. **Low connectivity first** — every frontend API call goes through
   `fetchWithCache` (`src/lib/api.ts`). Never bypass the offline caching or
   the sync queue. Queued records are learner work: never deleted on auth
   or consent failures.
2. **No fabricated fallbacks** — no invented numbers, placeholder students,
   or guessed analytics. Render honest `0`/empty/`null`/error states.
3. **Supportive language** — observations and guidance are positive.
   "This area could use more practice", not "You failed".
4. **No committed secrets** — `JWT_SECRET` and friends come from the
   environment. Copy `.env.example` → `.env`; never commit a real `.env`
   or a dev fallback secret.
5. **Evidence over vibes** — new metrics or analytics must be derivable
   from real backend data. If you add a hot-path query, capture its
   `EXPLAIN QUERY PLAN` in `docs/QUERY_PLANS.md`.

## Working on a feature

- Tie your work to a work package in the tracker
  (`docs/html/07-phased-implementation-plan.html`) and check it off with an
  evidence line (tests or files) when done.
- Phase reports live in `docs/reports/`; index cards in `docs/html/index.html`.

## Code conventions

- **Backend**: Go + Gin + GORM. Services own business rules; handlers map
  errors to status codes; repositories own transactions. Run
  `gofmt` on your files, `go vet ./...`, `go test ./...` (runs against a
  real local `log.db` — wipe it before a demo for pristine seed data), and
  `golangci-lint run ./...` from `backend/`.
- **Frontend**: Next.js 14 App Router, TypeScript, Tailwind. Run
  `npx tsc --noEmit`, `npx jest`, `npm run lint`, and the budget gate
  (`node scripts/check-budget.mjs` — builds + enforces the 500 kB
  First Load JS limit).
- **Git hygiene**: never track `backend/log.db`, `backend/server`,
  `frontend/public/sw.js`, `frontend/public/workbox-*.js`, or `.env`.

## Releasing

Follow `docs/RELEASE.md` — including the manual real-device TTI < 5s check
that CI cannot perform.

## Reporting bugs

Open an issue with: the WP id it touches (if any), repro steps, and — for
data issues — whether the value came from the backend or was rendered
client-side. If you found a fabricated number, that is a P1: the
"no fabricated fallbacks" rule is the project's core honesty contract.