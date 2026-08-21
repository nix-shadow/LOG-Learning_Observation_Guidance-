# LOG Docs Hub

The LOG (Learning Observation Guidance) platform — an offline-first learning
platform for low-connectivity schools in Nepal. Every doc here is
evidence-based: no invented numbers, no fabricated states.

## Quick start

- **Architecture**: `ARCHITECTURE.md` · `IMPLEMENTATION_GUIDE.md` ·
  `API_SPECIFICATION.md` · `FRONTEND_GUIDE.md` · `DATABASE_SCHEMA.md`
- **Running the app**: see `AGENTS.md §5` (env setup, backend, frontend, Docker, tests)
- **Reporting**: `docs/html/07-phased-implementation-plan.html` (work-package
  tracker, 72/72) · `docs/html/` index for the research hub
- **Releasing**: `RELEASE.md` (incl. the mandatory manual TTI < 5s check) ·
  `CHANGELOG.md` · `VERSION`

## Operational docs

- `PRIVACY_RUNBOOK.md` — S1/S2/S3 incident playbook + legal annex (Nepal
  Privacy Act 2075, DCCS Directives 2081, IT & Cybersecurity Bill 2082)
- `QUERY_PLANS.md` — real `EXPLAIN QUERY PLAN` evidence for the hot-path
  queries, including the indexes studied and deliberately rejected
- `ENHANCEMENT.md` — audited issue list and phased improvement plan
- `BUGS.md` — known issues

## Engineering constraints (short version)

Full rules in `AGENTS.md`. The non-negotiables:

1. **Low connectivity first** — never bypass the offline caching layer
   (`src/lib/api.ts`); offline queue records are never deleted on auth or
   consent failures.
2. **No fabricated fallbacks** — no invented numbers, students, or states;
   honest `0`/empty/`null`/error views instead.
3. **Supportive language** — observations and guidance are positive; no
   negative phrasing.
4. **No committed secrets** — secrets come from the environment;
   `.env.example` templates are the only allowed files.
5. **Evidence over vibes** — metrics, guidance, and analytics derive from
   real backend data, and every claim in this hub points at the code or
   test that proves it.