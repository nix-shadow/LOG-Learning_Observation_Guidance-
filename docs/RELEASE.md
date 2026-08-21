# LOG Release Procedure

How a release of the LOG platform ships. Version lives in `VERSION` and
`frontend/package.json`; history is in `CHANGELOG.md`.

## Release checklist

1. **Freeze the tree** — verify the working tree is clean and CI is green
   (`.github/workflows/ci.yml`): backend build/vet/golangci-lint/test, frontend
   typecheck/jest/build + RESPECT budget, git hygiene job.
2. **Bump the version** — decide the bump (WP-0.x → 0.1.0; each phase adds a
   minor: 0.2.0 … 0.5.0). Write `VERSION`, update `frontend/package.json`,
   add the `CHANGELOG.md` section tied to the WP ids in this phase.
3. **Manual TTI check (required, cannot be automated)** — the RESPECT budget
   gate only measures bundle size; real-device Time-To-Interactive must be
   verified by hand on the lowest-spec device the school actually has:
   - Open the deployed login page on that device on a 2G-class connection
     (or throttle in DevTools to Slow 3G).
   - The first interactive paint (login form usable) must occur **under 5
     seconds**.
   - Record the device, network setting, and measured time in the release
     notes. If TTI ≥ 5s, do NOT ship: cut bundle with dynamic imports and
     re-run this step.
4. **Smoke the offline layer** — on a fresh profile: complete an activity,
   kill the network, complete another (must queue with optimistic 202),
   restore network, confirm the queue flushes and the dashboard reflects
   both completions. Then export a `.logsync` file and import it on a
   second device.
5. **Seed check** — with a fresh `log.db`, confirm seeded accounts work
   (`admin@log.edu` / `Admin@123` etc.) and the FK migration reports clean
   (`MigrateForeignKeys` logs no orphan skips).
6. **Tag & ship** — tag `v<version>`; build artifacts stay gitignored
   (`backend/server`, `frontend/.next`, `log.db`).

## Post-release

- Re-run `docs/PRIVACY_RUNBOOK.md` S1/S2/S3 checks if the release touched
  consent, retention, or erasure code.
- If the release touched hot-path queries, re-run the EXPLAIN evidence in
  `docs/QUERY_PLANS.md` against the new database and update the doc.

## Rollback

- SQLite schema is migrated idempotently (`migrate_fks.go`); a rollback is a
  code revert + old binary — the DB remains usable by either version.
- Backups taken before an erasure still contain erased rows — on rollback
  after a privacy incident, rotate/destroy pre-erasure backups (runbook §5).