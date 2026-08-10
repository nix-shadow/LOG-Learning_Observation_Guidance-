import { initDB, CACHE_STORE } from './api';

export interface LocalGuidance {
  id: string;
  learner_id: string;
  text: string;
  action: string;
  type: string;
  created_at: string;
}

/**
 * Evaluates the student's recent activity history to generate adaptive 
 * guidance messages offline without needing to contact the server.
 */
export async function evaluateLocalAdaptivity(learnerId: string) {
  try {
    const db = await initDB();
    
    // In a full implementation, we would pull quiz scores from the local cache.
    // For this prototype, we simulate checking local progress.
    const cachedData = await db.get(CACHE_STORE, '/dashboard');
    if (!cachedData || !cachedData.data) return;

    const data = cachedData.data;
    
    // Example Rule: If overall score is dropping below 80, generate a review prompt
    if (data.progress && data.progress.overall_score < 80) {
      const guidance: LocalGuidance = {
        id: `local_gui_${Date.now()}`,
        learner_id: learnerId,
        text: "We noticed your score dipped a bit. Let's do a quick review of the basics to strengthen your foundation!",
        action: "/learning?filter=review",
        type: "adaptive_review",
        created_at: new Date().toISOString()
      };
      
      // Inject this directly into the cached dashboard data so the UI updates
      // even while completely offline.
      const existingGuidance = data.guidance || [];
      const updatedGuidance = [guidance, ...existingGuidance];
      
      data.guidance = updatedGuidance;
      
      // Store with the same shape api.ts's getFromCache() expects:
      // { data, cachedAt } — otherwise the cache entry would be misread as legacy.
      await db.put(CACHE_STORE, {
        data: data,
        cachedAt: Date.now(),
      }, '/dashboard');
      
      console.log('Client-Side Adaptive Engine generated a new guidance block.');
    }
  } catch (err) {
    console.error('Failed to run adaptive engine locally:', err);
  }
}
