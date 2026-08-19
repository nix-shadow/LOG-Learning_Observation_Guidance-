# LOG Privacy Incident Runbook

Operational response for privacy incidents involving LOG learner data.
Familiarity with the data map in the WP-0.1 design doc
(`docs/html/08-wp01-privacy-pack.html`) is assumed. **All incidents are
reported to the school leadership. Note: the Nepal Privacy Act 2075 has no
statutory breach-notification duty** — enforcement runs through the District
Court (civil remedy, §31) and regulator complaints; see the legal annex in
§8 for exactly what the law does and does not require.

Severity grading (always escalate one level for learner data involving minors):

| Severity | Definition | Example |
|---|---|---|
| S1 | Personal data of any user disclosed to an unauthorized party, or device/DB compromise | Stolen laptop with `backend/data/log.db`; exported `.logsync` file lost |
| S2 | Data integrity loss or erasure risk without disclosure | Queue key wiped while records pending; failed sync deleted records |
| S3 | Policy violation without data exposure | Consent recorded with wrong policy version; retention promise not met |

---

## 1. Detect

- Backend: `backend/log.db` is the only persistent store. Check `audit_logs`
  for unexpected `privacy.*` actions; check `consent_records` for grants you
  did not see in the UI.
- Offline devices: `sync-queue` (IndexedDB) holds learner work, encrypted
  since policy `2026-08-v1`; `.logsync` export files are **plaintext by
  design** — a lost file is an S1 incident.
- Breach clues: unknown admin/role changes, `/me/export` calls from unknown
  IPs (see `X-Request-ID` in request logs), `DELETE /me` erasures the user
  did not request.

## 2. Contain

1. **Revoke sessions:** as the principal (ADMIN), have the affected user
   (or admin) call `POST /api/v1/auth/logout-all`. For a suspected backend
   compromise, stop the service (`kill` the `server` process) and copy
   `backend/log.db` to offline storage **before** any change — it is the
   evidence and the erasure trail.
2. **Identify the affected data:** query `audit_logs` (anonymized rows carry
   `erasure_hash=` in Detail — the truncated SHA-256 of the user ID) and
   `consent_records`. The export envelope lists every data category; the
   erasure map (design doc §4) is the canonical table of what exists.
3. **Cut external reach:** the backend trusts no proxies; if the compromise
   is network-level, block at the firewall and rotate `JWT_SECRET`, then
   restart — all tokens die with the secret.

## 3. Assess & Preserve Evidence

- Preserve: raw request logs (`slog` JSON on stdout — capture the process
  output), `backend/log.db`, any recovered `.logsync` files, device logs.
- Timestamp everything: when found, when contained, when fixed.
- For S1: do NOT delete rows to "clean up" before the analysis is done.
  The anonymized audit trail is the erasure evidence — leave it intact.

## 4. Notify (S1 only)

- Notify the affected users/guardians (bilingual notice, per consent
  language recorded in `consent_records`) and the school principal.
- Nepal law has **no statutory breach-notification duty** (Privacy Act 2075
  is silent on it; enforcement is via District Court civil remedy, §31).
  Voluntary, prompt notification is still the responsible default, and the
  DCCS Directives 2081 make it an obligation where the data sits in a
  licensed data center/cloud — then also notify the National Cyber Security
  Centre.
- Only share what is confirmed. Never fabricate numbers (AGENTS.md §1).

## 5. Recover & Remediate

