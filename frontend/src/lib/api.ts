import { openDB } from 'idb';

const DB_NAME = 'log-db';
const STORE_NAME = 'api-cache';

// Initialize IndexedDB
const initDB = async () => {
  return openDB(DB_NAME, 1, {
    upgrade(db) {
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME);
      }
    },
  });
};

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

/**
 * Fetch data from network, fallback to IndexedDB if offline.
 */
export async function fetchWithCache(endpoint: string, options: RequestInit = {}) {
  const url = `${BASE_URL}${endpoint}`;
  const isOnline = typeof window !== 'undefined' && navigator.onLine;

  if (isOnline) {
    try {
      const response = await fetch(url, options);
      if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);

      const data = await response.json();

      // Cache the successful response
      if (options.method === 'GET' || !options.method) {
        const db = await initDB();
        await db.put(STORE_NAME, data, endpoint);
      }

      return data;
    } catch (error) {
      console.warn('Network fetch failed, attempting to read from cache...', error);
      return getFromCache(endpoint);
    }
  } else {
    console.log('App is offline, serving from cache:', endpoint);
    return getFromCache(endpoint);
  }
}

async function getFromCache(endpoint: string) {
  try {
    const db = await initDB();
    const data = await db.get(STORE_NAME, endpoint);
    if (!data) throw new Error('No cached data available');
    return data;
  } catch (error) {
    console.error('Failed to retrieve from cache', error);
    throw error;
  }
}

// Ensure the local dev server knows how to fetch from our backend
