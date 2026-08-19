import { openDB } from 'idb';
import toast from 'react-hot-toast';
import {
  encryptQueuePayload,
  decryptQueuePayload,
  headersToRecord,
  KEY_STORE,
} from './crypto';

export class HttpError extends Error {
  is4xx: boolean;
  constructor(message: string, is4xx: boolean) {
    super(message);
    this.name = 'HttpError';
    this.is4xx = is4xx;
  }
}

const DB_NAME = 'log-db';
export const CACHE_STORE = 'api-cache';
export const QUEUE_STORE = 'sync-queue';

// Cache TTL: entries older than this are considered stale (24 hours)
const CACHE_TTL_MS = 24 * 60 * 60 * 1000;

// Sync retry config
const MAX_RETRIES = 3;
const BASE_BACKOFF_MS = 1000;

let manualOfflineOverride = false;
if (typeof window !== 'undefined') {
  manualOfflineOverride = localStorage.getItem('log_offline_mode') === 'true';
}

// True once the browser confirms our IndexedDB stores are exempt from
// automatic eviction. null until the persist() promise settles.
let storagePersisted: boolean | null = null;

/** Honest view of the storage-persistence grant (null = not yet known). */
export function getStoragePersistence(): boolean | null {
  return storagePersisted;
}

let isAppOnline = true;
if (typeof window !== 'undefined') {
  isAppOnline = navigator.onLine && !manualOfflineOverride;
  // WP-0.1 research round: ask the browser to persist our IndexedDB stores
  // (sync queue + cache). Without a grant, an aggressive mobile OS can evict
  // offline learner work when storage pressure hits. Best-effort and honest:
  // the result is tracked but never fabricated into the UI.
  if (navigator.storage && typeof navigator.storage.persist === 'function') {
    navigator.storage.persist().then((granted) => {
      storagePersisted = granted;
      try {
        localStorage.setItem('log_storage_persisted', granted ? 'true' : 'false');
      } catch { /* non-critical */ }
    }).catch(() => {});
  }
  // F1: flush any queued work as soon as the app boots — a device that was
  // offline when the user last closed the tab must not keep its queue hostage
  // until the next online/offline event fires.
  window.addEventListener('load', () => {
    if (navigator.onLine && !manualOfflineOverride) {
      flushSyncQueue();
    }
  });
  window.addEventListener('online', async () => {
    if (manualOfflineOverride) return;
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
  return openDB(DB_NAME, 4, {
    upgrade(db, oldVersion) {
      if (oldVersion < 1) {
        db.createObjectStore(CACHE_STORE);
      }
      if (oldVersion < 2 && !db.objectStoreNames.contains(QUEUE_STORE)) {
        db.createObjectStore(QUEUE_STORE, { keyPath: 'id', autoIncrement: true });
      }
      // Version 3: no schema change, but cache entries now include cachedAt
      // Version 4 (WP-0.1): sync-keys store holds the AES-GCM queue key.
      // Legacy plaintext queue records are NOT migrated — they flush and
      // disappear naturally; new records are always encrypted.
      if (oldVersion < 4 && !db.objectStoreNames.contains(KEY_STORE)) {
        db.createObjectStore(KEY_STORE);
      }
    },
  });
};



export const setManualOffline = (offline: boolean) => {
  manualOfflineOverride = offline;
  if (typeof window !== 'undefined') {
    if (offline) {
      localStorage.setItem('log_offline_mode', 'true');
    } else {
      localStorage.removeItem('log_offline_mode');
    }
  }
  
  if (!offline && navigator.onLine) {
    isAppOnline = true;
    syncQueue();
  } else if (offline) {
    isAppOnline = false;
  }
};

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6101/api/v1';

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
            clearCredentialsAndRedirect();
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
          // F6: a server 401 must end the session loudly, not fall back to
          // cached data — the cache may belong to another account or an old
          // role, so silently serving it would leak stale/foreign data.
          if (response.status === 401) {
            clearCredentialsAndRedirect();
            throw new HttpError('Session expired. Please log in again.', true);
          }
          throw new HttpError(errData.detail || errData.message || errData.error || `HTTP error! status: ${response.status}`, true);
        }
        // For 5xx errors, we treat it like offline and queue it.
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();

      // Cache successful GETs with timestamp
      if (method === 'GET') {
        const db = await initDB();
        await db.put(CACHE_STORE, { data, cachedAt: Date.now() }, endpoint);
      } else if (method === 'POST' || method === 'PUT' || method === 'DELETE') {
        // F4: successful live mutations must invalidate the related cached
        // reads immediately — otherwise a fresh POST is followed by a stale
        // GET (e.g. a new announcement not appearing until TTL expiry).
        await invalidateRelatedCache(endpoint);
      }
      return data;
    } catch (error: unknown) {
      console.warn('Network fetch failed...', error);
      
      // If it's a 4xx error we threw above, propagate it immediately.
      if (error instanceof HttpError && error.is4xx) {
         throw error; 
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

    // F2: credential endpoints must NEVER be queued for offline replay.
    // Queuing a login/OTP request stores the password/OTP in IndexedDB in
    // plaintext and replays it later; besides leaking secrets, a replayed
    // verify-otp would consume a fresh code. Fail loudly instead.
    if (endpoint.startsWith('/auth/')) {
      throw new HttpError('This action requires a connection. Please try again when online.', true);
    }

    // Deduplication is body-aware: two queued writes to the same endpoint are
    // the same logical action, so the queue keeps ONE entry — but which body
    // wins depends on the action type:
    //   - completions (/activities/:id/complete): keep the higher-accuracy
    //     attempt — the server applies best-score semantics, so a queued
    //     lower-accuracy replay must never overwrite a better one
    //   - everything else (submissions, announcements): last write wins
    // WP-0.1: bodies are encrypted at rest, so the duplicate's plaintext is
    // recovered in memory (and only in memory) for the comparison.
    const existing = await db.getAll(QUEUE_STORE);
    const duplicate = existing.find(
      (req: { endpoint: string; method: string }) => req.endpoint === endpoint && req.method === method
    );

    if (duplicate) {
      const newBody = typeof options.body === 'string' ? options.body : null;
      const existingPlain = await decryptQueuePayload(duplicate);
      const existingBody = existingPlain ? existingPlain.body : null;
      const keptBody = mergeQueuedBody(endpoint, existingBody, newBody);
      if (keptBody === existingBody && keptBody === newBody) {
        console.log('Duplicate request already queued, skipping:', endpoint);
        return { queued: true, status: 202, deduplicated: true };
      }
      // Replace the queued entry in place (keyPath id preserved) with the
      // dominant body — the queue never holds two entries for one action.
      const { enc, fp } = await encryptQueuePayload(endpoint, method, duplicate.headers, keptBody ?? duplicate.body);
      await db.put(QUEUE_STORE, {
        ...duplicate,
        enc,
        fp,
        timestamp: new Date().toISOString(),
        retryCount: 0,
      });
      console.log('Queued request updated with newer payload:', endpoint);
      return { queued: true, status: 202, deduplicated: true };
    }

    const newBody = typeof options.body === 'string' ? options.body : null;
    const { enc, fp } = await encryptQueuePayload(endpoint, method, options.headers, newBody);
    await db.add(QUEUE_STORE, {
      endpoint,
      method,
      enc,
      fp,
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

// mergeQueuedBody decides which body the queue keeps when the same endpoint is
// queued twice. Completions keep the attempt with the best accuracy (ties keep
// the newest); all other endpoints keep the newest payload.
function mergeQueuedBody(
  endpoint: string,
  oldBody: string | null | undefined,
  newBody: string | null | undefined
): string | null | undefined {
  if (!endpoint.includes('/activities/') || !endpoint.includes('/complete')) {
    return newBody;
  }

  const parse = (body: string | null | undefined): { correct_count?: number; total_count?: number } | null => {
    if (typeof body !== 'string' || !body) return null;
    try {
      return JSON.parse(body);
    } catch {
      return null;
    }
  };

  const oldStats = parse(oldBody);
  const newStats = parse(newBody);
  const oldAccuracy = oldStats && oldStats.total_count ? (oldStats.correct_count || 0) / oldStats.total_count : -1;
  const newAccuracy = newStats && newStats.total_count ? (newStats.correct_count || 0) / newStats.total_count : -1;

  // A higher (or equally good) fresh attempt replaces the queued one; a lower
  // attempt is only a replay and must not overwrite the queued best.
  return newAccuracy >= oldAccuracy ? newBody : oldBody;
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
    // Assignment submissions change the learner's assignment list
    if (endpoint.includes('/submit')) {
      await db.delete(CACHE_STORE, '/assignments');
    }
    // F4: school-module mutations (classes, enrollment, announcements,
    // assignments, roles) read through dynamic cache keys (e.g. per-class
    // assignment lists) that cannot be enumerated here — clear the whole
    // cache instead. These actions are low-frequency, so the blast radius is
    // one stale-free page load, not a performance concern.
    if (
      endpoint.includes('/classes') ||
      endpoint.includes('/enroll') ||
      endpoint.includes('/announcements') ||
      endpoint.includes('/assignments') ||
      endpoint.includes('/users') ||
      endpoint.includes('/roles')
    ) {
      await clearApiCache();
    }
  } catch (e) {
    console.warn('Cache invalidation failed (non-critical)', e);
  }
}

/** Clears every entry in the API cache store. */
export async function clearApiCache(): Promise<void> {
  try {
    const db = await initDB();
    const keys = await db.getAllKeys(CACHE_STORE);
    await Promise.all(keys.map((key) => db.delete(CACHE_STORE, key)));
  } catch (e) {
    console.warn('Cache clear failed (non-critical)', e);
  }
}

/** Clears credentials and bounces to the login screen (401 / expired session). */
export function clearCredentialsAndRedirect(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem('log_token');
  localStorage.removeItem('log_user');
  document.cookie = 'log_token=; path=/; max-age=0';
  if (!window.location.pathname.startsWith('/login')) {
    window.location.href = '/login';
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
      // F2 guard (defense in depth): auth requests can only have entered the
      // queue from an older build — drop them instead of replaying secrets.
      if (req.endpoint && req.endpoint.startsWith('/auth/')) {
        await db.delete(QUEUE_STORE, req.id);
        failedCount++;
        continue;
      }


      for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
        try {
          // WP-0.1: queued payloads are encrypted at rest — recover the
          // plaintext in memory before replay. A failed decrypt keeps the
          // record (never delete learner work on a key hiccup).
          const plain = await decryptQueuePayload(req);
          if (!plain) {
            console.error(`Cannot decrypt ${req.endpoint} — keeping in queue`);
            failedCount++;
            break;
          }
          // Re-attach the current token so records queued under an expired
          // session can still sync after the user logs back in.
          const replayHeaders: Record<string, string> = headersToRecord(plain.headers);
          if (typeof window !== 'undefined') {
            const freshToken = localStorage.getItem('log_token');
            if (freshToken) replayHeaders['Authorization'] = `Bearer ${freshToken}`;
          }
          const response = await fetch(`${BASE_URL}${req.endpoint}`, {
            method: req.method,
            headers: replayHeaders,
            body: plain.body,
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

          // 403 consent_required (WP-0.1 enforcement round): the server-side
          // consent gate rejected a learner mutation. This is NOT a terminal
          // error — the record stays queued and the flush stops so the learner
          // (or guardian) can grant consent, after which the next online event
          // syncs it. Deleting here would silently lose offline work.
          if (response.status === 403) {
            const errData = await response.json().catch(() => ({}));
            if (errData.code === 'consent_required') {
              console.error(`Guardian consent required for ${req.endpoint} — queue preserved`);
              toast('Guardian consent is needed to sync saved changes. Please grant it in Settings.', { icon: '🛡️' });
              failedCount++;
              return;
            }
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
 * Public flush entry point — used by the boot loader, SyncIsland's "Sync now"
 * button, the command palette, and post-login/sneakernet-import flows.
 */
export async function flushSyncQueue(): Promise<{ synced: number; failed: number }> {
  const before = await getSyncQueueCount();
  if (before === 0) return { synced: 0, failed: 0 };
  await syncQueue();
  const after = await getSyncQueueCount();
  return { synced: before - after, failed: after };
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
