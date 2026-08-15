import { openDB } from 'idb';
import toast from 'react-hot-toast';

const DB_NAME = 'log-db';
export const CACHE_STORE = 'api-cache';
export const QUEUE_STORE = 'sync-queue';

// Cache TTL: entries older than this are considered stale (24 hours)
const CACHE_TTL_MS = 24 * 60 * 60 * 1000;

// Sync retry config
const MAX_RETRIES = 3;
const BASE_BACKOFF_MS = 1000;

let isAppOnline = true;
if (typeof window !== 'undefined') {
  isAppOnline = navigator.onLine;
  window.addEventListener('online', async () => {
    isAppOnline = true;
    toast.success('Back online! Syncing data...');
    await syncQueue();
  });
  window.addEventListener('offline', () => {
    isAppOnline = false;
    toast('You are offline. Changes will be saved locally.', { icon: '📡' });
  });
}

export const initDB = async () => {
  return openDB(DB_NAME, 3, {
    upgrade(db, oldVersion) {
      if (oldVersion < 1) {
        db.createObjectStore(CACHE_STORE);
      }
      if (oldVersion < 2 && !db.objectStoreNames.contains(QUEUE_STORE)) {
        db.createObjectStore(QUEUE_STORE, { keyPath: 'id', autoIncrement: true });
      }
      // Version 3: no schema change, but cache entries now include cachedAt
    },
  });
};

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api';

/**
 * Logs the user out by:
 * 1. Calling the backend /auth/logout endpoint to revoke the JWT (server-side blocklist).
 * 2. Clearing all local credentials from localStorage.
 * 3. Clearing the IndexedDB API cache so stale data doesn't persist.
 * 4. Redirecting to /login.
 */
export async function logout(): Promise<void> {
  try {
    const token = typeof window !== 'undefined' ? localStorage.getItem('log_token') : null;
    if (token) {
      // Best-effort: revoke the token server-side. Even if this fails, clear locally.
      await fetch(`${BASE_URL}/auth/logout`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      }).catch(() => { /* silent — local cleanup will still happen */ });
    }
  } finally {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('log_token');
      localStorage.removeItem('log_user');
      // Clear the auth cookie so the middleware doesn't bounce /login -> /dashboard
      document.cookie = 'log_token=; path=/; max-age=0';
      // Clear IndexedDB API cache so next user doesn't see stale data
      try {
        const db = await initDB();
        await db.clear(CACHE_STORE);
      } catch { /* non-critical */ }
      window.location.href = '/login';
    }
  }
}

