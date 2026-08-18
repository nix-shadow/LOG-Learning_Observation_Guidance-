# Frontend Architecture & UI/UX Guide

## 1. Overview
The LOG frontend is built with **Next.js 14 (App Router)**, React 18, and TypeScript. It utilizes Tailwind CSS for a custom utility-based design system, Framer Motion for animations, Recharts for analytics, and Lucide Icons for navigation and UI elements.

---

## 2. Directory Layout & Page Structure

```
frontend/src/
├── app/
│   ├── layout.tsx             # Root layout with AuthProvider & Navigation
│   ├── page.tsx               # Public landing page with LOG cycle
│   ├── login/page.tsx         # Email/password login, registration & Google sign-in
│   ├── forgot-password/       # Password recovery request form
│   ├── settings/              # Theme toggle, offline-mode simulation, password change
│   ├── dashboard/page.tsx     # Student home with streaks and goal meters
│   ├── learning/
│   │   ├── page.tsx           # Sequenced timeline of curriculum modules
│   │   └── [id]/page.tsx      # Interactive multi-step quiz player
│   ├── courses/page.tsx       # Course catalog with multi-category filters
│   ├── observation/page.tsx   # Visualized habit and performance analytics
│   ├── guidance/page.tsx      # Actionable recommendations & direct links
│   ├── moderator/page.tsx     # Teacher classroom and student roster portal
│   └── admin/page.tsx         # Principal control center & role manager
├── components/
│   ├── Navigation.tsx         # Sticky desktop nav and mobile bottom bar
│   ├── PageTransition.tsx     # Framer Motion page transition wrapper
│   ├── OfflineBanner.tsx      # Amber banner + floating badge when offline
│   ├── InstallPrompt.tsx      # PWA beforeinstallprompt with 7-day dismissal
│   ├── SkeletonLoader.tsx     # Animated loading placeholders (card/stats/text/chart)
│   ├── MicroModuleViewer.tsx  # Swipeable bite-sized module player
│   ├── SyncIsland.tsx         # Floating sync/offline indicator driven by the real queue
│   ├── CommandPalette.tsx     # ⌘K quick-navigation palette
│   ├── ThemeProvider.tsx / ThemeToggle.tsx  # Dark/light theming
│   └── ThreeBackground.tsx    # Three.js particle backdrop
├── context/
│   └── AuthContext.tsx        # Global auth state, session token, & login/logout
├── hooks/
│   └── useSyncQueue.ts        # Polls pending offline sync count (5s)
└── lib/
    ├── api.ts                 # Offline interceptor & sync queue manager
    ├── adaptiveEngine.ts      # Offline rule-based guidance generator
    ├── syncExport.ts          # .logsync sneakernet export/import
    └── types.ts               # Shared TypeScript interfaces
```

---

## 3. Design System & Brand Palette

The user interface follows a tailored color palette defined in `tailwind.config.ts`:

| Token | Hex Value | Semantic Usage |
| :--- | :--- | :--- |
| `brand-blue` | `#0A2540` | Primary brand identity, dark headers, cards, and primary buttons |
| `brand-teal` | `#00B4D8` | Accent color, active indicators, completed milestones, progress bars |
| `brand-amber`| `#FFB703` | Attention flags, in-progress badges, streaks, and warning highlights |
| `brand-gray` | `#E9ECEF` | Subtle borders, light background fills, and neutral dividers |
| `brand-white`| `#F8F9FA` | Soft page surface |
| `brand-text` | `#212529` | High-contrast body typography for optimal readability |

---

## 4. Page Walkthrough & Feature Breakdown

### 4.1 Home Page (`src/app/page.tsx`)
- Hero section highlighting the low-connectivity value proposition.
- 5-stage LOG cycle interactive cards (Learn, Observe, Understand, Guide, Improve).
- Direct call-to-action to begin learning.

### 4.2 Student Dashboard (`src/app/dashboard/page.tsx`)
- Gradient header with active streak counter and Logic Master badge.
- SVG circular daily goal progress indicator (`react-circular-progressbar`).
- Current Focus card grid rendered dynamically from the `guidance` payload.
- Recent Observations sidebar displaying category pills and diagnostic notes.
- **Offline Sync card:** export/import `.logsync` sneakernet files, plus a full logout button (JWT revocation + cache purge).

### 4.3 Learning Journey Timeline (`src/app/learning/page.tsx`)
- Connected vertical timeline connecting modular curriculum topics.
- Dynamic status icons (`CheckCircle2` for Completed, `PlayCircle` for In Progress, `Circle` for Locked).
- Action buttons ("Continue"/"Review") are live links into the module player at `/learning/[id]`.

### 4.4 Interactive Quiz Engine (`src/app/learning/[id]/page.tsx`)
- Fetches real micro-module content from `GET /api/v1/activities/:id/modules` (cached for offline replay) and renders it through `MicroModuleViewer`.
- Falls back to the built-in multi-step demo lesson when no modules exist (e.g. catalog previews) or when fully offline with no cached modules.
- Supports step types:
  - `concept`: Explanatory theoretical slides with contextual examples.
  - `interactive`: Knowledge checks with multiple choice options, immediate green/red feedback, and explanatory lightbulb notes.
  - `completion`: Celebratory conclusion with completion toast and automated journey update.
- Completing the last module posts to `/api/v1/activities/:id/complete` (queued offline when disconnected).

### 4.5 Course Catalog (`src/app/courses/page.tsx`)
- Live catalog rendered from `GET /api/v1/courses?page=1&limit=100` (paginated, cached for offline browsing).
- Category pill filters derived dynamically from the loaded data (Computer Science, Frontend, Backend, Design...).
- Full-text search, difficulty badges, star ratings, enrolled counts, and duration estimates per card.
- Shows an "offline — last synced catalog" notice when the network fetch falls back to cache.

### 4.6 Observation Analytics (`src/app/observation/page.tsx`)
- 3 KPI metric cards: Topics Mastered, Current Streak, and Overall Score.
- Area Chart: Weekly performance trajectory with custom SVG gradients.
- Bar Chart: Daily engagement duration metrics.
- Categorized insight tiles with custom icon mappings (Star for Strengths, Target for Areas of Growth, LineChart for Consistency).

### 4.7 Guidance Center (`src/app/guidance/page.tsx`)
- Grouped recommendations categorized by `next_step`, `practice`, and `insight`.
- Interactive action buttons navigating the learner directly to the relevant curriculum module.

### 4.8 Moderator / Teacher Portal (`src/app/moderator/page.tsx`)
- Classroom telemetry summary: Active Students, Needs Attention count (computed server-side from zero-streak learners), and Assignments Due.
- **Pre-fetch for Offline** button caches the roster into IndexedDB so the classroom table works fully disconnected.
- Class Roster table for "Logic 101" displaying student completion percentage bars, current streaks, and direct action triggers (pulled from `GET /api/v1/moderator/roster`).

### 4.9 Admin Control Center (`src/app/admin/page.tsx`)
- Global system analytics cards: Total Users, Active Daily Users (updated within 24h), Total Completions — all computed from live database rows.
- Registered User Table with role pills (`ADMIN` in red, `STUDENT` in blue).
- Quick Action triggers for creating activities and broadcasting notices.
