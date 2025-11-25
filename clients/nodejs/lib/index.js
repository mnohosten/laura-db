/**
 * Index - Manager for collection indexes
 * @class
 */
class Index {
  /**
   * Create an Index manager
   * @param {Client} client - LauraDB client instance
   * @param {string} collectionName - Collection name
   */
  constructor(client, collectionName) {
    this._client = client;
    this._collectionName = collectionName;
  }

  /**
   * Create a B+ tree index on a single field
   * @param {string} field - Field name to index
   * @param {Object} [options={}] - Index options
   * @param {boolean} [options.unique=false] - Enforce uniqueness
   * @param {boolean} [options.sparse=false] - Only index documents with the field
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().create('email', { unique: true });
   */
  async create(field, options = {}) {
    const path = `/${encodeURIComponent(this._collectionName)}/_index`;
    const body = {
      field: field,
      unique: options.unique || false,
      sparse: options.sparse || false
    };

    await this._client._request('POST', path, body);
  }

  /**
   * Create a compound index on multiple fields
   * @param {Object} fields - Field specifications (e.g., { city: 1, age: -1 })
   * @param {Object} [options={}] - Index options
   * @param {boolean} [options.unique=false] - Enforce uniqueness
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().createCompound(
   *   { city: 1, age: -1 },
   *   { unique: false }
   * );
   */
  async createCompound(fields, options = {}) {
    const path = `/${encodeURIComponent(this._collectionName)}/_index/compound`;
    const body = {
      fields: fields,
      unique: options.unique || false
    };

    await this._client._request('POST', path, body);
  }

  /**
   * Create a text index for full-text search
   * @param {string[]} fields - Array of field names to index
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().createText(['title', 'description']);
   */
  async createText(fields) {
    const path = `/${encodeURIComponent(this._collectionName)}/_index/text`;
    const body = { fields: fields };

    await this._client._request('POST', path, body);
  }

  /**
   * Create a geospatial index
   * @param {string} field - Field containing coordinates
   * @param {string} [type='2d'] - Index type ('2d' or '2dsphere')
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().createGeo('location', '2dsphere');
   */
  async createGeo(field, type = '2d') {
    const path = `/${encodeURIComponent(this._collectionName)}/_index/geo`;
    const body = {
      field: field,
      type: type
    };

    await this._client._request('POST', path, body);
  }

  /**
   * Create a TTL (time-to-live) index for automatic document expiration
   * @param {string} field - Field containing timestamp
   * @param {number} expireAfterSeconds - Time in seconds after which documents expire
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().createTTL('createdAt', 86400); // Expire after 24 hours
   */
  async createTTL(field, expireAfterSeconds) {
    const path = `/${encodeURIComponent(this._collectionName)}/_index/ttl`;
    const body = {
      field: field,
      expire_after_seconds: expireAfterSeconds
    };

    await this._client._request('POST', path, body);
  }

  /**
   * Create a partial index with a filter
   * @param {string} field - Field name to index
   * @param {Object} filter - Filter expression for partial indexing
   * @param {Object} [options={}] - Index options
   * @param {boolean} [options.unique=false] - Enforce uniqueness
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().createPartial(
   *   'email',
   *   { active: true },
   *   { unique: true }
   * );
   */
  async createPartial(field, filter, options = {}) {
    const path = `/${encodeURIComponent(this._collectionName)}/_index/partial`;
    const body = {
      field: field,
      filter: filter,
      unique: options.unique || false
    };

    await this._client._request('POST', path, body);
  }

  /**
   * List all indexes on the collection
   * @returns {Promise<Object[]>} Array of index information
   * @example
   * const indexes = await collection.indexes().list();
   * console.log(indexes);
   */
  async list() {
    const path = `/${encodeURIComponent(this._collectionName)}/_index`;
    const response = await this._client._request('GET', path);
    return response.result.indexes || [];
  }

  /**
   * Drop an index
   * @param {string} field - Field name of the index to drop
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().drop('email');
   */
  async drop(field) {
    const path = `/${encodeURIComponent(this._collectionName)}/_index/${encodeURIComponent(field)}`;
    await this._client._request('DELETE', path);
  }

  /**
   * Drop a compound index
   * @param {string} name - Name of the compound index to drop
   * @returns {Promise<void>}
   * @example
   * await collection.indexes().dropCompound('city_1_age_1');
   */
  async dropCompound(name) {
    const path = `/${encodeURIComponent(this._collectionName)}/_index/compound/${encodeURIComponent(name)}`;
    await this._client._request('DELETE', path);
  }
}

module.exports = Index;