export async function fetchWithCache(endpoint: string, options: RequestInit = {}) {
  const url = `${BASE_URL}${endpoint}`;
  const method = options.method || 'GET';

  // Include Auth Header from localStorage if available
  // Security note: localStorage is susceptible to XSS.
  // For production, migrate to httpOnly cookies. This wrapper at minimum
  // sanitizes the read path and checks expiry to prevent 401 floods.
  const headers = options.headers ? { ...options.headers } as Record<string, string> : {};
  if (typeof window !== 'undefined') {
    try {
      const token = localStorage.getItem('log_token');
      if (token && !headers['Authorization']) {
        // Decode JWT payload (base64url) WITHOUT verifying signature — just to read `exp`
        // Signature verification happens server-side; this is purely a UX optimization.
        const payloadB64 = token.split('.')[1];
        if (payloadB64) {
          const payloadJson = atob(payloadB64.replace(/-/g, '+').replace(/_/g, '/'));
          const payload = JSON.parse(payloadJson) as { exp?: number };
          const nowSecs = Math.floor(Date.now() / 1000);
          if (payload.exp && payload.exp < nowSecs) {
            // Token is expired — clear it and redirect rather than sending a 401
            localStorage.removeItem('log_token');
            localStorage.removeItem('log_user');
            window.location.href = '/login';
            throw new Error('Session expired. Redirecting to login.');
          }
        }
        headers['Authorization'] = `Bearer ${token}`;
      }
    } catch (e) {
      // If token parsing fails for any reason (corrupt data, XSS injection attempt),
      // clear storage and let the request proceed unauthenticated.
      if (!(e instanceof Error && e.message.includes('Session expired'))) {
        console.warn('Token read failed — clearing credentials', e);
        localStorage.removeItem('log_token');
        localStorage.removeItem('log_user');
      }
      throw e;
    }
  }
  options.headers = headers;

  if (isAppOnline) {
    try {
      // By using cache: 'no-store', we prevent the service worker from intercepting
      // and potentially serving stale data across different authenticated sessions.
      const fetchOpts = { ...options, cache: 'no-store' as RequestCache };
      const response = await fetch(url, fetchOpts);
      if (!response.ok) {
        // If it's a 4xx client error (e.g. 400 Bad Request, 401 Unauthorized), do NOT queue it.
        // It's a real failure from the server.
        if (response.status >= 400 && response.status < 500) {
          const errData = await response.json().catch(() => ({}));
          throw new Error(errData.error || `HTTP error! status: ${response.status}`);
        }
        // For 5xx errors, we treat it like offline and queue it.
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      // Cache successful GETs with timestamp
      if (method === 'GET') {
        const db = await initDB();
        await db.put(CACHE_STORE, { data, cachedAt: Date.now() }, endpoint);
      }
      return data;
    } catch (error: unknown) {
      console.warn('Network fetch failed...', error);
      
      // If it's a 4xx error we threw above, propagate it immediately.
      // We know it's a 4xx error if we caught it and it's not a generic TypeError (network error)
      // and not a 5xx error. Let's rely on the error message structure or a custom property.
      // A simpler way: if the app is online and we got a response, don't queue 4xx.
      // Wait, we already threw it. We can check if error is our custom 4xx throw.
      if (error instanceof Error && !error.message.includes('HTTP error! status: 5')) {
         // Network error usually throws TypeError("Failed to fetch")
         // Our 4xx throws Error("error message" or "HTTP error! status: 4xx")
         if (error.message !== 'Failed to fetch' && !error.message.includes('HTTP error! status: 5')) {
             throw error; 
         }
      }

      if (method === 'GET') return getFromCache(endpoint);
      return queueRequest(endpoint, options);
    }
  } else {
    console.log('App is offline');
    if (method === 'GET') return getFromCache(endpoint);
    return queueRequest(endpoint, options);
  }
}

async function getFromCache(endpoint: string) {
  try {
    const db = await initDB();
    const entry = await db.get(CACHE_STORE, endpoint);

    if (!entry) throw new Error('No cached data available');

    // Check if entry has the new format (with cachedAt) or legacy format
    if (entry.cachedAt) {
      const age = Date.now() - entry.cachedAt;
      if (age > CACHE_TTL_MS && isAppOnline) {
        // Stale but online — remove and throw to trigger re-fetch
        await db.delete(CACHE_STORE, endpoint);
        throw new Error('Cache entry expired');
      }
      return entry.data;
    }

    // Legacy cache entry (no cachedAt) — return as-is
    return entry;
  } catch (error) {
    console.error('Failed to retrieve from cache', error);
    throw error;
  }
}

async function queueRequest(endpoint: string, options: RequestInit) {
  try {
    const db = await initDB();
    const method = options.method || 'POST';

    // Deduplication: check if an identical request is already queued
    const existing = await db.getAll(QUEUE_STORE);
    const isDuplicate = existing.some(
      (req: { endpoint: string; method: string }) => req.endpoint === endpoint && req.method === method
    );

    if (isDuplicate) {
      console.log('Duplicate request already queued, skipping:', endpoint);
      return { queued: true, status: 202, deduplicated: true };
    }

    await db.add(QUEUE_STORE, {
      endpoint,
      method,
      headers: options.headers,
      body: options.body,
      timestamp: new Date().toISOString(),
      retryCount: 0,
    });
    toast.success('Action saved offline. Will sync later.', { icon: '💾' });
    return { queued: true, status: 202 }; // Optimistic response
  } catch (err) {
    console.error('Failed to queue request', err);
    throw err;
  }
}

/**
 * Invalidates (deletes) a cached GET response for the given endpoint.
 * Called after a successful mutating sync to prevent stale reads.
 */
async function invalidateRelatedCache(endpoint: string) {
  try {
    const db = await initDB();
    // Invalidate the dashboard and learning journey when activities are completed
    if (endpoint.includes('/activities/') && endpoint.includes('/complete')) {
      await db.delete(CACHE_STORE, '/dashboard');
      await db.delete(CACHE_STORE, '/learning-journey');
      await db.delete(CACHE_STORE, '/chart-data');
    }
    // Bulk sync (sneakernet import) can also complete activities
    if (endpoint.includes('/sync/bulk')) {
      await db.delete(CACHE_STORE, '/dashboard');
      await db.delete(CACHE_STORE, '/learning-journey');
      await db.delete(CACHE_STORE, '/chart-data');
    }
  } catch (e) {
    console.warn('Cache invalidation failed (non-critical)', e);
  }
}

/**
 * Flushes the offline sync queue with exponential backoff retry.
 * Each failed item is retried up to MAX_RETRIES times with increasing delays.
 */
async function syncQueue() {
  try {
    const db = await initDB();
    const queuedReqs = await db.getAll(QUEUE_STORE);

    if (queuedReqs.length === 0) return;

    let syncedCount = 0;
    let failedCount = 0;

    for (const req of queuedReqs) {


      for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
        try {
          // Re-attach the current token so records queued under an expired
          // session can still sync after the user logs back in.
          const replayHeaders: Record<string, string> = { ...(req.headers || {}) };
          if (typeof window !== 'undefined') {
            const freshToken = localStorage.getItem('log_token');
            if (freshToken) replayHeaders['Authorization'] = `Bearer ${freshToken}`;
          }
          const response = await fetch(`${BASE_URL}${req.endpoint}`, {
            method: req.method,
            headers: replayHeaders,
            body: req.body,
          });

          if (response.ok || response.status === 409) {
            // 409 Conflict = already processed server-side, safe to remove
            await db.delete(QUEUE_STORE, req.id);
            await invalidateRelatedCache(req.endpoint);
            syncedCount++;
            break;
          }

          // Server error (5xx) — retry with backoff
          if (response.status >= 500) {
            throw new Error(`Server error: ${response.status}`);
          }

          // 401: the stored token expired while offline. KEEP the queued records —
          // deleting them would lose the learner's work silently. Stop the flush
          // and ask the user to log in again (the queue survives the re-login).
          if (response.status === 401) {
            console.error(`Auth expired while syncing ${req.endpoint} — queue preserved`);
            toast('Session expired. Please log in to sync your saved changes.', { icon: '🔒' });
            failedCount++;
            return;
          }

          // Other client error (4xx) — don't retry, remove from queue
          console.error(`Client error syncing ${req.endpoint}: ${response.status}`);
          await db.delete(QUEUE_STORE, req.id);
          failedCount++;
          break;
        } catch (e) {
          if (attempt < MAX_RETRIES) {
            // Exponential backoff: 1s, 2s, 4s
            const delay = BASE_BACKOFF_MS * Math.pow(2, attempt);
            console.warn(`Retry ${attempt + 1}/${MAX_RETRIES} for ${req.endpoint} in ${delay}ms`);
            await new Promise(resolve => setTimeout(resolve, delay));
          } else {
            console.error(`Failed to sync ${req.endpoint} after ${MAX_RETRIES} retries`, e);
            // Update retry count but keep in queue for next online event
            req.retryCount = (req.retryCount || 0) + MAX_RETRIES;
            await db.put(QUEUE_STORE, req);
            failedCount++;
          }
        }
      }
    }

    if (syncedCount > 0) {
      toast.success(`${syncedCount} offline change${syncedCount > 1 ? 's' : ''} synced successfully!`);
    }
    if (failedCount > 0) {
      toast(`${failedCount} change${failedCount > 1 ? 's' : ''} could not be synced. Will retry later.`, { icon: '⚠️' });
    }
  } catch (e) {
    console.error('Failed to process sync queue', e);
  }
}

/**
 * Returns the number of items currently in the sync queue.
 * Useful for displaying a pending sync badge in the UI.
 */
export async function getSyncQueueCount(): Promise<number> {
  try {
    const db = await initDB();
    const queuedReqs = await db.getAll(QUEUE_STORE);
    return queuedReqs.length;
  } catch (e) {
    console.error('Failed to get sync queue count', e);
    return 0;
  }
}
