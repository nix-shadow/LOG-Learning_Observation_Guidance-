# LMS & EdTech Comparison and Enhancement-Idea Report for LOG

**Status:** Research only — no code changed.
**Audience:** LOG product/engineering (Next.js 14 + Go/Gin + SQLite; offline-first for Nepali schools).
**Date:** 2026-08-19
**Method:** ~20 web searches (2026 sources); every claim cites its source domain inline. Complements `docs/PRODUCT_RESEARCH.md` (SIS/ERP inventory, SEE grading, IEMIS, privacy), which this report does not repeat.

---

## 1. Feature Matrix

Legend: ✅ native/strong · ◐ partial/add-on · ❌ absent. LOG-current reflects the codebase (`frontend/src/lib/api.ts`, `backend/internal/...`, docs); "Recommended" is what this report proposes.

| Feature | Moodle | Canvas | Google Classroom | Kolibri | Duolingo | LOG (current) | Recommended for LOG |
|---|---|---|---|---|---|---|---|
| Role-based access | ✅ granular roles (raccoongang.com) | ✅ | ✅ | ✅ learner/coach/admin | ✅ | ✅ ADMIN/MODERATOR/STUDENT | + PARENT read-only role |
| Lesson/content authoring | ✅ rich editor, H5P, SCORM | ✅ built-in authoring | ◐ Drive attachments | ✅ Kolibri Studio channels (learningequality.org) | ✅ in-house | ◐ micro-modules (admin-seeded) | Teacher lesson authoring |
| Assessments / question banks | ✅ quiz engine, randomization, banks | ✅ | ◐ via Forms (getclasswise.com) | ✅ lessons + quizzes (offlineinternet.org) | ✅ adaptive drills | ◐ scored activities | Question bank + offline quiz packs |
| Gradebook | ✅ aggregation, export (docs.moodle.org) | ✅ SpeedGrader, weighting (instructure.com) | ❌ no built-in (getclasswise.com) | ◐ coach reports | ❌ | ❌ | Class gradebook + NEB letter grades |
| Assignments & submissions | ✅ multiple types | ✅ | ✅ core workflow | ✅ assign resources | ❌ | ✅ assignments | Offline submission queue |
| Forums / discussion | ✅ forums, wikis, workshops | ✅ | ✅ class stream | ◐ | ❌ | ❌ | ❌ (async notice board only — see §5) |
| Announcements / notifications | ✅ SMS/email/push | ✅ | ✅ | ❌ | ✅ push nudges | ✅ announcements | SMS fallback via OTP gateway |
| Badges / credentials | ✅ Open Badges (issuebadge.com) | ✅ | ❌ | ◐ | ✅ awards + personal records (trophy.so) | ◐ cosmetic badge only | Competency badges w/ criteria |
| Streaks / XP / leaderboards | ◐ | ❌ | ❌ | ◐ points | ✅ leagues of ~30, XP, freezes (trophy.so) | ◐ streak + score, no leaderboard | Class-scoped weekly XP + streak freezes |
| Teacher analytics | ✅ completion tracking, reports | ✅ detailed analytics (getclasswise.com) | ❌ | ✅ coach dashboard real-time (learningequality.org) | ◐ | ◐ moderator roster (streak status) | Per-class drill-down, honest zeros |
| Attendance | ❌ core | ◐ Ultra attendance (gptzero.me) | ❌ | ❌ | ❌ | ❌ | Offline-first daily attendance + SMS |
| Parent/guardian portal | ✅ | ✅ | ◐ guardian summaries | ❌ | ❌ | ❌ | Shareable/print progress reports |
| Offline-first mode | ✅ app caches + offline quizzes (warwick.ac.uk) | ◐ HTML/ePub export (lddi.educ.ubc.ca) | ◐ limited | ✅ built for it (learningequality.org) | ✅ mobile-offline | ✅ IndexedDB cache + queue + .logsync | Strengthen; LAN peer sync |
| Peer-to-peer / LAN sync | ❌ | ❌ | ❌ | ✅ peer import + facility sync (kolibri.readthedocs.io) | ❌ | ❌ | Learn-only devices sync on school Wi-Fi |
| Spaced repetition | ◐ plugins | ❌ | ❌ | ◐ | ◐ spaced practice | ❌ | SM-2 review decks |
| Mastery / competency tracking | ✅ competencies, plans | ✅ Mastery Connect (instructure.com/mastery) | ❌ | ✅ mastery-based recommendations (aiforcause.org) | ✅ | ◐ score/streak only | Skill-level competency model |
| Adaptive paths | ✅ conditional release | ✅ | ❌ | ✅ local recommendation algo (aiforcause.org) | ✅ | ◐ local rule engine (`adaptiveEngine.ts`) | Data-driven next-activity suggestions |
| Mobile-first PWA | ✅ app | ✅ app | ✅ app | ✅ Android app (Android 6+) | ✅ | ✅ PWA + SW | Offline quiz UX on low-end Android |
| Nepali localization | ✅ 100+ langs | ✅ | ✅ | ◐ 17 langs (no Nepali) | ✅ | ❌ planned (ENHANCEMENT #3) | Nepali/Devanagari toggle |
| CSV roster import/export | ✅ import/export | ✅ OneRoster SIS sync (notion4teachers.com) | ◐ CSV export only | ✅ facility import | ❌ | ◐ export only (`ExportStudentsCSV`) | CSV roster import for EMIS sheets |
| Printable progress reports | ✅ certificates via plugin | ✅ | ❌ | ◐ | ❌ | ❌ | A4 PDF report, parent signature line |

Takeaway: LOG already out-offlines Moodle/Canvas/Classroom and matches Kolibri's core offline premise; its gaps are exactly the columns where mainstream LMS (gradebook, parent access, badges, analytics) and Kolibri (content packs, LAN sync, CSV provisioning) score — i.e., the "Recommended" column.

---

## 2. Offline-First Patterns from Kolibri / KA Lite Worth Adopting

Kolibri is the reference implementation of offline-first education (220+ countries; deployed by UNHCR/UNICEF/World Vision; Cameroon study: +14% math vs control — aiforcause.org, learningequality.org). Specifics LOG should copy:

1. **Content channels + USB sneakernet (already 80% there).** Kolibri's core content flow: import channels from Kolibri Studio while online → *export selected channels to an external drive* → import on fully offline devices (`kolibri.readthedocs.io` channels guide). LOG's `.logsync` files are the same idea; extend them from a flat queue dump to *curated curriculum packs* (channel = subject+grade+term) so a principal can download a "Grade 6 Math Term 1" pack on one connected device and carry it to school. Export/import with checksums and progress UI, exactly as Kolibri's Task Manager does.
2. **Peer import over local Wi-Fi (the biggest single upgrade).** Kolibri 0.11+ auto-discovers other Kolibri instances on the LAN and imports resources from them ("Import from local network or internet"; explicit IP fallback when firewalls block discovery — `kolibri.readthedocs.io`). For LOG: a school server (or teacher's phone hosting the PWA) becomes the sync hub; student devices sync attendance/quizzes/guidance over school Wi-Fi *without internet*. Kolibri proves this exact topology: "provisioned learn-only devices sync automatically with full-facility devices on the same local network" (community.learningequality.org).
3. **Certificate-based sync engine.** Kolibri's facility sync uses Morango, a pure-Python DB replication engine with certificate-based auth ("protects privacy and integrity of data") applied atomically (`kolibri-dev.readthedocs.io`). LOG's sync-queue already has idempotency + 401-preservation; the lesson is to make the sync payloads *partitioned per facility* and replayable, and to never drop queued records on auth failure (LOG already complies — AGENTS.md §3).
4. **Coach dashboard as the teacher anchor.** Kolibri's teacher loop: before class → assign from pre-selected content; during → real-time coach dashboard; after → cohort reports (progress, quiz scores, time-on-material) (`learningequality.org/about-kolibri`). LOG's moderator roster is the seed; the missing pieces are the *during/after* loop (see §4: Guidance Reply Loop, Teacher Class Workbench).
5. **The "mobile school van" model.** KA Lite's documented use cases: computer labs slowly syncing a central server via cell/satellite or USB keys; mobile school vans carrying a server between remote schools, syncing when connectivity appears (github.com/learningequality/ka-lite). This maps to Nepali reality (roadless hill districts): one teacher with a laptop + `.logsync` files rotating through schools is a legitimate deployment pattern — keep imports/exports *file-based* so no network of any kind is required.
6. **Raspberry Pi / low-power server as Wi-Fi AP.** KA Lite demonstrated a Pi serving 35 tablets simultaneously streaming video (raspberrypi.com); Kolibri documents low-cost/legacy hardware and Android-6+ clients (learningequality.org). LOG should treat the school server role (a phone or Pi hosting the PWA + local API) as a first-class topology, not an afterthought.
7. **Content prep for offline is an engineering discipline.** Kolibri Studio compresses videos, adapts interactive sims to run without web services, and standardizes metadata so recommendations work across providers (aiforcause.org). For LOG: keep guidance deterministic (AGENTS.md: no hallucination), keep payloads small, and version the metadata so the offline cache invalidation list (dashboard/learning-journey/chart-data) stays correct.
8. **Roster provisioning via CSV.** Kolibri facilities import users en masse; mainstream LMS sync rosters via OneRoster/SIS (Canvas and Schoology do bidirectional sync — notion4teachers.com). Nepal's schools already maintain Excel EMIS sheets (PRODUCT_RESEARCH.md §2.1); CSV import is the lowest-friction onboarding path.
9. **Rumie's microlearning packaging.** Rumie "Bytes" are 6-minute, mobile-first lessons, ~20% more effective at retention than traditional formats and ideal where "a mobile phone is often their 'computer'" (about.rumie.org). LOG's micro-modules already fit this; the takeaway is *keep lessons ≤ 6–10 minutes* and design for one hand / one thumb.

---

## 3. Gamification + Retention Mechanics That Fit a Supportive-Language Principle

Duolingo's mechanics are the most documented in edtech (40M+ daily users; DAU grew ~5M→40M 2020–2024 while gamification became central — trophy.so). The catch: several mechanics *depend on loss aversion and public shame*, which conflicts with LOG's supportive-language rule (AGENTS.md §1: "This area could use more practice", never "You failed"). Adopt the mechanics, reframe the psychology:

| Duolingo mechanic | Evidence (2026) | LOG reframe (supportive) |
|---|---|---|
| **Streaks** | Streaks leverage loss aversion ("protecting an investment"); users with 14+ day streaks retain 5x better (wikiproblem.com) | Keep streaks, but add **streak freezes** (Duolingo: 17.2 avg days on streak with freeze vs 11.6 without — trophy.so) framed as "grace days," never as a penalty. Missed days pause, don't shame. |
| **XP as immediate feedback** | XP awarded *before* the lesson screen closes = reward tightly coupled to behavior (trophy.so) | Score already exists; award XP on completion screen with a celebratory but honest animation. XP should be the common currency feeding streaks, class leagues, and badges simultaneously. |
| **Segmented leagues, not global** | Weekly leagues of ~30 learners; segmented competition outperforms global; demotion risk is the main re-engagement signal (trophy.so) | **Class-level leaderboards only** (user explicitly requested). No global rankings — privacy + discouragement. Optionally make leagues *promote-only* (no demotion messaging) to keep it positive. |
| **Badges tied to milestones** | Badge earners 30% more likely to finish a course (orizon.co); day-one achievements retain 33.4% vs 20.4% (trophy.so) | Competency badges with **explicit criteria** (see §4 #9). Make first badge achievable in the first session. |
| **Daily quests / micro-goals** | Daily Quests raised DAU 25% (orizon.co) | "Today's goal: 1 lesson + 1 review." One-tap goal, achievable in 5 minutes on shared phones. |
| **Customizable daily goals** | Duolingo lets users set goals (trophy.so) | Default 1 lesson/day — lower than you think; consistency beats volume. |

**Two caution flags from the research:**
- Rewards *disconnected from meaningful outcomes* feel artificial and reduce motivation (arXiv:2203.16175, via besitoscorp.com). Badges/XP must always trace to real progress data (LOG's "No Fabricated Fallbacks" already guarantees this).
- Loss-aversion copy ("Don't lose your 50-day streak!") is effective but borderline manipulative for children. LOG should use *encouraging* nudges ("Your streak is growing — keep the rhythm going!") and treat streak freezes as generosity, not a shop item.

---

## 4. New Feature Ideas for LOG (23)

Effort: S < 1 week · M 1–4 weeks · L 1–3 months (single dev). Items marked (P-xx) already exist in `docs/ENHANCEMENT.md` / `docs/PRODUCT_RESEARCH.md` roadmaps — listed here as confirmation with new research backing.

**Teacher workflow**

1. **Teacher Class Workbench** — Replace the placeholder `/api/moderator/classes` with a real daily screen: roster with per-student status (streak, pending guidance, quiz accuracy), assignment creation, one-tap guidance replies. *MODERATOR · L · High.* Closes LOG's one-directional student→teacher loop (ENHANCEMENT #4) and mirrors Kolibri's coach dashboard, the anchor that made Kolibri effective only when paired with teacher training (aiforcause.org).
2. **CSV Roster Import** — Upload the school's existing Excel/EMIS sheet (name, phone, class) to bulk-create users + enrollments; dry-run preview showing "312 rows, 11 errors" before commit; honest per-row error report. *ADMIN · S–M · High.* Kolibri provisions facilities en masse; Canvas/Schoology sync rosters via OneRoster (notion4teachers.com). Nepali schools already keep EMIS sheets (PRODUCT_RESEARCH.md §2.1) — this is the #1 onboarding unlock and the natural inverse of LOG's existing `ExportStudentsCSV`.
3. **Offline Question Bank & Quiz Packs** — Teachers download a question-bank pack (subject/grade), assemble quizzes offline, and distribute via the sync queue; students attempt offline, results flush on reconnect. *MODERATOR+STUDENT · M · High.* Moodle's mobile app does exactly this: cache the quiz, attempt offline, auto-sync results on reconnect (docs.moodle.org "Moodle Mobile quiz offline attempts"; warwick.ac.uk). Same pattern as Kolibri lesson/quiz creation.
4. **Attendance (offline-first)** — Daily mark-all screen (one tap per student, present/absent/late), queued through the existing sync layer, parent SMS alerts when connectivity returns. *MODERATOR · M · High.* Core module of every SIS (PRODUCT_RESEARCH.md #4); the highest-frequency teacher workflow; fits the sync-queue perfectly (already P1-4). Note: Nepal's DoE circular *bans classroom phone use up to Plus Two* (education-profiles.org) — attendance marking must be a teacher-side workflow, not a student app feature.
5. **Teacher Onboarding Kit** — Nepali wizard + printable step-by-step guide + sandbox/demo-data mode + a 90-minute session plan. *ADMIN/MODERATOR · S · Med.* 64% of teachers cite equipment/training gaps (PRODUCT_RESEARCH.md P1-8); Kolibri's "training of trainers" toolkit is the model (learningequality.org).
6. **Guidance Reply Loop** — Students can ask "why?" on a guidance note and teachers reply; replies sync via the queue. *MODERATOR+STUDENT · M · High.* Turns one-way guidance into mentorship; mirrors Canvas's "Message Students Who..." (instructure.com).

**Parent / guardian visibility**

7. **Parent/Guardian Progress Report (print/shareable PDF)** — One-click A4 report per child: streak, lessons completed, quiz trend, NEB letter-grade projection, teacher note, parent signature line; printable or shareable via WhatsApp. *MODERATOR/ADMIN · S · High.* Parent portals are baseline in every SIS/LMS (Schoology's portal "justifies the switch" — notion4teachers.com); in Nepal, print/PDF + WhatsApp beats a login the parent never opens. Only real data (No Fabricated Fallbacks).
8. **PARENT Role (read-only)** — New RBAC tier: parent sees only own children's progress, attendance, announcements. *ADMIN+parent · M · Med.* Requested; already flagged in PRODUCT_RESEARCH.md §4.3 (parent-identity management is the hard part — verify via the same OTP phone channel).
9. **Reconnect Digest (SMS/push)** — "While you were away: 3 new guidance notes, 2 assignments due." *STUDENT · S · Med.* Already ENHANCEMENT #5; low-bandwidth periodic check-in pattern, leverages the cache. Duolingo's timed nudges show reminders drive re-engagement (wikiproblem.com).

**Learning science**

10. **Spaced Repetition Review Deck (SM-2)** — After completing a micro-module, key facts become flashcards scheduled by SM-2 (interval doubling on success, reset on miss, easiness factor ≥ 1.3 — dev.to "how spaced repetition works", activerecalling.com). Fully offline (pure client math). *STUDENT · M · High.* Ebbinghaus forgetting curve is the most evidence-based retention lever; SM-2 is a few lines of math (no ML, no connectivity) and works per-card even on shared phones.
11. **Competency Badges with Criteria** — Badges bound to skill criteria (e.g., "Fractions — 3 consecutive quizzes ≥ 80%"): criteria URL + description in metadata, Open-Badges-style export later. *All · M · Med-High.* Moodle's badges are the standard (issuebadge.com); badges with real criteria are the anti-fabrication-safe form of credentialing (digitalpromise.org micro-credentials report).
12. **Micro-credential Track** — Stackable badge chains per subject term (3 badges → "Term 1 Math Champion" credential, printable). *STUDENT+ADMIN · M · Med.* UNESCO defines micro-credentials as evidence-based records of focused achievement (journals.ku.edu); for Nepal, a print credential a student can take to parents is culturally legible.
13. **Adaptive Review Scheduler** — Rule-based "you're ready for the next module / here's a review" from real attempt data (score, elapsed time, retry count), extending `adaptiveEngine.ts` from simulated to data-driven. *STUDENT · S–M · Med.* Kolibri does exactly this locally: recommendations based on demonstrated mastery, no cloud AI (aiforcause.org).

**Gamification (supportive framing per §3)**

14. **Class Leaderboard (weekly XP, class-scoped)** — Weekly league within the class, promote-only tiers, opt-out per student, no global rankings. *STUDENT · S · Med.* Duolingo's ~30-learner segmented leagues (trophy.so); class scope respects privacy and keeps the comparison friendly.
15. **Streak Freeze / Grace Days** — 2 freezes per month auto-applied on missed days; "your rhythm is protected" messaging. *STUDENT · S · Med.* Freeze mechanics materially extend streaks (17.2 vs 11.6 days — trophy.so) without loss-aversion guilt.
16. **Daily Quest / Micro-Goal** — "Complete 1 lesson + 1 review" one-tap daily goal; completing it earns a small XP bonus and protects the streak. *STUDENT · S · Med.* Daily Quests +25% DAU (orizon.co); tiny goals fit 5-minute usage windows and shared-device constraints.
17. **Milestone Moments** — Confetti + supportive copy at milestones (10-day streak, first 100 XP week, first badge); the existing framer-motion confetti reused honestly. *STUDENT · S · Med.* Day-one/early achievements drive retention (trophy.so); keeps celebrations tied to real data.

**School operations & reporting**

18. **Principal Analytics Drill-Down** — From the admin dashboard: per-class cohort trends (attendance, mastery, guidance uptake, streak decay) with honest empty/zero states (AGENTS.md §1.4). *ADMIN · M · High.* Reporting is the leadership value of every LMS (Synergy SIS via PRODUCT_RESEARCH.md); Kolibri's Data Portal aggregates across offline installs (learningequality.org).
19. **IEMIS / Flash Export** — Export learner/teacher/attendance data aligned to IEMIS fields so the school's mandated Flash Report is generated from clean data. *ADMIN · M · Med.* Already P2-4; IEMIS accuracy is a documented pain (Flash I 2081 verification, PRODUCT_RESEARCH.md §2.1).
20. **Content Packs (curriculum channels via .logsync)** — Principal downloads "Grade 6 Math, Term 1" pack (lessons + quizzes + review decks) on a connected device, carries it to school on USB, imports offline; extension of `syncExport.ts` with checksums and progress UI. *ADMIN · M · Med.* Kolibri's channel export/import model (kolibri.readthedocs.io).

**Nepal fit & platform**

21. **Nepali Language Toggle** — Full Nepali/Devanagari UI for student-facing screens (and parent reports), English default for teachers. *All · M · High.* Already P0-2/ENHANCEMENT #3; the strongest adoption lever: 96% of Nepalis access internet via mobile, and local-language content is a documented exclusion factor (thediplomat.com; digitalrightsnepal.org). E-Paath proves Nepali content works at scale (olenepal.org).
22. **SMS OTP Gateway** — Connect a Nepali SMS provider so field login actually works (today OTP is demo-only in server logs). *All · S · High.* Already P0-3; SMS is the only practical auth channel for schools (PRODUCT_RESEARCH.md). Digital Nepal Framework 2.0 explicitly targets mobile-first services (hamroniti.com).
23. **Device-Sharing Safe Mode** — Auto-logout on close, session codes, "log out all devices" (revoke jti). *All · S · High.* Already ENHANCEMENT #7; Nepali families share phones/browsers (digitalrightsnepal.org) and the DoE phone ban means school-device sharing is the norm — cross-account cache leaks are a real risk (ENHANCEMENT 1.6).

**Prioritization note:** highest total value = teacher workbench + CSV import (2) → offline quizzes (3) + attendance (4) → SM-2 reviews (10) + Nepali toggle (21) → parent PDF (7) → badges (11). The class leaderboard (14) and streak freezes (15) are cheap and land well early.

---

## 5. "Don't Do" Anti-Patterns

1. **Video-heavy content without progressive download / offline packaging.** Nepal's median mobile *upload* is ~14 Mbps and data packs cost NPR 300–500 per 5GB; rural students walk 30+ minutes for a signal and connections drop on power cuts (digitalrightsnepal.org; thediplomat.com). Streaming-first content = non-functional learning for the exact audience LOG serves. Kolibri's discipline: videos compressed and pre-formatted for offline delivery (aiforcause.org). If video ships at all, it ships as a small, preloaded file in a content pack — never as a required stream.
2. **Real-time chat / live classes.** Synchronous tools are the first thing low-bandwidth guides cut: UBC's low-bandwidth guidance recommends async materials, captions/transcripts over video, offline-exportable content (lddi.educ.ubc.ca). A "live class" feature that works for Kathmandu private schools is dead-on-arrival in Karnali (68.5% household internet — ekantipur.com). Use the queue-based announcement + digest pattern instead; it works at any bandwidth.
3. **Heavy cloud AI integration.** AGENTS.md forbids hallucinated guidance; cloud LLM calls fail offline by definition and cost real money per API call at Nepali data prices. Kolibri's lesson is the opposite: tiny local models (recommendations, early-warning) that run on a Raspberry Pi (aiforcause.org). Keep guidance deterministic/rule-based locally; any AI stays optional, offline-capable, and never the source of learner metrics.
4. **Global leaderboards / public rankings.** Global or cross-school rankings (a) expose child data, (b) demoralize the majority, and (c) violate the supportive-language principle — Duolingo itself uses *segmented* leagues of ~30, not global (trophy.so). LOG's leaderboard, if it exists, is class-scoped, promote-only, and opt-out.
5. **Monetized gamification (gems, paid streak repair, loot boxes).** Premium-currency shops pressure low-income families and disconnect rewards from learning — gamification research shows rewards decoupled from meaningful outcomes feel artificial and backfire (arXiv:2203.16175). Streak freezes should be free generosity (supportive), not a revenue line.
6. **Any feature that gates on the cloud "occasionally."** iPrep's field lesson: platforms "marketed as low-bandwidth" that still require occasional cloud calls for auth/analytics/content are unacceptable in no-connectivity schools (idreameducation.org). Every LOG feature must have a complete offline path — LOG's own "No Fabricated Fallbacks" and 401-preserving queue are the guardrails; don't erode them with convenience features that assume a connection.

---

## Sources (domains cited above)

moodle.com · docs.moodle.org · raccoongang.com · issuebadge.com · instructure.com · instructure.com/mastery · notion4teachers.com · getclasswise.com · gptzero.me · warwick.ac.uk · lddi.educ.ubc.ca · learningequality.org · kolibri.readthedocs.io · kolibri-dev.readthedocs.io · community.learningequality.org · offlineinternet.org · aiforcause.org · raspberrypi.com · github.com/learningequality/ka-lite · about.rumie.org · wikiproblem.com · orizon.co · trophy.so · besitoscorp.com · arXiv:2203.16175 · olenepal.org · education-profiles.org · ekantipur.com · thediplomat.com · digitalrightsnepal.org · hamroniti.com · ictframe.com · datareportal.com · thehimalayantimes.com · idreameducation.org · digitalpromise.org · journals.ku.edu · dev.to (SM-2) · activerecalling.com · studyglen.com