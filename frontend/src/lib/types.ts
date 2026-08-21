export interface Learner {
  id: string;
  name: string;
  email: string;
  created_at: string;
}

export interface Activity {
  id: string;
  title: string;
  description: string;
  status: string;
  topic: string;
  order: number;
  // WP-3.1 (RC-07): OER metadata — every activity carries an honest license
  // + attribution; empty means none exists (never rendered as fabricated).
  license?: string;
  license_url?: string;
  attribution?: string;
  source_url?: string;
  // WP-3.4 (RC-12): NSL caption text (empty = no caption track).
  caption_text?: string;
}

export interface Progress {
  learner_id: string;
  total_topics: number;
  completed: number;
  current_streak: number;
  overall_score: number;
}

export interface Observation {
  id: string;
  learner_id: string;
  category: string;
  text: string;
  created_at: string;
}

export interface Guidance {
  id: string;
  learner_id: string;
  text: string;
  action: string;
  type: string;
  created_at: string;
}

export interface DashboardData {
  learner: Learner;
  progress: Progress;
  activities: Activity[];
  observations: Observation[];
  guidance: Guidance[];
}

export interface LearningJourneyData {
  activities: Activity[];
}

export interface ChartDataPoint {
  name: string;
  score: number;
  duration: number;
  attempts?: number;
  accuracy?: number;
}

export interface SystemAnalytics {
  total_users: number;
  active_daily: number;
  total_completions: number;
}

// WP-3.3 (RC-10): honest pilot measurement — every number is derived from
// real stored scan rows; zeros are real zeros, never invented.
export interface PilotStats {
  total_scans: number;
  scans_today: number;
  starts: number;
  distinct_posters: number;
  start_rate: number;
  per_poster: { poster_id: string; scans: number; starts: number }[];
}

export interface AdminData {
  analytics: SystemAnalytics;
  recent_users: Learner[];
}

export interface SchoolClass {
  id: string;
  name: string;
  grade: string;
  section: string;
  teacher_id?: string;
  created_at: string;
  member_count?: number;
  invite_code?: string;
}

export interface Announcement {
  id: string;
  title: string;
  body: string;
  author_id: string;
  created_at: string;
}

export interface Assignment {
  id: string;
  class_id: string;
  title: string;
  description: string;
  activity_id?: string;
  due_date?: string;
  created_by?: string;
  created_at: string;
  submissions?: number;
}

export interface Submission {
  id: string;
  assignment_id: string;
  learner_id: string;
  note: string;
  submitted_at: string;
}

export interface AuditLogEntry {
  id: number;
  user_id: string;
  action: string;
  detail: string;
  ip: string;
  created_at: string;
}

// WP-2.1 parent portal: minimal identity — id + name + digest opt-in only.
// No email, phone or contact fields (privacy boundary, mirror of ParentChild).
export interface ParentChild {
  id: string;
  name: string;
  digest_opt_in: boolean;
}

export interface ParentActivityDigest {
  id: string;
  title: string;
  topic: string;
  status: string;
}

export interface ParentGuidanceDigest {
  id: string;
  learner_id: string;
  text: string;
  action: string;
  type: string;
  created_at: string;
}

// Read-only, sanitized progress digest for one linked learner.
export interface ChildDigest {
  learner: { id: string; name: string };
  progress: Progress;
  activities: ParentActivityDigest[];
  guidance: ParentGuidanceDigest[];
  as_of: string;
}

// WP-2.2 support funnel issue.
export interface SupportIssue {
  id: string;
  user_id: string;
  category: string;
  description: string;
  escalated: boolean;
  status: string;
  resolver_id?: string;
  resolution_note?: string;
  created_at: string;
  resolved_at?: string | null;
}

// WP-2.3 honest gradebook: one real data point (canonical status + REAL
// stored accuracy/attempts; attempts === 0 means "not yet assessed").
export interface GradebookRow {
  activity_id: string;
  title: string;
  topic: string;
  status: string;
  accuracy: number;
  attempts: number;
}

export interface GradebookStudent {
  student_id: string;
  name: string;
  rows: GradebookRow[];
}

export interface LearnerNote {
  id: string;
  student_id: string;
  teacher_id: string;
  note: string;
  created_at: string;
  updated_at: string;
}
