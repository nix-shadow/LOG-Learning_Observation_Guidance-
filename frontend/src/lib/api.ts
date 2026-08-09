import { openDB } from 'idb';
import toast from 'react-hot-toast';

const DB_NAME = 'log-db';
const CACHE_STORE = 'api-cache';
const QUEUE_STORE = 'sync-queue';

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

const initDB = async () => {
  return openDB(DB_NAME, 2, {
    upgrade(db, oldVersion) {
      if (oldVersion < 1) {
        db.createObjectStore(CACHE_STORE);
      }
      if (oldVersion < 2 && !db.objectStoreNames.contains(QUEUE_STORE)) {
        db.createObjectStore(QUEUE_STORE, { keyPath: 'id', autoIncrement: true });
      }
    },
  });
};

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

export async function fetchWithCache(endpoint: string, options: RequestInit = {}) {
  const url = `${BASE_URL}${endpoint}`;
  const method = options.method || 'GET';

  // Include Auth Header from localStorage if available
  const headers = options.headers ? { ...options.headers } as Record<string, string> : {};
  if (typeof window !== 'undefined') {
    const token = localStorage.getItem('log_token');
    if (token && !headers['Authorization']) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  }
  options.headers = headers;

  if (isAppOnline) {
    try {
      const response = await fetch(url, options);
      if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);

      const data = await response.json();

      // Cache successful GETs
      if (method === 'GET') {
        const db = await initDB();
        await db.put(CACHE_STORE, data, endpoint);
      }
      return data;
    } catch (error) {
      console.warn('Network fetch failed...', error);
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
    const data = await db.get(CACHE_STORE, endpoint);
    if (!data) throw new Error('No cached data available');
    return data;
  } catch (error) {
    console.error('Failed to retrieve from cache', error);
    throw error;
  }
}

async function queueRequest(endpoint: string, options: RequestInit) {
  try {
    const db = await initDB();
    await db.add(QUEUE_STORE, {
      endpoint,
      method: options.method,
      headers: options.headers,
      body: options.body,
      timestamp: new Date().toISOString(),
    });
    toast.success('Action saved offline. Will sync later.', { icon: '💾' });
    return { queued: true, status: 202 }; // Mock optimistic response
  } catch (err) {
    console.error('Failed to queue request', err);
    throw err;
  }
}

async function syncQueue() {
  try {
    const db = await initDB();
    const queuedReqs = await db.getAll(QUEUE_STORE);

    if (queuedReqs.length === 0) return;

    for (const req of queuedReqs) {
      try {
        await fetch(`${BASE_URL}${req.endpoint}`, {
          method: req.method,
          headers: req.headers,
          body: req.body,
        });
        await db.delete(QUEUE_STORE, req.id);
      } catch (e) {
        console.error('Failed to sync queued request', req.id, e);
        // Will remain in queue to retry next time
      }
    }
    toast.success('Offline changes synced successfully!');
  } catch (e) {
    console.error('Failed to process sync queue', e);
  }
}
