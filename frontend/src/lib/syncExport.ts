import { initDB, QUEUE_STORE } from './api';

/**
 * Exports the offline sync queue to a JSON string for "Sneakernet" syncing.
 */
export async function exportSyncQueue(): Promise<string> {
  try {
    const db = await initDB();
    const queuedReqs = await db.getAll(QUEUE_STORE);
    
    if (queuedReqs.length === 0) {
      throw new Error('Sync queue is empty. Nothing to export.');
    }

    const exportPayload = {
      version: '1.0',
      timestamp: new Date().toISOString(),
      data: queuedReqs,
    };

    return JSON.stringify(exportPayload);
  } catch (err) {
    console.error('Failed to export sync queue:', err);
    throw err;
  }
}

/**
 * Triggers a download of the exported sync queue as a .logsync file.
 */
export async function downloadSyncFile() {
  const dataStr = await exportSyncQueue();
  const blob = new Blob([dataStr], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  
  const a = document.createElement('a');
  a.href = url;
  a.download = `progress_sync_${new Date().getTime()}.logsync`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

/**
 * Imports a .logsync file, reads the payload, and merges it into the local QUEUE_STORE.
 * This is useful if a student moved their data to another device, or a teacher is collecting it.
 */
export async function importSyncFile(file: File): Promise<number> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = async (e) => {
      try {
        const text = e.target?.result as string;
        const payload = JSON.parse(text);
        
        if (!payload.data || !Array.isArray(payload.data)) {
          throw new Error('Invalid .logsync file format');
        }

        const db = await initDB();
        let importedCount = 0;
        const existingReqs = await db.getAll(QUEUE_STORE);

        for (const req of payload.data) {
          // Deduplicate before adding
          const isDuplicate = existingReqs.some(
            (existing: { endpoint: string; method: string; body?: string }) => 
              existing.endpoint === req.endpoint && 
              existing.method === req.method &&
              existing.body === req.body
          );

          if (!isDuplicate) {
            // Remove the old auto-incrementing ID so it gets a fresh one on this device
            // eslint-disable-next-line @typescript-eslint/no-unused-vars
            const { id, ...reqWithoutId } = req;
            await db.add(QUEUE_STORE, {
              ...reqWithoutId,
              timestamp: new Date().toISOString(), // mark as imported recently
            });
            importedCount++;
          }
        }
        resolve(importedCount);
      } catch (err) {
        console.error('Failed to import sync file', err);
        reject(err);
      }
    };
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsText(file);
  });
}
