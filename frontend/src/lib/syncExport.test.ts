import { exportSyncQueue, importSyncFile } from './syncExport';
import { initDB } from './api';

// Mock the api module
jest.mock('./api', () => ({
  initDB: jest.fn(),
  QUEUE_STORE: 'sync-queue',
}));

describe('syncExport', () => {
  let mockDb: { getAll: jest.Mock; add: jest.Mock };

  beforeEach(() => {
    mockDb = {
      getAll: jest.fn(),
      add: jest.fn(),
    };
    (initDB as jest.Mock).mockResolvedValue(mockDb);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('exportSyncQueue', () => {
    it('should throw an error if the queue is empty', async () => {
      mockDb.getAll.mockResolvedValue([]);
      await expect(exportSyncQueue()).rejects.toThrow('Sync queue is empty. Nothing to export.');
    });

    it('should export the queue as a JSON string', async () => {
      const mockQueue = [
        { id: 1, endpoint: '/api/test', method: 'POST', body: '{"data":"test"}' }
      ];
      mockDb.getAll.mockResolvedValue(mockQueue);

      const result = await exportSyncQueue();
      const payload = JSON.parse(result);

      expect(payload.version).toBe('1.0');
      expect(payload.data).toEqual(mockQueue);
      expect(payload.timestamp).toBeDefined();
    });
  });

  describe('importSyncFile', () => {
    it('should parse the file and insert non-duplicate records', async () => {
      mockDb.getAll.mockResolvedValue([
        { id: 1, endpoint: '/api/duplicate', method: 'POST' }
      ]);

      const payload = {
        version: '1.0',
        data: [
          { id: 2, endpoint: '/api/duplicate', method: 'POST' }, // Should be skipped
          { id: 3, endpoint: '/api/new', method: 'POST' }        // Should be added
        ]
      };

      const file = new File([JSON.stringify(payload)], 'test.logsync', { type: 'application/json' });

      // Mock FileReader
      const mockFileReader = {
        readAsText: function(this: FileReader) {
          if (this.onload) {
            this.onload({ target: { result: JSON.stringify(payload) } } as unknown as ProgressEvent<FileReader>);
          }
        }
      };
      global.FileReader = jest.fn(() => mockFileReader) as unknown as typeof FileReader;

      const count = await importSyncFile(file);

      expect(count).toBe(1);
      expect(mockDb.add).toHaveBeenCalledTimes(1);
      expect(mockDb.add.mock.calls[0][1].endpoint).toBe('/api/new');
      expect(mockDb.add.mock.calls[0][1].id).toBeUndefined(); // ID should be stripped
    });
  });
});