| Incident | Remediation |
|---|---|
| Lost device with `log.db` | Rotate `JWT_SECRET` (all sessions die), force password reset, monitor for new logins. DB file is SQLite WAL — plaintext; treat as fully disclosed. |
| Lost `.logsync` file | File is plaintext by design (user's own data); the learner work within is real — S1 as above. |
| Queue key lost locally | `sync-keys` in IndexedDB is gone → queued records cannot decrypt. They are **preserved** (never auto-deleted); user must export via sneaker-net path (export fails loudly) or wipe and redo work. S2. |
| Unauthorized export | Revoke all sessions; audit who/what; consider account deletion at user's request (`DELETE /me`). |
| Consent record error | Upsert correct record via `POST /me/consent` with correct version; record the correction in `audit_logs`. S3. |
| Deleted account still recoverable from a copy | A `DELETE /me` erasure triggers WAL checkpoint (TRUNCATE) + VACUUM on the live DB, but **any backup taken before the erasure still contains the rows** (SQLite forensics: DELETE only marks free space; backups are pre-erasure snapshots). Destroy/rotate the affected backup immediately. S1. |
| Retention promise missed | Document in audit log; schedule purge (inactive users > 2y) as a manual SQL op until the scheduled job lands (see roadmap). S3. |

## 6. Post-Incident

- Update the WP-0.1 tracker in `docs/html/07-phased-implementation-plan.html`
  with a "Lessons" note.
- If the incident reveals a code gap, file it in the plan's risk register and
  add a regression test (backend `go test ./...`, frontend `npx jest`).
- Retention commitments (policy `2026-08-v1`): learner data ≤ 2 years after
  last activity; audit logs ≤ 3 years (anonymized on erasure); offline queue
  is user-controlled (90-day guidance, enforced client-side).

## 7. Drill (once per term)

- Restore `backend/log.db` from the last offline copy; verify `/healthz`.
- Simulate an S1: revoke all sessions, rotate the secret, confirm the erasure
  trail in `audit_logs`.
- Verify an export + delete round trip on a test learner; confirm the user
  row is gone and authored content is anonymized (see the Go tests in
  `backend/internal/handler/privacy_test.go` — they are the executable
  version of this drill).

---

## 8. Legal Annex — Nepal law that actually applies

Research base for the WP-0.1 hardening (compiled 2026-08; law moves — the
IT & Cybersecurity Bill 2082 passed the House of Representatives on 14 Aug
2025 and was pending before the National Assembly when this was revised; the
amendment-phase note below was corrected to match the research round).

**Privacy Act 2075 (2018) — in force.**
- §12(2): collection of personal/family data requires the person's consent.
- Guardian/curator consent is required for matters concerning a person under
  18, and must serve the minor's benefit (benefit-of-the-child test).
- §23(4): no collection without informing the purpose — this is why the
  disclosure notice text is shown, and why its hash is recorded
  (`disclosure_hash`).
- §26: purpose limitation — data may only be used for the declared purpose.
- §28: the data subject may request correction/removal.
- §29: penalties — up to 3 years' imprisonment and/or NPR 30,000 fine.
- §31: civil remedy — a claim lies to the **District Court within 3 months**
  of the wrong; the school is a "controller" with exposure here.
- **No dedicated data-protection regulator and no statutory breach
  notification** — do not promise regulators in communications; notify
  affected users directly. (Verified in the research round: no amendment to
  the Act or Privacy Regulation 2077 was enacted through 2026; the Private
  Life (Privacy) Act remains the operative statute.)

**Privacy Regulation 2077 (2021).**
- §5(2)(b): disclosure/publishing of electronically stored personal data
  needs the data subject's **written consent** — keep consent records
  (we do: `consent_records` + `disclosure_hash`).
- Rule 10: the notice must state purpose, content, nature, objective,
  collection time, method/process, and privacy assurances.

**DCCS Directives 2081 (data centers / cloud providers only).** If LOG is
ever hosted in a licensed Nepali data center, breaches must additionally be
reported to the National Cyber Security Centre (NCSC). Self-hosted school
servers are outside these directives but following the practice is free.

**IT & Cybersecurity Bill 2082 (passed House of Representatives 14 Aug 2025;
pending National Assembly as of the research round).** Clause 61 would
require destruction of personal data within **35 days of the purpose being
fulfilled** — a purpose-based retention clock, not inactivity-based (the
30-day figure in some older briefs was an earlier draft; the bill as passed
says 35 days). If enacted, LOG's 2-year inactivity retention must move to
event-based triggers (graduation, withdrawal, request). Digital Rights Nepal
and others have flagged drafting gaps; track the bill before the 2026-08-v2
policy. The current policy (2026-08-v1) already ships the machinery this
bill will require: consent evidence with disclosure hashes, export, and
forensic erasure — the clock change is a policy constant, not a redesign.

**Data Act 2079 (2022)** — the broader data-governance framework the Privacy
Act sits inside; no direct obligations beyond the above for a school SIS.

**COPPA 16 CFR §312.5 (US practice, not Nepali law — adopted voluntarily):**
verifiable parental consent; separate consent for AI/advertising (never
integral); log the hash of the exact disclosure text presented
(`disclosure_hash` does this) so the school can prove what a guardian saw.

**No-disclosure commitment (research round):** LOG does not sell, share, or
disclose learner data to third parties — the consent notice states this
explicitly in both languages, the `GET /me/consent` and export envelopes
carry `sharing.third_party_disclosure: false`, and there is no analytics or
ad-tech integration in the product. Do not weaken this in a future policy
version without a new consent round (a changed notice text produces a new
`disclosure_hash` — existing grants remain valid evidence of the earlier
text, per COPPA practice).