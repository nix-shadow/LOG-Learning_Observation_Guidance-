# Partnership Lanes (WP-3.5 · RC-11)

How LOG opens its learning content to school partners, districts, and the
MoE — without ever weakening the child-data protections the platform is built
on. Everything here is a **template and policy**, not a claim of active
partnerships: the lanes become real only when a partner signs the MoU and
content passes the WP-3.1 license check.

Policy version: **2026-08-v1** (aligned with the privacy policy version in
`backend/internal/domain/privacy.go`).

---

## 1. Why partnership lanes exist

LOG is built for regions with spotty internet and scarce local content. A
school, a district office, or the MoE may want to:

- adopt LOG content inside their own teaching materials,
- contribute curriculum-aligned content back into LOG's library,
- run the QR poster pilot (WP-3.3) across their schools,
- host a shared content library that multiple schools read offline.

Every lane is governed by the same rule: **a learner's data never leaves the
LOG platform, and third-party content enters only with a validated OER
license and honest attribution.** There is no "data for content" barter.

## 2. MoU template (school / district / MoE)

Adapt the boxes below to the partner; the numbered clauses are non-negotiable
minimums and must not be weakened.

**Memorandum of Understanding — LOG Learning Platform Partnership**

Parties: **LOG Learning Team** ("LOG") and **[Partner name]** ("Partner").

Effective date: **[date]**. Term: **[12] months**, renewable.

1. **Purpose.** Partner adopts LOG content and/or the QR poster pilot for
   the benefit of learners in **[schools / district / province]**.

2. **Learner data stays on LOG.** LOG processes learner activity data only
   within the LOG platform and its offline queue. Partner receives
   **aggregate, anonymized** summaries (e.g. "module completion rates") where
   LOG has an API for them — never raw learner records, and never any
   guardian/student names, phone numbers, or IPs.

3. **Consent first.** Every learner account requires an evidenced guardian
   consent grant (per LOG's `consent_records` policy `2026-08-v1`, including
   the `disclosure_hash` evidence check). Partner agrees not to create
   learner accounts without the same consent, and to follow the Nepal
   Privacy Act 2075 and applicable school-data rules for any data it holds
   outside LOG.

4. **Retention and erasure.** LOG retains learner data at most 2 years after
   the learner's last activity and erases on request (`DELETE /me`). Partner
   agrees to delete any derived copies of learner data it may hold on the
   same or shorter schedule and to honor an erasure request within 14 days.

5. **No re-sharing.** Partner will not sell, share, or re-publish learner
   data or any personally identifying information obtained through the
   partnership.

6. **Content in = content licensed.** Content Partner contributes must be
   (a) original work Partner may license, or (b) already-licensed OER the
   Partner may redistribute. Every contribution is reviewed by the LOG OER
   import pipeline (WP-3.1), which **rejects** unknown, empty, or
   un-attributed licenses — the platform never guesses a license.

7. **Content out = attribution honored.** LOG content the Partner prints or
   hosts keeps its license + attribution line (see §3). Derivative works
   carry the same license.

8. **QR pilot (optional).** If the pilot runs: posters are LOG-provided, QR
   codes point at LOG-hosted `/qr/<activityId>` pages, and pilot measurement
   (scans, starts, start-rate) is aggregate only. No learner identity is
   recorded by a scan.

9. **Incident response.** Any suspected data incident is reported to both
   parties within 72 hours and handled per LOG's `docs/PRIVACY_RUNBOOK.md`.

10. **Termination.** Either party may end the MoU with 30 days' notice.
    On termination, Partner deletes learner-derived data per clause 4 and
    stops printing/hosting LOG content, or removes it from circulation
    within 90 days.

**Signatures.** LOG: **[name/role]** · Partner: **[name/role]**.

## 3. Attribution & remix rules

These rules are enforced by the OER import pipeline (`backend/internal/
service/oer_service_impl.go`) and rendered by the frontend catalog:

- **Allowed licenses** (`backend/internal/domain/domain.go`
  `OERAllowedLicenses`): CC BY 4.0, CC BY-SA 4.0, CC BY-NC 4.0, CC BY-NC-SA
  4.0, CC0 1.0 (Public Domain), or **"Own work (LOG team)"** for original
  LOG-authored content.
- **Attribution required** for anything not "Own work": the pipeline rejects
  third-party rows with an empty attribution, with a per-row reason.
- **SA (share-alike) rule:** content licensed CC BY-SA may only be remixed
  into materials released under a compatible share-alike license — the
  license itself says so, and the platform honors it on export.
- **NC (non-commercial) rule:** CC BY-NC content is for non-commercial use;
  a partner printing posters for free distribution is fine, selling them is
  not.
- **Credit in the UI:** every activity card and lesson page renders its
  license + attribution line (WP-3.1), so credit travels with the content.

## 4. Shared-library model

A partner-hosted offline library (e.g. a school-server content bundle) works
like this:

1. LOG exports a content pack through the admin **OER import/export** seam.
   Every row already carries license + attribution; nothing is exported
   without them.
2. The partner's server serves the pack over the school LAN. LOG frontends
   on that LAN warm their offline cache from it (the same cache the QR
   landing page warms — WP-3.3).
3. If the partner contributes content back, it enters the same pipeline:
   license-checked, attribution-required, audited (`oer.import` audit rows).
4. The **library is additive, not extractive**: contributing to it never
   grants the partner access to learner data.

## 5. Contributor credits

- Every activity stores its author/attribution in the `attribution` field,
  shown in the catalog and on the lesson page.
- Original LOG-authored units credit "LOG Learning Team (original content)".
- Third-party OER keeps its original author's credit — the platform never
  replaces it with its own name.
- Audit log entries record every import (`oer.import`) with the pack name
  and per-row outcome, so the provenance trail is permanent.

## 6. Review checklist

- [ ] MoU template reviewed by legal contact for the jurisdiction (Nepal
      Privacy Act 2075; track the IT & Cybersecurity Bill 2082 for the next
      policy version per `docs/PRIVACY_RUNBOOK.md` §8).
- [ ] Attribution/remix rules match the allowed-license allowlist in code.
- [ ] No clause above promises any access to learner-level data.
- [ ] Pilot measurement documented as aggregate-only (WP-3.3 stats API).