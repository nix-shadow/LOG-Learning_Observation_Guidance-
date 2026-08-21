// WP-1.4: local-first review schedule store (IndexedDB via idb).
// The scheduler state is per-device local data — it survives offline and is
// never the source of truth for metrics. Every review is ALSO a real
// completion replay, which already flows through the sync queue, so what a
// learner sees here is always backed by honest backend completions.

import { openDB } from 'idb';
import { ReviewState } from './spacedRepetition';

const DB_NAME = 'log-db';
const REVIEW_STORE = 'review-schedule';

export const openReviewDB = async () => {
  return openDB(DB_NAME, 5, {
    upgrade(db, oldVersion) {
      // Version 5 (WP-1.4): review schedule store. Created idempotently here
      // and in api.ts's initDB — whichever connection upgrades first wins,
      // the other sees version 5 and skips.
      if (oldVersion < 5 && !db.objectStoreNames.contains(REVIEW_STORE)) {
        db.createObjectStore(REVIEW_STORE, { keyPath: 'activityId' });
      }
    },
  });
};

export const getReviewState = async (activityId: string): Promise<ReviewState | undefined> => {
  const db = await openReviewDB();
  return db.get(REVIEW_STORE, activityId);
};

export const saveReviewState = async (state: ReviewState): Promise<void> => {
  const db = await openReviewDB();
  await db.put(REVIEW_STORE, state);
};

export const getAllReviewStates = async (): Promise<ReviewState[]> => {
  const db = await openReviewDB();
  return db.getAll(REVIEW_STORE);
};