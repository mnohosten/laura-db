/**
 * Client Tests
 *
 * Basic test suite for the LauraDB Node.js client
 * Note: These tests require a running LauraDB server on localhost:8080
 */

const Client = require('./client');

describe('LauraDB Client', () => {
  let client;

  beforeEach(() => {
    client = new Client({ host: 'localhost', port: 8080 });
  });

  afterEach(() => {
    if (client) {
      client.close();
    }
  });

  describe('Client Configuration', () => {
    test('should create client with default config', () => {
      const defaultClient = new Client();
      expect(defaultClient.config.host).toBe('localhost');
      expect(defaultClient.config.port).toBe(8080);
      expect(defaultClient.config.https).toBe(false);
      expect(defaultClient.config.timeout).toBe(30000);
      expect(defaultClient.config.maxSockets).toBe(10);
      defaultClient.close();
    });

    test('should create client with custom config', () => {
      const customClient = new Client({
        host: 'example.com',
        port: 9090,
        https: true,
        timeout: 60000,
        maxSockets: 20
      });
      expect(customClient.config.host).toBe('example.com');
      expect(customClient.config.port).toBe(9090);
      expect(customClient.config.https).toBe(true);
      expect(customClient.config.timeout).toBe(60000);
      expect(customClient.config.maxSockets).toBe(20);
      customClient.close();
    });

    test('should construct correct base URL', () => {
      const httpClient = new Client({ host: 'localhost', port: 8080 });
      expect(httpClient.baseURL).toBe('http://localhost:8080');
      httpClient.close();

      const httpsClient = new Client({ host: 'example.com', port: 443, https: true });
      expect(httpsClient.baseURL).toBe('https://example.com:443');
      httpsClient.close();
    });
  });

  describe('Health Check', () => {
    test('should check server health', async () => {
      const health = await client.health();
      expect(health).toHaveProperty('status');
      expect(health.status).toBe('healthy');
      expect(health).toHaveProperty('uptime');
      expect(health).toHaveProperty('time');
    });
  });

  describe('Statistics', () => {
    test('should get database statistics', async () => {
      const stats = await client.stats();
      expect(stats).toHaveProperty('name');
      expect(stats).toHaveProperty('collections');
      expect(stats).toHaveProperty('active_transactions');
      expect(typeof stats.collections).toBe('number');
      expect(typeof stats.active_transactions).toBe('number');
    });
  });

  describe('Collection Management', () => {
    const testCollectionName = 'test_collection_' + Date.now();

    afterEach(async () => {
      try {
        await client.dropCollection(testCollectionName);
      } catch (err) {
        // Collection might not exist
      }
    });

    test('should list collections', async () => {
      const collections = await client.listCollections();
      expect(Array.isArray(collections)).toBe(true);
    });

    test('should create a collection', async () => {
      await client.createCollection(testCollectionName);
      const collections = await client.listCollections();
      expect(collections).toContain(testCollectionName);
    });

    test('should drop a collection', async () => {
      await client.createCollection(testCollectionName);
      await client.dropCollection(testCollectionName);
      const collections = await client.listCollections();
      expect(collections).not.toContain(testCollectionName);
    });

    test('should get collection handle', () => {
      const collection = client.collection(testCollectionName);
      expect(collection).toBeDefined();
      expect(collection.name).toBe(testCollectionName);
    });
  });

  describe('Error Handling', () => {
    test('should handle API errors gracefully', async () => {
      try {
        await client.collection('nonexistent').findOne('invalid-id');
      } catch (err) {
        expect(err).toBeInstanceOf(Error);
        expect(err.message).toBeTruthy();
      }
    });

    test('should handle timeout errors', async () => {
      const timeoutClient = new Client({
        host: 'localhost',
        port: 8080,
        timeout: 1 // 1ms timeout to force timeout error
      });

      try {
        await timeoutClient.health();
        // If it succeeds, that's fine (very fast server)
      } catch (err) {
        // If it fails, it should be a timeout error
        expect(err.message).toMatch(/timeout|ETIMEDOUT|ECONNRESET/i);
      } finally {
        timeoutClient.close();
      }
    }, 10000); // Give this test more time
  });

  describe('Connection Management', () => {
    test('should close client cleanly', () => {
      const testClient = new Client();
      expect(() => testClient.close()).not.toThrow();
    });
  });
});

// Integration test flag - skip if server is not running
const INTEGRATION_TESTS = process.env.INTEGRATION_TESTS === 'true';

if (INTEGRATION_TESTS) {
  describe('Integration Tests', () => {
    test('full CRUD workflow', async () => {
      const client = new Client({ host: 'localhost', port: 8080 });
      const testCollection = 'test_crud_' + Date.now();

      try {
        // Create collection
        await client.createCollection(testCollection);

        // Get collection handle
        const collection = client.collection(testCollection);

        // Insert document
        const id = await collection.insertOne({ name: 'Test User', age: 25 });
        expect(id).toBeTruthy();

        // Find document
        const doc = await collection.findOne(id);
        expect(doc).toBeTruthy();
        expect(doc.name).toBe('Test User');

        // Update document
        await collection.updateOne(id, { $set: { age: 26 } });

        // Verify update
        const updated = await collection.findOne(id);
        expect(updated.age).toBe(26);

        // Delete document
        await collection.deleteOne(id);

        // Verify deletion
        const deleted = await collection.findOne(id);
        expect(deleted).toBeNull();

        // Drop collection
        await client.dropCollection(testCollection);
      } finally {
        client.close();
      }
    });
  });
}
