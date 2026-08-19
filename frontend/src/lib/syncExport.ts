import { initDB, QUEUE_STORE } from './api';
import { decryptQueuePayload } from './crypto';

/**
 * Exports the offline sync queue to a JSON string for "Sneakernet" syncing.
 *
 * WP-0.1: records are encrypted at rest, so the export decrypts them into the
 * user's own file. This is a deliberate, documented escape hatch — the file is
 * the user's personal data, meant to be carried (and re-imported) on another
 * device that does not share the queue key. Treat the .logsync file with the
 * same care as the account password. Imported records are re-stored as
 * plaintext and re-encrypt on the destination device's next queue write.
 */
export async function exportSyncQueue(): Promise<string> {
  try {
    const db = await initDB();
    const queuedReqs = await db.getAll(QUEUE_STORE);

    if (queuedReqs.length === 0) {
      throw new Error('Sync queue is empty. Nothing to export.');
    }

    const plainRecords: Array<Record<string, unknown>> = [];
    for (const req of queuedReqs) {
      const plain = await decryptQueuePayload(req);
      if (!plain) {
        // Never drop a record from an export — surface the failure instead so
        // the user knows the file would be incomplete.
        throw new Error('A queued record could not be decrypted. Export aborted.');
      }
      const meta = { ...req };
      delete meta.enc;
      delete meta.fp;
      plainRecords.push({ ...meta, headers: plain.headers, body: plain.body });
    }

    const exportPayload = {
      version: '1.1', // 1.1: decrypted payloads (see note above)
      timestamp: new Date().toISOString(),
      data: plainRecords,
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
 * Imported records are stored as plaintext (they were decrypted on the source
 * device); the destination's next queue write re-encrypts them.
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
          // Deduplicate before adding. Bodies are plaintext in the file and
          // in imported records, so a direct comparison is honest. Records
          // already encrypted on this device are decrypted for the check.
          let isDuplicate = false;
          for (const existing of existingReqs) {
            if (existing.endpoint !== req.endpoint || existing.method !== req.method) continue;
            const existingPlain = await decryptQueuePayload(existing);
            if (!existingPlain) continue;
            if ((existingPlain.body ?? null) === (req.body ?? null)) {
              isDuplicate = true;
              break;
            }
          }

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