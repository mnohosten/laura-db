/**
 * Query - Builder for creating and executing queries
 * @class
 */
class Query {
  /**
   * Create a Query builder
   * @param {Client} client - LauraDB client instance
   * @param {string} collectionName - Collection name
   */
  constructor(client, collectionName) {
    this._client = client;
    this._collectionName = collectionName;
    this._filter = {};
    this._projection = null;
    this._sortBy = null;
    this._limitNum = null;
    this._skipNum = null;
  }

  /**
   * Set query filter
   * @param {Object} filter - Query filter (e.g., { age: { $gt: 25 } })
   * @returns {Query} This query builder for chaining
   * @example
   * query.filter({ age: { $gt: 25 }, city: 'New York' })
   */
  filter(filter) {
    this._filter = filter;
    return this;
  }

  /**
   * Set field projection
   * @param {Object} projection - Field projection (e.g., { name: 1, age: 1 })
   * @returns {Query} This query builder for chaining
   * @example
   * query.project({ name: 1, age: 1 }) // Include only name and age
   */
  project(projection) {
    this._projection = projection;
    return this;
  }

  /**
   * Set sort order
   * @param {Object} sortBy - Sort specification (e.g., { age: -1, name: 1 })
   * @returns {Query} This query builder for chaining
   * @example
   * query.sort({ age: -1, name: 1 }) // Sort by age descending, then name ascending
   */
  sort(sortBy) {
    this._sortBy = sortBy;
    return this;
  }

  /**
   * Set maximum number of results
   * @param {number} limit - Maximum number of documents to return
   * @returns {Query} This query builder for chaining
   * @example
   * query.limit(10)
   */
  limit(limit) {
    this._limitNum = limit;
    return this;
  }

  /**
   * Set number of results to skip
   * @param {number} skip - Number of documents to skip
   * @returns {Query} This query builder for chaining
   * @example
   * query.skip(20)
   */
  skip(skip) {
    this._skipNum = skip;
    return this;
  }

  /**
   * Execute the query and return results
   * @returns {Promise<Object[]>} Array of matching documents
   * @example
   * const results = await query
   *   .filter({ age: { $gt: 25 } })
   *   .sort({ name: 1 })
   *   .limit(10)
   *   .execute();
   */
  async execute() {
    const path = `/${encodeURIComponent(this._collectionName)}/_query`;

    const body = {
      filter: this._filter
    };

    if (this._projection !== null) {
      body.projection = this._projection;
    }

    if (this._sortBy !== null) {
      body.sort = this._sortBy;
    }

    if (this._limitNum !== null) {
      body.limit = this._limitNum;
    }

    if (this._skipNum !== null) {
      body.skip = this._skipNum;
    }

    const response = await this._client._request('POST', path, body);
    return response.result;
  }

  /**
   * Execute the query and return the first result
   * @returns {Promise<Object|null>} First matching document or null
   * @example
   * const user = await query.filter({ email: 'alice@example.com' }).first();
   */
  async first() {
    const results = await this.limit(1).execute();
    return results.length > 0 ? results[0] : null;
  }

  /**
   * Count documents matching the query filter
   * @returns {Promise<number>} Count of matching documents
   * @example
   * const count = await query.filter({ age: { $gt: 25 } }).count();
   */
  async count() {
    const path = `/${encodeURIComponent(this._collectionName)}/_count`;
    const response = await this._client._request('POST', path, { filter: this._filter });
    return response.count || 0;
  }

  /**
   * Update all documents matching the query filter
   * @param {Object} update - Update operations
   * @returns {Promise<number>} Number of documents updated
   * @example
   * const updated = await query
   *   .filter({ age: { $lt: 18 } })
   *   .update({ $set: { minor: true } });
   */
  async update(update) {
    const path = `/${encodeURIComponent(this._collectionName)}/_update`;
    const body = {
      filter: this._filter,
      update: update
    };

    const response = await this._client._request('POST', path, body);
    return response.count || 0;
  }

  /**
   * Delete all documents matching the query filter
   * @returns {Promise<number>} Number of documents deleted
   * @example
   * const deleted = await query
   *   .filter({ inactive: true })
   *   .delete();
   */
  async delete() {
    const path = `/${encodeURIComponent(this._collectionName)}/_delete`;
    const response = await this._client._request('POST', path, { filter: this._filter });
    return response.count || 0;
  }

  /**
   * Get query execution plan (for optimization)
   * @returns {Promise<Object>} Query execution plan
   * @example
   * const plan = await query.filter({ age: { $gt: 25 } }).explain();
   * console.log(plan.index_used); // Which index was selected
   */
  async explain() {
    const path = `/${encodeURIComponent(this._collectionName)}/_explain`;
    const body = {
      filter: this._filter
    };

    if (this._sortBy !== null) {
      body.sort = this._sortBy;
    }

    const response = await this._client._request('POST', path, body);
    return response.result;
  }
}

module.exports = Query;
