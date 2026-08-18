# Product Research — Production-Grade School/College Platform Expansion for LOG

**Status:** Research only — no code changed.
**Scope:** What a production-grade school/college management + learning platform needs, researched for expanding LOG (Learning Observation Guidance), an offline-first edtech platform for Nepali schools.
**Date:** 2026-08-16
**Method:** 12+ web searches across SIS/LMS vendor documentation, Nepal MoEST/NEB/CEHRD sources, World Bank/UNESCO/UNICEF/CISA reports, offline-first engineering guides, and privacy regulation primary sources. Every claim below carries its source URL.

---

## Table of Contents

1. [School/College Platform Feature Inventory](#1-schoolcollege-platform-feature-inventory)
2. [Nepal-Specific Requirements](#2-nepal-specific-requirements)
3. [Offline-First / Low-Bandwidth Best Practices](#3-offline-first--low-bandwidth-best-practices)
4. [Production-Grade Checklist](#4-production-grade-checklist)
5. [Prioritized Feature Roadmap for LOG](#5-prioritized-feature-roadmap-for-log)
6. [Top 10 Highest-Value Feature Recommendations](#6-top-10-highest-value-feature-recommendations)

---

## 1. School/College Platform Feature Inventory

Industry vendors (openSIS, OpenEduCat, EdFleet, Classter, Edupoint Synergy, eSchoolApp, ScholaSystem) converge on a well-defined module list. The industry distinguishes two families: a **Student Information System (SIS)** (student-centered: records, enrollment, gradebook, attendance, transcripts, parent portals) and a **School Management System / ERP (SMS)** (institution-centered: billing, staff HR, timetables, facilities, procurement, multi-campus). Most commercial platforms are all-in-one SIS + SMS + LMS, sharing one data foundation so a student enrolled in admissions is automatically present in billing, attendance, and parent communication ([Classter](https://www.classter.com/blog/edtech/student-information-system-sis-vs-school-management-system-sms-what-do-you-need-in-2026/)).

### 1.1 Full feature inventory (with LOG relevance)

| # | Module | What it covers | Core for LOG? |
|---|--------|---------------|---------------|
| 1 | **Student records / SIS spine** | Single complete profile per student: demographics, guardians, class history, documents, fee status. "Every other module is built on this one" ([EdFleet](https://edfleet.com/blog/school-management-system-features/), [openSIS](https://opensis.com/features)) | **Core** — without a real learner record, per-learner tracking has no anchor |
| 2 | **Admissions / enrollment** | Online applications, document upload, merit lists, enrollment confirmation ([openSIS](https://opensis.com/features), [OpenEduCat](https://openeducat.org/open-source-education-erp/)) | Nice-to-have (P2) — LOG is learning-first, not admissions-first |
| 3 | **Class & section management** | Grade/class/division/section hierarchy, room & teacher assignment, schedule conflicts ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/)) | **Core** — teacher roster/class view already assumes classes exist |
| 4 | **Attendance** | Daily + period-wise recording, biometric/RFID options, absence alerts to parents, compliance reporting ([OpenEduCat buyer's guide](https://openeducat.org/articles/school-management-software-buyers-guide/), [eSchoolApp](https://eschoolapp.com/features/)) | **Core** — highest-frequency teacher workflow; offline-first data entry pattern fits LOG perfectly |
| 5 | **Gradebook & assessments** | Digital gradebook, weighted categories, letter/percentage/GPA/CGPA scales, auto report cards ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/), [Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)) | **Core** — LOG already tracks quiz scores; extend to NEB letter grading (see §2.2) |
| 6 | **Exams & report cards** | Exam creation, scheduling, grading, results publication, printable report cards ([OpenEduCat](https://openeducat.org/open-source-education-erp/)) | **Core** — SEE context is Nepal's defining exam moment (§2.2) |
| 7 | **Timetabling** | Automated scheduling with teacher availability, room capacity, lab constraints; "weeks → hours" ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/), [ScholaSystem](https://www.scholasystem.com/core-modules)) | Nice-to-have (P2) |
| 8 | **Announcements / communications** | SMS, email, push; fee reminders, attendance alerts, emergency comms ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/)) | **Core** (SMS/offline-queued announcements fit low-connectivity reality) |
| 9 | **Parent/guardian portal** | Parents check grades, attendance, fee status, announcements; reduces office visits/phone calls ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/), [openSIS](https://opensis.com/features)) | Nice-to-have (P1/P2) — high value for adoption, but needs guardian identity management |
| 10 | **Reports & analytics (principal/HOD)** | Enrollment summaries, grade distributions, fee collection, attendance trends; custom reports ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/), [Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)) | **Core** — LOG admin dashboard is the seed; add per-class drill-down |
| 11 | **Academic calendar** | Terms, holidays, exam dates, events with app reminders ([Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)) | Nice-to-have (P2) |
| 12 | **Library management** | Catalog, issue/return, fines, digital library ([OpenEduCat](https://openeducat.org/open-source-education-erp/), [eSchoolApp](https://eschoolapp.com/features/)) | Nice-to-have (P2); Nepal has E-Pustakalaya precedent (§2.4) |
| 13 | **Fees / billing** | Invoicing, online payments, installments, late fees, scholarships; integrates with student DB ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/), [EdFleet](https://edfleet.com/blog/school-management-system-features/)) | Nice-to-have (P2) — biggest revenue case for schools, but out of LOG's learning core |
| 14 | **Staff / HR & payroll** | Employee records, leave, payroll, contracts ([OpenEduCat](https://openeducat.org/open-source-education-erp/), [ScholaSystem](https://www.scholasystem.com/core-modules)) | Out of scope for LOG v1 |
| 15 | **Transport / hostel / inventory** | Bus routes, vehicle tracking, hostel, canteen ([eSchoolApp](https://eschoolapp.com/features/), [eAcademy](https://eacademynepal.com/)) | Out of scope for LOG v1 |
| 16 | **Teacher lesson plans & LMS integration** | Lesson planning; two-way sync between SIS and LMS (courses, sections, grades, schedules) ([openSIS](https://opensis.com/features)) | **Core** — LOG's courses/activities are the LMS side; lesson-plan authoring for teachers is the missing workflow |
| 17 | **Role-based access** | Granular per-role permissions, no "data browsing" ([EdFleet](https://edfleet.com/blog/school-management-system-features/), [Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)) | **Already in LOG** (ADMIN/MODERATOR/STUDENT) |
| 18 | **Data security, backups, audit logs** | Encryption, backup/restore, audit trails, compliance ([EdFleet](https://edfleet.com/blog/school-management-system-features/), [Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)) | **Already partially in LOG** (see §4) |
| 19 | **Mobile apps (teachers & parents)** | Mobile-first teacher/parent experience ([EdFleet](https://edfleet.com/blog/school-management-system-features/)) | **Core** — LOG is PWA-first; mobile is the primary channel |
| 20 | **Discipline management** | Incident logs, referrals, health records ([Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS), [openSIS](https://opensis.com/features)) | Nice-to-have (P2) |

### 1.2 What a "digital learning platform" like LOG should NOT clone (scope discipline)

- Full **ERP** (payroll, procurement, facilities, multi-campus) is a different product class ([Classter](https://www.classter.com/blog/edtech/student-information-system-sis-vs-school-management-system-sms-what-do-you-need-in-2026/)). Vendors sell 73-module suites ([OpenEduCat](https://openeducat.org/open-source-education-erp/)); LOG should stay a *learning + classroom-operations* platform.
- The correct expansion axis for LOG is: **SIS-lite (learner records + classes) → teacher classroom workflows (attendance, gradebook, assignments, exams) → school-level analytics & communication → parent access** — roughly matching openSIS's documented SIS ↔ LMS integration pattern (courses/sections/grades synced both ways) ([openSIS](https://opensis.com/features)).
- Nepali competitors confirm the market expectation set: local vendors (Vidyapith, YEDU, Project School, eAcademy, SupportMeNepal, e-Billing) all bundle **fee management, SMS parent notifications, attendance, exams, and NEB grading** as baseline ([Vidyapith](https://vpit.com.np/), [YEDU](https://www.yedu.app/), [Project School](https://webbank.com.np/project-school/), [eAcademy](https://eacademynepal.com/), [SupportMeNepal](https://supportmenepal.com/), [e-Billing](https://www.e-billingnepal.com/school-management-software.html)). LOG's differentiator must be **offline-first learning + guidance**, not ERP breadth.

---

## 2. Nepal-Specific Requirements

### 2.1 Government systems: EMIS / IEMIS / Flash Reports

- Nepal's **Integrated Education Management Information System (IEMIS)** is the government's web-based system where every school is **mandated** to report student retention, attendance, and physical infrastructure data; it feeds the annual **Flash Reports** ([IEMIS portal](https://iemis.doe.gov.np/), [Flash I Report 2081](https://giwmscdnone.gov.np/media/pdf_upload/Flash%201%20Report%202081%20Final_rn76ynj.pdf)). Schools maintain their own Excel-based EMIS which they upload to web IEMIS ([Flash I Report 2080](https://giwmscdnone.gov.np/media/pdf_upload/media_file-17-428622471_nizhbxl.pdf)).
- The IEMIS strategic framework (2072 BS) is the MoEST planning document for this system ([MoEST eLibrary](http://elibrary.moest.gov.np/handle/123456789/76)). An independent verification survey found EMIS data tends to **overstate enrollment** vs. verified surveys, with ~60% of head teachers admitting data inflation — recommending third-party monitoring, capacity building, and better data-review processes ([DoE verification report](https://doe.gov.np/assets/uploads/files/b0b738c76e2204fede94c495edf25519.pdf), [academia.edu analysis](https://www.academia.edu/39890591/Education_management_information_system_EMIS_under_the_school_sector_development_plan_in_Nepal_an_analysis)).
- **Implication for LOG:** a school-level platform that *exports clean, accurate learner/teacher/attendance data* compatible with IEMIS fields is a genuine government-alignment feature (Flash reporting), and "data accuracy" is a documented pain point LOG can help with — never fabricate numbers (consistent with AGENTS.md "No Fabricated Fallbacks").
- UNESCO has been supporting CEHRD to integrate non-formal education data into IEMIS — the ecosystem is actively digitizing and local governments are the key users ([UNESCO](https://www.unesco.org/en/articles/integrating-non-formal-education-data-nepals-education-management-information-system), [British Council EMIS training](https://www.britishcouncil.org.np/training-education-management-information-system)).

### 2.2 NEB / SEE examination structure (grade 10, the national milestone)

- **SEE** (Secondary Education Examination, Grade 10) replaced SLC in 2017 and is administered by the **National Examination Board (NEB)** via the Office of the Controller of Examinations; ~550,000–600,000 students sit it annually; scheduled each March ([Wikipedia](https://en.wikipedia.org/wiki/Secondary_Education_Examination_(Nepal)), [Colleges Nepal](https://collegesinnepal.com/education-system/neb), [Nepalnews explainer](https://english.nepalnews.com/s/explainers/see-everything-you-need-to-know-about-nepals-national-secondary-examination/)).
- **Letter grading (4.0 scale, Letter Grading Directive 2078 / 2083):** A+ 90–100% (4.0), A 80–90 (3.6), B+ 70–80 (3.2), B 60–70 (2.8), C+ 50–60 (2.4), C 40–50 (2.0), D 35–40 (1.6), **NG (Not Graded) below 35%** ([Wikipedia](https://en.wikipedia.org/wiki/Secondary_Education_Examination_(Nepal)), [CollegeNP](https://www.collegenp.com/news/new-grading-system-in-see-examination-must-have-secured-35-marks), [EducateNepal 2083 guidelines](https://blog.educatenepal.com/2026/05/see-letter-grading-guidelines-system.html)).
- **Theory/internal split:** most compulsory subjects are **75:25** — 75 marks written theory (pass ≥ 27, i.e. 35%) and 25 marks internal/practical assessment (pass ≥ 10, i.e. 40%). **Both components must be passed separately** or the subject is NG; NG in any subject blocks the original certificate and forces a grade-improvement exam ([Nepalnews](https://english.nepalnews.com/s/explainers/see-everything-you-need-to-know-about-nepals-national-secondary-examination/), [Himalayan Times](https://thehimalayantimes.com/nepal/marked-improvement-in-see-results-this-year), [CollegeNP SEE 2082 notice](https://www.collegenp.com/news/see-2082-internal-assessment-marks-entry-notice)).
- **Internal assessment marks are entered/uploaded by schools into the OCE database** (for SEE 2082, deadline 2082/12/15 ≈ 2026-03-29) and **cannot be amended after upload** — a real, deadline-driven school workflow ([Edusanjal](https://edusanjal.com/news/notice-entry-of-internal-assessment-marks-for-secondary-education-examination-see/), [CollegeNP](https://www.collegenp.com/news/see-2082-internal-assessment-marks-entry-notice)).
- **Implication for LOG:** a gradebook that (a) models theory/internal split, (b) maps to NEB letter grades per subject, (c) produces marks ledgers for OCE upload and GPA report cards — is a concrete, high-value Nepal feature. Local competitors already advertise "built-in NEB grading system — print GPA gradesheets and marks ledgers with one click" ([e-Billing](https://www.e-billingnepal.com/school-management-software.html)).

### 2.3 Digital divide & school infrastructure reality (the hard constraint)

- **Only ~58.6% of schools have broadband internet** (Flash I 2081: 20.1% community / 50.0% private... 58.6% combined), ~80.7% have electricity; ICT equipment in 54.2% of schools ([Flash I 2081](https://giwmscdnone.gov.np/media/pdf_upload/Flash%201%20Report%202081%20Final_rn76ynj.pdf)).
- Earlier World Bank data: only 19% of ~28,800 public schools had internet (2021 EMIS), 42% had computers, and of those only 37% used them for teaching ([World Bank EdTech Readiness (ETRI)](https://thedocs.worldbank.org/en/doc/a8d5d09196b996cc3a802810b04f76c7-0140022023/related/231016-ETRI-Nepal-Marta-Conte-Domingue.pdf)). The SESP (2022–2031) targets expanding connectivity to 20,000 public schools ([World Bank ETRI](https://thedocs.worldbank.org/en/doc/a8d5d09196b996cc3a802810b04f76c7-0140022023/related/231016-ETRI-Nepal-Marta-Conte-Domingue.pdf)).
- Connectivity quality, not just presence, is the top challenge — especially rural; principals report unstable connections ([World Bank ETRI](https://thedocs.worldbank.org/en/doc/a8d5d09196b996cc3a802810b04f76c7-0140022023/related/231016-ETRI-Nepal-Marta-Conte-Domingue.pdf)).
- Digital divide correlates with socio-economic status (SES groups 3.92× more likely to have internet), gender, urban/rural ([NepJOL study](https://nepjol.info/index.php/tjec/article/view/70246)); rural case studies document poor connectivity, irregular electricity, low digital literacy, and workarounds like **offline downloading, device sharing, and blended learning** ([Baitadi case study](https://nepjol.info/index.php/sudurpaschim/article/view/90853)).
- Teachers' #1 reason for not integrating digital tools: **lack of ICT equipment (64.3%)**; digital readiness rated "not satisfactory" overall ([NCE SDG4 Spotlight report](https://educationoutloud.org/wp-content/uploads/2026/01/243-Spotlight-on-SDG4-Report_2025.pdf)).
- **Implication for LOG:** the offline-first architecture is not a nicety — it is the only viable model for most community schools. Device sharing is normal, so per-user isolation on shared devices is mandatory (§4.7).

### 2.4 Existing edtech in Nepal (what they cover — and LOG's gap)

| Platform | Type | Coverage | Notable strengths | Gaps LOG can exploit |
|---|---|---|---|---|
| **E-Paath** (OLE Nepal) | Free curriculum-aligned interactive modules, grades 1–8, Nepali/English (+ NSL sign language); offline servers for schools ([OLE Nepal](https://olenepal.org/digital-learning-solutions/e-paath/), [Google Play](https://play.google.com/store/apps/details?hl=en_US&id=org.olenepal.epaath)) | Content delivery | Deep curriculum alignment, offline infrastructure, E-Pustakalaya digital library | No per-learner tracking/teacher analytics; content only |
| **Sikai Chautari** (CEHRD, govt) | Official government learning portal; teacher PD courses + certificates ([CEHRD](https://learning.cehrd.gov.np/en/)) | Teacher training + content | Free, official, open-source | Not a school-management or classroom tool |
| **Thinko** | Integrated platform: video lessons, gamified practice, smart assessment, school/parent/LG dashboards; grades 4–10; **offline video mode** ([Thinko](https://thinko.com.np/)) | Full-stack learning + assessment | Closest commercial analogue to LOG's ambition; school partnerships, 18h support | Proprietary, cloud-dependency, no open offline sync story |
| **Sajilo Concept** | LMS video courses for SEE/NEB(+2)/BBS/MBS with quizzes ([Sajilo Concept](https://sajiloconcept.com/aboutus)) | Exam-prep content | Syllabus-based | Content-only, no school ops |
| **Sajilo Sikchya / Sikcha** | Early-learning app (2–6); SMART School Project LMS vendor (650+ schools, MoEST/CEHRD partnership, per-teacher-laptop program) ([Sajilo Sikshya](https://sajiloshikshya.com/), [Sajilo Sikcha](https://branch.sajilosikcha.com.np/index/about_us/1/1/1)) | ECE content + school LMS | Government partnership | Closed vendor; offline depth unclear |
| **Looma Education** | Offline-first hardware+software: solar-powered Looma Box, Looma Server, Looma Online; Nepali textbooks + Khan Academy etc.; teacher training; 81 village schools ([Looma](https://www.looma.education/)) | Offline content infrastructure | True no-power/no-internet operation; teacher training | Content distribution, not learner data/management |
| **School ERP vendors** (Vidyapith, YEDU, Project School, eAcademy, SupportMeNepal, EduSewa, e-Billing) | Fee management, GPS/RFID attendance, SMS to parents, exams, NEB grading, parent apps, eSewa/Khalti/ConnectIPS payments ([Vidyapith](https://vpit.com.np/), [YEDU](https://www.yedu.app/), [Project School](https://webbank.com.np/project-school/), [eAcademy](https://eacademynepal.com/), [SupportMeNepal](https://supportmenepal.com/), [EduSewa](https://edusewa.org/), [e-Billing](https://www.e-billingnepal.com/school-management-software.html)) | School administration | Nepal-specific billing/IRD compliance, parent apps | **Almost universally cloud-dependent, online-first** — the offline learning loop is missing |

**LOG's competitive position:** no Nepali player combines (a) curriculum-aligned learning content with per-learner progress, (b) genuine offline-first sync (IndexedDB + sneakernet), (c) classroom teacher workflows, and (d) NEB-aware gradebook in one platform. E-Paath proves the content model; Looma proves offline infrastructure; the ERP vendors prove the admin module demand — LOG sits at the intersection, which is currently empty.

### 2.5 Government vs private school needs

- **Government/community schools:** no/low fees, IEMIS reporting obligations, ICT grants uneven (only ~70% of public secondary schools received basic ICT grants), low digital literacy, teacher shortages ([World Bank ETRI](https://thedocs.worldbank.org/en/doc/a8d5d09196b996cc3a802810b04f76c7-0140022023/related/231016-ETRI-Nepal-Marta-Conte-Domingue.pdf)). Need: free/low-cost, offline-first, minimal training overhead, IEMIS-aligned exports, solar/LAN-friendly.
- **Private schools:** fee management with IRD-compliant billing, eSewa/Khalti/ConnectIPS payments, parent SMS/notifications, GPS attendance — the core pitch of local ERP vendors ([Vidyapith](https://vpit.com.np/), [e-Billing](https://www.e-billingnepal.com/school-management-software.html), [eAcademy](https://eacademynepal.com/)). Willing to pay (Project School packages NPR 45k–250k+/yr ([Project School](https://webbank.com.np/project-school/))).
- **Nepali language medium:** national curriculum is Nepali-medium with English as a subject; E-Paath and Thinko both deliver Nepali + English content ([OLE Nepal](https://olenepal.org/digital-learning-solutions/e-paath/), [Thinko](https://thinko.com.np/)); Vidyapith advertises multi-language (Nepali/English) UI ([Vidyapith](https://vpit.com.np/)). **Nepali UI localization is a baseline adoption requirement, not a differentiator.**

---

## 3. Offline-First / Low-Bandwidth Best Practices

Consensus across Android's "Build for Billions" guidelines, MDN, and field practitioners (Kalinko Labs, pupil.cloud, PANEOTECH, werun.dev):

### 3.1 Architecture principles

| Principle | Guidance | Source |
|---|---|---|
| **Offline is the default state, not an error state** | In emerging markets connectivity drops mid-session routinely; design for it as the baseline | [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/) |
| **Local store is source of truth; server is a sync target** | IndexedDB (web, via Dexie.js) / SQLite (mobile); OPFS for large binary files | [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/), [pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh) |
| **Queue-based deterministic sync** | Append-only local queue; idempotent ops with operation UUIDs; exponential backoff **with jitter**; batched syncs; prioritize critical ops (exam submissions, grades) over telemetry | [pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh), [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/) |
| **Conflict policy, explicitly** | Last-write-wins suffices for most business data; timestamp-merge with user notification for true conflicts; CRDTs (Yjs/Automerge) only for collaborative docs; server-side dedupe for append-only events | [pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh), [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/) |
| **Delta sync + compression** | Sync only changes since last sync (server timestamps/version vectors); gzip/brotli request bodies — compressing a 50KB payload to 8KB is the difference between success and timeout on 2G | [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/), [werun.dev](https://werun.dev/blog/progressive-web-apps-for-emerging-markets-the-business-case-for-building-lighter) |
| **Service-worker caching strategy per content type** | Cache-first (static assets: CSS/JS/fonts/images); stale-while-revalidate (dashboards, listings); network-first with fallback (transactions, auth); versioned immutable bundles | [MDN](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Offline_and_background_operation), [werun.dev](https://werun.dev/blog/progressive-web-apps-for-emerging-markets-the-business-case-for-building-lighter), [pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh) |
| **Background Sync API** | Retry queued submissions when connectivity returns (with local queue as fallback where unsupported); Periodic Background Sync for content refresh; Push for notifications | [MDN](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Offline_and_background_operation) |
| **Scope offline parity to mission-critical flows** | Not every feature needs offline parity — decide the top 5 flows (view lesson, answer quiz, submit homework, view grades, sync) and make those fully offline | [pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh) |
| **Optimistic UI + honest sync status** | Show action results immediately; subtle sync indicator; explicit offline/syncing/read-only states; "the contract you make with the user about sync status" | [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/), [PANEOTECH](https://paneo.tech/insights/offline-first-field-operations-pwa-twa-android), [pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh) |
| **Fail to useful** | When degrading, prefer read-only or queuing over catastrophic failure; preserve user agency and data integrity | [pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh) |
| **Offline-capable auth** | Long-lived tokens (24–72h) stored securely; no network round-trip per action; re-auth when connectivity returns | [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/) |

### 3.2 Low-end device / Android Go specifics

- Over half of users worldwide experience apps over 2G; optimize for low-speed connections: **WebP images, request images at target rendering size, never fetch an image twice, cache everything, queue writes, Room/SQLite for local DB, WorkManager for background cache updates** ([Android Build for Billions — Connectivity](https://developer.android.com/docs/quality-guidelines/build-for-billions/connectivity)).
- Adapt to network quality: **Data Saver detection, detect network type, scale request count/size with quality, prefetch on high-quality unmetered networks**; prioritize text before rich media ([Android Build for Billions](https://developer.android.com/docs/quality-guidelines/build-for-billions/connectivity)).
- **PWAs beat native apps** for this segment: no app-store friction, installable to home screen, ~fraction of storage (median device is a 16GB Android with ~4GB free, not an iPhone), instant repeat loads ([werun.dev](https://werun.dev/blog/progressive-web-apps-for-emerging-markets-the-business-case-for-building-lighter)).
- First-load budget guidance: **sub-100KB initial loads, code-splitting by route, Brotli over gzip (15–25% better), `font-display: swap`, WebP/AVIF (30–50% smaller), preconnect/DNS-prefetch, eliminate render-blocking** ([werun.dev](https://werun.dev/blog/progressive-web-apps-for-emerging-markets-the-business-case-for-building-lighter)).
- First-run offline: Trusted Web Activity wrappers need a custom offline launcher because the service worker isn't registered on first launch ([Chrome Dev](https://developer.chrome.com/docs/android/trusted-web-activity/offline-first)).
- Storage: design for **~200MB free**; browser storage quotas apply; don't drain data plans with polling — sync on meaningful events ([Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/)).
- Expected cost: offline-first adds **~15–25% development cost** but yields 2–3× engagement and 50–70% better multi-step completion rates in low-connectivity environments ([Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/)).

### 3.3 How LOG's current implementation maps

| Best practice | LOG today | Gap |
|---|---|---|
| Local store = source of truth | IndexedDB `api-cache` + `sync-queue` ([OFFLINE_SYNC.md](OFFLINE_SYNC.md)) | ✓ core pattern exists |
| Queue + optimistic 202 + flush on `online` | ✓ `fetchWithCache` (`frontend/src/lib/api.ts`) | ✓ |
| Idempotency | ✓ completion + SyncBulk idempotent ([ENHANCEMENT.md](ENHANCEMENT.md) §0) | ✓ |
| Exponential backoff | ✓ | ✓ |
| 401-preserving queue + token replay | ✓ | ✓ |
| Honest offline UI states | ✓ offline banner, queued-state UI | partial (per-page polish) |
| **Sneakernet** (`.logsync` export/import) | ✓ unique capability ([syncExport.ts](OFFLINE_SYNC.md)) | extend to more record types |
| Delta sync / batched payload compression | ✗ SyncBulk is full-queue | add delta + gzip |
| Content versioning for bundles | ✗ | add versioned course packs |
| Network-adaptive loading (Network Information API) | ✗ | P2 |
| Background Sync API (native browser API) | ✗ custom queue used (fine) | optional |
| Monitoring: queue depth, sync latency, collision counts | ✗ | P1 ops |
| Offline auth | ✓ 72h tokens | ✓ |

---

## 4. Production-Grade Checklist

### 4.1 Child data protection (minors)

- **COPPA (US, applies globally to child-directed services):** verifiable parental consent before collecting personal info from children under 13; notice to parents; rights to review/delete; **retention only as long as reasonably necessary (16 CFR §312.10)**; persistent identifiers count as personal information; 2025 amendments broaden "personal information" to biometric/government identifiers; schools can sometimes consent for school-authorized educational purposes ([FTC COPPA rule](https://www.ftc.gov/legal-library/browse/rules/childrens-online-privacy-protection-rule-coppa), [2025 amendments](https://www.govinfo.gov/content/pkg/FR-2025-04-22/pdf/2025-05904.pdf), [cookie-script explainer](https://cookie-script.com/guides/edtech-coppa-gdprk-ferpa)).
- **GDPR-K (children under GDPR Art. 8):** consent from parental responsibility holder when the service is offered directly to children; consent must be specific, informed, unambiguous ([cookie-script](https://cookie-script.com/guides/edtech-coppa-gdprk-ferpa)).
- **UK ICO Children's Code for edtech:** edtech providers are in scope when they process children's data beyond the school's instructions (e.g., own commercial purposes, product development); if the platform is purely a digital extension of school functions, the school decides — a useful framing for LOG's school-first model ([ICO](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/childrens-information/childrens-code-guidance-and-resources/the-children-s-code-and-education-technologies-edtech/)).
- **FERPA (education records):** school-official exception lets vendors process PII only for the school's authorized purposes; no re-disclosure; no use for marketing/ads; direct control by the school; annual notice; parents' access rights ([FERPA](https://studentprivacy.ed.gov/ferpa), [PTAC training](https://studentprivacy.ed.gov/sites/default/files/training/supporting_materials/Protecting_Student_Privacy_While_Using_Online_Educational_Services_0.pdf), [PTAC vendor FAQ](https://studentprivacy.ed.gov/sites/default/files/resource_document/file/Vendor%20FAQ.pdf)).
- **Practical checklist for LOG:** no third-party analytics/advertising/trackers on learner surfaces; explicit consent capture at enrollment (school signs as data controller); data inventory + sensitivity classification; retention/deletion policy with automated destruction; parental access workflow; contracts (terms) positioning LOG as processor acting for the school ([PTAC checklists](https://studentprivacy.ed.gov/sites/default/files/resource_document/file/Policies%20for%20Users%20of%20Student%20Data%20Checklist.pdf), [PTAC data governance](https://studentprivacy.ed.gov/sites/default/files/resource_document/file/Data%20Governance%20Checklist_0.pdf)).

### 4.2 Local data residency & Nepal law

- Nepal has **no unified data protection law**; the operative stack: **Constitution Art. 28** (privacy as fundamental right), **Privacy Act 2075 (2018)** (consent required; use limited to stated purpose; protection duties; no dedicated regulator — enforcement via District Court; penalties up to 3 yrs imprisonment / NPR 30,000), **Privacy Regulation 2077**, **Data Act 2079 (2022)** (establishes National Data Office; consent; accuracy; **data portability & deletion rights**; **cross-border transfers only to countries with "adequate protection"**; breach notification to government and affected individuals; 30-day access responses), **Data Regulation 2080**, plus Electronic Transactions Act 2063 and Muluki Criminal Code ([DataGuidance](https://www.dataguidance.com/notes/nepal-privacy-overview), [Privacy Act 2075](https://lawcommission.gov.np/en/wp-content/uploads/2019/07/The-Privacy-Act-2075-2018.pdf), [Sherpa Law](https://www.sherpalawassociates.com/resources/articles/cmkqogl24000iktrvmub1a78g), [Corporate Biz Legal](https://corporatebizlegal.com/insight/data-protection-law-in-nepal/), [DLA Piper](https://www.dlapiperdataprotection.com/index.html?c=NP&t=law)).
- **No current mandate to store personal data on servers inside Nepal** (only OTT providers are mandated; no local-processing mandates exist today) ([UNECA DTRI country profile](https://dtri.uneca.org/v1/uploads/country-profile/npl-country-profile-en.pdf)), but the Data Act's "adequate protection" test for transfers + the **Data Center & Cloud Service Directives** (data centers must be registered with the Department of IT; government/private entities encouraged to use domestically listed providers) make **in-Nepal hosting the safe long-term posture** ([DLA Piper transfer](https://www.dlapiperdataprotection.com/index.html?c=NP&t=transfer)).
- **Practical checklist for LOG:** hosting in Nepal (or documented adequacy + contractual safeguards); consent notices in Nepali; breach-notification runbook (NDO + affected parents/students); deletion/portability endpoints; a named data-protection point of contact; school (not LOG) is the controller for learner records.

### 4.3 Role-based access

- Role-based permissions by job function, explicitly prohibiting "data browsing"; documented acknowledgement for access to student PII ([PTAC policies checklist](https://studentprivacy.ed.gov/sites/default/files/resource_document/file/Policies%20for%20Users%20of%20Student%20Data%20Checklist.pdf), [Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)).
- LOG already has 3-tier RBAC with server-side role revalidation, edge role checks, and privilege-escalation rules ([SECURITY_AND_RBAC.md](SECURITY_AND_RBAC.md)). Gaps: a **PARENT role** (view own children only) for the parent portal, and class-scoped moderator data boundaries (a teacher sees only their classes).

### 4.4 Audit logging

- Maintain records of **requests for access and disclosures** of PII; parties, interests, timestamps; parents may review ([FERPA §99.32](https://studentprivacy.ed.gov/ferpa)). Monitor compliance via access-log audits ([PTAC](https://studentprivacy.ed.gov/sites/default/files/resource_document/file/Policies%20for%20Users%20of%20Student%20Data%20Checklist.pdf)).
- LOG already has per-request structured audit logging (request ID, user ID, IP, duration) ([ENHANCEMENT.md](ENHANCEMENT.md) §0). Gaps: **immutable, append-only event log for sensitive operations** (role changes, grade edits, data exports, bulk deletes) with "who/what/when/from-where"; periodic review habit; log retention policy.

### 4.5 Backup / DR for school data

- **Backups must be separated from the operational network**, tested with **recurring real restore drills** (most districts never test restores; backups often fail in practice during ransomware), and ideally **offline/immutable** ([CISA K-12 Digital Infrastructure Brief](https://files.eric.ed.gov/fulltext/ED650652.pdf), [EdTech Magazine](https://edtechmagazine.com/k12/article/2023/09/how-schools-can-securely-back-and-recover-private-data-now)).
- **Encrypt data in transit and at rest** (FERPA expectations); treat backup media as carefully as working data; air-gap sensitive backups; document recovery procedure, encryption keys, and who is authorized ([EdTech Magazine](https://edtechmagazine.com/k12/article/2023/09/how-schools-can-securely-back-and-recover-private-data-now), [EdTech Magazine 5 questions](https://edtechmagazine.com/k12/article/2024/02/five-backup-and-recovery-questions-ask-data-goes-missing-k-12)).
- **RPO/RTO discipline:** define per-function Recovery Point/Time Objectives; nightly backups of student records to offsite storage; annual plan testing with board review ([NOLA policy example](https://nolapublicschools.com/CAPS/Policies/EFD_-_Business_Continuity_and_Technology_Disaster_Recovery_(Amended_12_17_20).htm)).
- **LOG-specific:** SQLite single-file backups are trivial to snapshot; add automated encrypted nightly backup + `restore` command + a **school-level "download my data" full export** (`.logsync` is the seed of this). Document RTO (e.g., <1 day) and test restores in CI.

### 4.6 Teacher onboarding (low digital literacy context)

- Nepal's teachers cite lack of equipment (64.3%) and training as the top barriers; digital readiness is "not satisfactory"; EMIS studies recommend **capacity building** as a first-class activity ([NCE SDG4 report](https://educationoutloud.org/wp-content/uploads/2026/01/243-Spotlight-on-SDG4-Report_2025.pdf), [IEMIS verification](https://www.academia.edu/39890591/Education_management_information_system_EMIS_under_the_school_sector_development_plan_in_Nepal_an_analysis)).
- Looma and Thinko both treat **on-site teacher training** as core to their model ([Looma](https://www.looma.education/), [Thinko](https://thinko.com.np/)). Sikai Chautari provides government PD courses ([CEHRD](https://learning.cehrd.gov.np/en/)).
- **Practical checklist for LOG:** Nepali-language onboarding wizard; printable step-by-step guides; in-app "demo data" (sandbox) mode for practice; session-based training kit for school visits; simple role-scoped home screens (teachers see only their class tools); offline-tolerant flows so a bad connection never blocks a first lesson.

### 4.7 Device-sharing safety (Nepali reality: shared phones/browsers)

- Device sharing is a documented normal practice in rural Nepal ([Baitadi study](https://nepjol.info/index.php/sudurpaschim/article/view/90853)); school labs and household phones are shared.
- Risks: cross-account data leakage via service-worker caches (LOG fixed the SW cross-account cache leak — [ENHANCEMENT.md](ENHANCEMENT.md) §1.6), browser autofill/session persistence, and orphans of offline queues on shared devices.
- **Practical checklist for LOG:** logout-all-devices (revoke all `jti` for a user — the infrastructure exists); explicit per-user cache partitioning in IndexedDB keyed by user ID; "end session" cleaning local stores; device-level "kid mode" that prevents clearing data; post-login/identity confirmation before showing sensitive data; keep 72h tokens short enough to limit exposure on shared devices.

---

## 5. Prioritized Feature Roadmap for LOG

Priorities: **P0** = foundation/production-readiness or core learner value; **P1** = high-value expansion; **P2** = strategic/nice-to-have. Effort: S (≤1 wk), M (2–4 wks), L (1–3 months). "Builds on" references existing LOG components ([ARCHITECTURE.md](ARCHITECTURE.md), [OFFLINE_SYNC.md](OFFLINE_SYNC.md), [ENHANCEMENT.md](ENHANCEMENT.md)).

### 5.1 P0 — Production readiness & core learner value

| # | Feature | Effort | Builds on | Rationale / research basis |
|---|---|---|---|---|
| P0-1 | **Real quiz correctness capture** — learner submits per-question answers/accuracy; guidance & analytics derive from actual attempts, not `+2.5` heuristics | M | `LearnerActivity`, SyncBulk, guidance engine | "No Hallucinations" principle; ENHANCEMENT roadmap item 2; guidance quality is LOG's differentiator |
| P0-2 | **Nepali localization (i18n)** — Nepali/English UI for all student-facing screens; Devanagari rendering | M | all frontend pages | Nepali-medium is the national norm ([OLE Nepal](https://olenepal.org/digital-learning-solutions/e-paath/)); local competitors ship bilingual ([Vidyapith](https://vpit.com.np/)); ENHANCEMENT item 3 |
| P0-3 | **SMS gateway for OTP** — connect a Nepali SMS provider; keep demo-log fallback | S | OTP flow, rate limiter, `DeleteOTP` | Field login is impossible today (ENHANCEMENT item 6); SMS is the only practical auth channel for schools without email ([Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/) — SMS as designed-in channel, not afterthought) |
| P0-4 | **Audit log for sensitive ops** — immutable append-only log: role changes, grade edits, data exports, deletes; review UI for admins | M | existing structured logging ([SECURITY_AND_RBAC.md](SECURITY_AND_RBAC.md) §8) | FERPA disclosure-records requirement ([FERPA §99.32](https://studentprivacy.ed.gov/ferpa)); PTAC access-log audits |
| P0-5 | **Backup & restore + full data export** — automated encrypted SQLite backup, restore command, school-level "download my data" (extends `.logsync`) | S–M | sneakernet sync (`syncExport.ts`), Go backend | CISA: backups separated + tested restores ([CISA brief](https://files.eric.ed.gov/fulltext/ED650652.pdf)); RPO/RTO discipline ([NOLA policy](https://nolapublicschools.com/CAPS/Policies/EFD_-_Business_Continuity_and_Technology_Disaster_Recovery_(Amended_12_17_20).htm)); Nepal Data Act deletion/portability rights ([Corporate Biz Legal](https://corporatebizlegal.com/insight/data-protection-law-in-nepal/)) |
| P0-6 | **Child-data privacy posture** — consent capture at enrollment, no third-party trackers, retention policy, parent data-access request flow, processor contract language | M | auth/RBAC, user model | COPPA retention rule ([FTC](https://www.ftc.gov/legal-library/browse/rules/childrens-online-privacy-protection-rule-coppa)), ICO edtech framing ([ICO](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/childrens-information/childrens-code-guidance-and-resources/the-children-s-code-and-education-technologies-edtech/)), PTAC checklists |
| P0-7 | **Device-sharing safety** — logout-all-devices (revoke all jti), user-scoped IndexedDB stores, shared-device session hygiene | S–M | JWT `jti` blocklist, IndexedDB layer | Device sharing is normal in Nepal ([Baitadi study](https://nepjol.info/index.php/sudurpaschim/article/view/90853)); ENHANCEMENT item 7 |
| P0-8 | **Class & section model** — real class entities with teacher assignment; replaces roster derivation from first course | M | users, moderator roster | SIS core ("class history" is the spine — [EdFleet](https://edfleet.com/blog/school-management-system-features/)); ENHANCEMENT 2.5 |

### 5.2 P1 — High-value expansion (school operations)

| # | Feature | Effort | Builds on | Rationale / research basis |
|---|---|---|---|---|
| P1-1 | **Teacher workflow: assignments & submissions** — teacher creates assignments for a class; learners submit offline (queued); teacher grades; per-student guidance replies | M–L | sync-queue, `LearnerActivity`, moderator API | The missing student↔teacher loop (ENHANCEMENT item 4); gradebook is a core SIS module ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/)) |
| P1-2 | **NEB letter-grade gradebook** — theory/practical split (75:25), per-subject pass rules (27/75, 10/25), NG logic, GPA 4.0 mapping, printable marks ledger & report card | M | `LearnerActivity` scores, P1-1 | SEE structure is precise and mandatory ([Wikipedia](https://en.wikipedia.org/wiki/Secondary_Education_Examination_(Nepal)), [CollegeNP](https://www.collegenp.com/news/see-2082-internal-assessment-marks-entry-notice)); competitors already ship "NEB marks ledger" ([e-Billing](https://www.e-billingnepal.com/school-management-software.html)) |
| P1-3 | **SEE internal-assessment workflow** — school enters 25-mark internal assessment per subject per student; export ready for OCE database upload; immutable-after-submit guard | M | P1-2 | OCE marks entry is a real deadline-driven school workflow, marks cannot be amended ([Edusanjal](https://edusanjal.com/news/notice-entry-of-internal-assessment-marks-for-secondary-education-examination-see/)) |
| P1-4 | **Attendance (offline-first)** — teacher marks daily/period attendance on the queue; parent SMS alerts; attendance reports | M | sync-queue, DailyActivity pattern, SMS (P0-3) | Core module in every SIS ([openSIS](https://opensis.com/features)); offline data entry is the killer offline use case |
| P1-5 | **Announcements & notifications** — admin/teacher broadcasts; offline-queued posts; SMS fallback; optional push | S–M | admin API, SMS (P0-3), sync-queue | Communication is a core module ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/)); Nepal parent-SMS expectation ([Vidyapith](https://vpit.com.np/)) |
| P1-6 | **Principal/HOD analytics** — per-class drill-down from existing admin dashboard; cohort trends (attendance, mastery, guidance uptake); CSV export | M | admin dashboard, `DailyActivity`, roster | Reporting is the leadership value ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/), [Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)) |
| P1-7 | **Offline content bundles** — versioned downloadable course/activity packs (immutable artifacts) for pre-caching on school Wi-Fi/lab PCs | M | service-worker cache, IndexedDB, sneakernet | Versioned immutable bundles are the offline-content standard ([pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh)); Looma/E-Paath prove school-lab content distribution ([Looma](https://www.looma.education/)) |
| P1-8 | **Teacher onboarding kit** — Nepali wizard, printable guides, sandbox/demo mode, session training material | S | frontend, docs | 64.3% of teachers cite equipment; training is the documented blocker ([NCE SDG4](https://educationoutloud.org/wp-content/uploads/2026/01/243-Spotlight-on-SDG4-Report_2025.pdf)); Looma/Thinko train teachers as core model |
| P1-9 | **Parent/guardian portal (v1)** — new PARENT role; view own children's progress, attendance, announcements, guidance | L | RBAC, roster, P1-4, P1-5 | Parent portals are baseline in every vendor ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/), [SupportMeNepal](https://supportmenepal.com/)) |
| P1-10 | **Sync telemetry & ops monitoring** — queue depth, sync success rate, collision counts, offline error rates; admin health page | S | backend observability | You can't improve offline reliability you can't measure ([pupil.cloud](https://pupil.cloud/designing-offline-first-edtech-how-to-keep-learning-going-wh)) |

### 5.2 P2 — Strategic / nice-to-have

| # | Feature | Effort | Builds on | Rationale / research basis |
|---|---|---|---|---|
| P2-1 | **Fee management (lite)** — fee structure, dues, receipts; eSewa/Khalti/ConnectIPS links; IRD-compliant billing | L | admin, student records | The #1 commercial pull in Nepal ERP ([Vidyapith](https://vpit.com.np/), [e-Billing](https://www.e-billingnepal.com/school-management-software.html)); big scope — delay until P1 revenue case exists |
| P2-2 | **Admissions/enrollment** — application → merit list → enrollment → class assignment | L | student records, P0-8 | Core SIS but out of learning-platform core ([openSIS](https://opensis.com/features)) |
| P2-3 | **Timetable generator** | L | P0-8 | "Weeks → hours" selling point ([OpenEduCat](https://openeducat.org/articles/school-management-software-buyers-guide/)) |
| P2-4 | **IEMIS/Flash export** — export learner/teacher/attendance data aligned to IEMIS fields | M | P1-6 | Schools are mandated to report into IEMIS; accuracy is a documented pain ([IEMIS](https://iemis.doe.gov.np/), [Flash 2081](https://giwmscdnone.gov.np/media/pdf_upload/Flash%201%20Report%202081%20Final_rn76ynj.pdf), [verification report](https://doe.gov.np/assets/uploads/files/b0b738c76e2204fede94c495edf25519.pdf)) |
| P2-5 | **Library & digital resources** — catalog + link to E-Pustakalaya/E-Paath content | M | content model | Library module standard ([OpenEduCat](https://openeducat.org/open-source-education-erp/)); OLE Nepal precedent ([OLE Nepal](https://olenepal.org/digital-learning-solutions/e-paath/)) |
| P2-6 | **Network-adaptive mode** — Network Information API; data-saver mode; compressed images; text-first on 2G | S–M | frontend | Android Data Saver + adaptive loading ([Android Build for Billions](https://developer.android.com/docs/quality-guidelines/build-for-billions/connectivity), [Kalinko Labs](https://kalinkolabs.com/blog/offline-first-applications-african-markets/)) |
| P2-7 | **Academic calendar** — terms, holidays, SEE/exam dates, reminders | S | announcements (P1-5) | Standard module ([Synergy SIS](https://www.edupoint.com/Synergy-Education-Platform/Student-Information-Suite/Synergy-SIS)) |
| P2-8 | **School LAN sync hub** — school-owned offline server that caches bundles and receives queues (Looma-style appliance or home-router class device) | L | P1-7, sneakernet | Looma proves no-internet school infrastructure ([Looma](https://www.looma.education/)); OLE Nepal offline servers ([OLE Nepal](https://olenepal.org/digital-learning-solutions/e-paath/)) |
| P2-9 | **Multi-school / multi-branch** — one deployment serving several schools with scoped data | L | RBAC | EduSewa is built multi-school ([EduSewa](https://edusewa.org/)); matches local-government (palika) deployment model |
| P2-10 | **Reconnect digest** — "3 new guidance notes since your last visit" low-bandwidth check-in | S | offline cache, guidance | ENHANCEMENT item 5; MDN periodic-sync pattern ([MDN](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Offline_and_background_operation)) |

### 5.3 Suggested sequencing

```
Phase A (P0):  P0-1 → P0-3 → P0-4 → P0-5 → P0-6 → P0-7   (data truth + production posture)
Phase B (P0):  P0-2 → P0-8                                 (i18n + class model unlock)
Phase C (P1):  P1-1 → P1-2 → P1-3 → P1-4 → P1-5            (teacher classroom workflows)
Phase D (P1):  P1-6 → P1-7 → P1-8 → P1-9 → P1-10           (analytics, content, parents, ops)
Phase E (P2):  P2-1..P2-10 as market feedback dictates
```

---

## 6. Top 10 Highest-Value Feature Recommendations

1. **Real quiz correctness capture (P0-1, M)** — the single change that makes LOG's guidance trustworthy and data honest; unlocks every analytics feature downstream.
2. **Nepali localization (P0-2, M)** — baseline adoption requirement for Nepali-medium schools; every local competitor ships bilingual.
3. **SMS gateway for OTP + notifications (P0-3, P1-5, S/M)** — unblocks field login today and is the parent-communication channel every Nepali school expects.
4. **Class & section model + real teacher workflows (P0-8, P1-1, M–L)** — turns the moderator roster into a working classroom tool: assignments, submissions, per-student guidance replies.
5. **NEB letter-grade gradebook + SEE internal-assessment workflow (P1-2, P1-3, M)** — precise national exam rules (75:25, NG at <35%, OCE marks entry deadline) make this a concrete, defensible Nepal feature competitors only half-cover.
6. **Offline-first attendance (P1-4, M)** — highest-frequency teacher workflow and the perfect fit for the existing sync-queue; parent SMS alerts on top.
7. **Backup/restore + full data export (P0-5, S–M)** — production trust: tested restores, school-owned data, and Data Act 2079 portability/deletion compliance in one feature.
8. **Audit log for sensitive operations (P0-4, M)** — regulatory-grade accountability for grades, roles, and exports; builds on existing structured logging.
9. **Child-data privacy posture (P0-6, M)** — consent, retention, no trackers, parent access; the difference between a demo and a school board approving the platform.
10. **Principal analytics + IEMIS-ready exports (P1-6, P2-4, M)** — turns learner data into leadership decisions and aligns with the government's mandated reporting (Flash/IEMIS), where data accuracy is a documented pain point.

**Strategic note:** LOG should not chase full ERP breadth (fees/payroll/transport are P2 at best). Its defensible niche — currently empty in Nepal — is the intersection of **offline-first per-learner learning**, **NEB-aware classroom workflows**, and **honest, exportable school data** (E-Paath = content, Looma = offline infra, local ERP vendors = admin; none do all three with LOG's offline sync depth).