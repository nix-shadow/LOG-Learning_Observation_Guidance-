# Advanced Offline Synchronization & PWA Architecture

## 1. Overview
The LOG platform is engineered to function seamlessly in low-bandwidth, intermittent, or completely offline environments. The offline subsystem comprises two coordinated layers:

1. **Service Worker Layer (`next-pwa` + Workbox):** Caches HTML, JS, CSS, fonts, and static assets so the app loads instantly without network connectivity.
2. **Data & Mutation Layer (`src/lib/api.ts` + `idb`):** Intercepts all REST API traffic, caching GET responses in an IndexedDB `api-cache` store and queuing mutating actions (POST/PUT/DELETE) in a `sync-queue` store.

---

## 2. IndexedDB Dual-Store Schema

The browser database is initialized with database name `log-db` (version `3`):

```
+-------------------------------------------------------------------+
|                        IndexedDB: "log-db"                        |
+-------------------------------------------------------------------+
|  Store 1: "api-cache"                                             |
|  - Key: Endpoint Path (e.g. "/dashboard", "/learning-journey")    |
|  - Value: { data: <JSON Response Payload>, cachedAt: <epoch ms> } |
|  - TTL: 24 hours — stale entries are purged when back online      |
+-------------------------------------------------------------------+
|  Store 2: "sync-queue"                                            |
|  - Key: Auto-incrementing Integer ID                              |
|  - Value: {                                                       |
|      endpoint: string,                                            |
|      method: string,                                              |
|      headers: Record<string, string>,                             |
|      body: any,                                                   |
|      timestamp: string,                                           |
|      retryCount: number                                           |
|    }                                                              |
|  - Deduplicated by (endpoint, method) before enqueueing           |
+-------------------------------------------------------------------+
```

> **Cache format note:** version 3 unified the cache entry shape to `{ data, cachedAt }`.
> Legacy entries without `cachedAt` are still readable for backward compatibility.

---

## 3. Request Interception Pipeline (`fetchWithCache`)

All API interactions across the frontend pass through `fetchWithCache(endpoint, options)`:

```
                          fetchWithCache(endpoint, options)
                                         |
                                         v
                         Attach Bearer Token from localStorage
                                         |
                                         v
                              Is navigator.onLine true?
                                    /        \
                             [YES] /          \ [NO]
                                  /            \
                                 v              v
                        Execute fetch(url)   Is Method GET?
                             /       \          /     \
                     [SUCCESS]       [FAIL] [YES]      \ [NO (POST/PUT/DELETE)]
                        /              |     /          v
                       v               |    v     queueRequest(endpoint)
                Is Method GET?         |  getFromCache   - Add to 'sync-queue'
                  /        \           |   (endpoint)    - Return { status: 202 }
             [YES]          \ [NO]     |                 - Toast: Saved offline
              /              \         |
             v                v        |
      Update 'api-cache'   Return data |
      Return data                      v
                         +-----------------------------+
                         | Fallback to:                |
                         | - GET: getFromCache()       |
                         | - Mutate: queueRequest()    |
                         +-----------------------------+
```

### 3.1 Handling GET Requests
- **When Online:** Fetches fresh data from the server. Upon a `200 OK` response, updates the corresponding key in the `api-cache` store.
- **When Offline / Network Failure:** Retrieves the latest cached payload from `api-cache`. If no cached data exists, displays an offline notice without crashing.

### 3.2 Handling Mutating Requests (POST, PUT, DELETE)
- **When Online:** Dispatches the HTTP request directly to the backend.
- **When Offline / Network Failure:** Intercepts the request and persists it to the `sync-queue` store. Returns an optimistic response:
  ```json
  {
    "queued": true,
    "status": 202
  }
  ```
  A non-intrusive toast notification alerts the user: *"Action saved offline. Will sync later."*

---

## 4. Background Synchronization Worker (`syncQueue`)

When the browser detects that internet connectivity is restored, the `window.online` event listener triggers the reconciliation process:

```typescript
window.addEventListener('online', async () => {
  isAppOnline = true;
  toast.success('Back online! Syncing data...');
  await syncQueue();
});
```

### Reconciliation Workflow:
1. Opens `log-db` and queries all entries from `sync-queue`.
2. Iterates over queued mutations in strict FIFO (First-In, First-Out) order.
3. Dispatches each HTTP request with original headers, payload, and auth token to the backend.
4. On success (or `409 Conflict` — already processed server-side), removes the entry from `sync-queue` and invalidates related GET caches (`/dashboard`, `/learning-journey`, `/chart-data`).
5. **Retry policy:** each item is retried up to 3 times with exponential backoff (`1s → 2s → 4s`). Server 5xx errors retry; client 4xx errors are removed (non-recoverable).
6. Displays a completion toast: *"N offline changes synced successfully!"* or a warning for unsynced items.

---

## 5. Sneakernet Sync (`.logsync` Files)

For learners with no connectivity at all, the dashboard exposes export/import via `src/lib/syncExport.ts`:

- **Export:** serializes the current `sync-queue` into a JSON payload (`version: "1.0"`, timestamp, data array) and downloads it as `progress_sync_<ts>.logsync`.
- **Import:** reads a `.logsync` file, deduplicates entries against the local queue (by endpoint + method + body), strips stale auto-increment IDs, and re-enqueues them for sync.
- **Upload:** the file can cross devices (USB / school computer) and be replayed by the queue, or bulk-uploaded directly via `POST /api/sync/bulk`.
- **Offline roster:** Moderators can pre-fetch `/moderator/roster` so the classroom table works fully disconnected.

---

## 6. Client-Side Adaptive Engine (`adaptiveEngine.ts`)

`evaluateLocalAdaptivity()` inspects the cached `/dashboard` payload and generates rule-based guidance **without any network call** (e.g. "score dipped below 80 → review prompt"). It writes back into the cache using the unified `{ data, cachedAt }` shape so the UI updates instantly while offline.

---

## 7. Automated Testing of Offline Layer

The offline layer is tested using Jest in `frontend/__tests__/api.test.ts` and `frontend/src/lib/syncExport.test.ts` (11 tests total):

- **Network-First Test:** Verifies network execution and cache writes when online.
- **Error Fallback Test:** Verifies automatic fallback to cached data when `fetch()` throws an error.
- **Offline Bypass Test:** Verifies that when `offline` is triggered, `fetch()` is not called and cached data is immediately returned.
- **Queuing & Dedup Tests:** Mutating requests queue offline with an optimistic `202`, and identical requests are deduplicated.
- **Cache TTL Tests:** Fresh cache within 24h TTL is returned; legacy (< v3) entries still read correctly.
- **Sneakernet Tests:** Empty queue export throws; import strips stale IDs and skips duplicates.
