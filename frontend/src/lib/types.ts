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
