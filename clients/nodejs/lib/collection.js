const Query = require('./query');
const Aggregation = require('./aggregation');
const Index = require('./index');

/**
 * Collection - Represents a database collection with CRUD operations
 * @class
 */
class Collection {
  /**
   * Create a Collection instance
   * @param {Client} client - LauraDB client instance
   * @param {string} name - Collection name
   */
  constructor(client, name) {
    this._client = client;
    this._name = name;
  }

  /**
   * Get the collection name
   * @returns {string} Collection name
   */
  get name() {
    return this._name;
  }

  /**
   * Insert a single document with auto-generated ID
   * @param {Object} doc - Document to insert
   * @returns {Promise<string>} Inserted document ID
   * @example
   * const id = await collection.insertOne({ name: 'Alice', age: 30 });
   */
  async insertOne(doc) {
    const path = `/${encodeURIComponent(this._name)}/_doc`;
    const response = await this._client._request('POST', path, doc);
    return response.result._id;
  }

  /**
   * Insert a document with a specific ID
   * @param {string} id - Document ID
   * @param {Object} doc - Document to insert
   * @returns {Promise<void>}
   * @example
   * await collection.insertOneWithID('user1', { name: 'Alice', age: 30 });
   */
  async insertOneWithID(id, doc) {
    const path = `/${encodeURIComponent(this._name)}/_doc/${encodeURIComponent(id)}`;
    await this._client._request('POST', path, doc);
  }

  /**
   * Insert multiple documents
   * @param {Object[]} docs - Array of documents to insert
   * @returns {Promise<string[]>} Array of inserted document IDs
   * @example
   * const ids = await collection.insertMany([
   *   { name: 'Alice', age: 30 },
   *   { name: 'Bob', age: 25 }
   * ]);
   */
  async insertMany(docs) {
    const path = `/${encodeURIComponent(this._name)}/_bulk`;
    const operations = docs.map(doc => ({
      operation: 'insert',
      document: doc
    }));

    const response = await this._client._request('POST', path, { operations });
    return response.result.ids || [];
  }

  /**
   * Find a single document by ID
   * @param {string} id - Document ID
   * @returns {Promise<Object|null>} Document or null if not found
   * @example
   * const user = await collection.findOne('user1');
   */
  async findOne(id) {
    try {
      const path = `/${encodeURIComponent(this._name)}/_doc/${encodeURIComponent(id)}`;
      const response = await this._client._request('GET', path);
      return response.result;
    } catch (err) {
      if (err.code === 404) {
        return null;
      }
      throw err;
    }
  }

  /**
   * Update a single document by ID
   * @param {string} id - Document ID
   * @param {Object} update - Update operations (e.g., { $set: { age: 31 } })
   * @returns {Promise<void>}
   * @example
   * await collection.updateOne('user1', { $set: { age: 31 } });
   */
  async updateOne(id, update) {
    const path = `/${encodeURIComponent(this._name)}/_doc/${encodeURIComponent(id)}`;
    await this._client._request('PUT', path, update);
  }

  /**
   * Delete a single document by ID
   * @param {string} id - Document ID
   * @returns {Promise<void>}
   * @example
   * await collection.deleteOne('user1');
   */
  async deleteOne(id) {
    const path = `/${encodeURIComponent(this._name)}/_doc/${encodeURIComponent(id)}`;
    await this._client._request('DELETE', path);
  }

  /**
   * Create a query builder for finding documents
   * @returns {Query} Query builder
   * @example
   * const results = await collection.find()
   *   .filter({ age: { $gt: 25 } })
   *   .sort({ name: 1 })
   *   .limit(10)
   *   .execute();
   */
  find() {
    return new Query(this._client, this._name);
  }

  /**
   * Count documents matching a filter
   * @param {Object} [filter={}] - Query filter
   * @returns {Promise<number>} Count of matching documents
   * @example
   * const count = await collection.count({ age: { $gt: 25 } });
   */
  async count(filter = {}) {
    const path = `/${encodeURIComponent(this._name)}/_count`;
    const response = await this._client._request('POST', path, { filter });
    return response.count || 0;
  }

  /**
   * Perform bulk operations
   * @param {Object[]} operations - Array of operations
   * @returns {Promise<Object>} Bulk operation result
   * @example
   * const result = await collection.bulk([
   *   { operation: 'insert', document: { name: 'Alice' } },
   *   { operation: 'update', _id: 'user1', update: { $set: { age: 31 } } },
   *   { operation: 'delete', _id: 'user2' }
   * ]);
   */
  async bulk(operations) {
    const path = `/${encodeURIComponent(this._name)}/_bulk`;
    const response = await this._client._request('POST', path, { operations });
    return response.result;
  }

  /**
   * Create an aggregation pipeline builder
   * @returns {Aggregation} Aggregation builder
   * @example
   * const results = await collection.aggregate()
   *   .match({ age: { $gt: 25 } })
   *   .group({ _id: '$city', avgAge: { $avg: '$age' } })
   *   .sort({ avgAge: -1 })
   *   .execute();
   */
  aggregate() {
    return new Aggregation(this._client, this._name);
  }

  /**
   * Get index manager for this collection
   * @returns {Index} Index manager
   * @example
   * await collection.indexes().create('age', { unique: false });
   */
  indexes() {
    return new Index(this._client, this._name);
  }

  /**
   * Get collection statistics
   * @returns {Promise<Object>} Collection statistics
   * @example
   * const stats = await collection.stats();
   * console.log(stats.count); // Number of documents
   */
  async stats() {
    const path = `/${encodeURIComponent(this._name)}/_stats`;
    const response = await this._client._request('GET', path);
    return response.result;
  }

  /**
   * Drop the collection
   * @returns {Promise<void>}
   * @example
   * await collection.drop();
   */
  async drop() {
    await this._client.dropCollection(this._name);
  }
}

module.exports = Collection;
