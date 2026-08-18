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
}

export interface SystemAnalytics {
  total_users: number;
  active_daily: number;
  total_completions: number;
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
