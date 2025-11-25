/**
 * Aggregation - Builder for creating and executing aggregation pipelines
 * @class
 */
class Aggregation {
  /**
   * Create an Aggregation builder
   * @param {Client} client - LauraDB client instance
   * @param {string} collectionName - Collection name
   */
  constructor(client, collectionName) {
    this._client = client;
    this._collectionName = collectionName;
    this._pipeline = [];
  }

  /**
   * Add a $match stage to filter documents
   * @param {Object} filter - Match filter
   * @returns {Aggregation} This aggregation builder for chaining
   * @example
   * aggregation.match({ age: { $gt: 25 } })
   */
  match(filter) {
    this._pipeline.push({ $match: filter });
    return this;
  }

  /**
   * Add a $group stage to group documents
   * @param {Object} groupBy - Group specification
   * @returns {Aggregation} This aggregation builder for chaining
   * @example
   * aggregation.group({
   *   _id: '$city',
   *   avgAge: { $avg: '$age' },
   *   count: { $sum: 1 }
   * })
   */
  group(groupBy) {
    this._pipeline.push({ $group: groupBy });
    return this;
  }

  /**
   * Add a $project stage to select/transform fields
   * @param {Object} projection - Project specification
   * @returns {Aggregation} This aggregation builder for chaining
   * @example
   * aggregation.project({ name: 1, age: 1, email: 1 })
   */
  project(projection) {
    this._pipeline.push({ $project: projection });
    return this;
  }

  /**
   * Add a $sort stage to order results
   * @param {Object} sortBy - Sort specification
   * @returns {Aggregation} This aggregation builder for chaining
   * @example
   * aggregation.sort({ age: -1, name: 1 })
   */
  sort(sortBy) {
    this._pipeline.push({ $sort: sortBy });
    return this;
  }

  /**
   * Add a $limit stage to limit results
   * @param {number} limit - Maximum number of documents
   * @returns {Aggregation} This aggregation builder for chaining
   * @example
   * aggregation.limit(10)
   */
  limit(limit) {
    this._pipeline.push({ $limit: limit });
    return this;
  }

  /**
   * Add a $skip stage to skip documents
   * @param {number} skip - Number of documents to skip
   * @returns {Aggregation} This aggregation builder for chaining
   * @example
   * aggregation.skip(20)
   */
  skip(skip) {
    this._pipeline.push({ $skip: skip });
    return this;
  }

  /**
   * Execute the aggregation pipeline
   * @returns {Promise<Object[]>} Array of aggregation results
   * @example
   * const results = await collection.aggregate()
   *   .match({ age: { $gt: 25 } })
   *   .group({ _id: '$city', avgAge: { $avg: '$age' } })
   *   .sort({ avgAge: -1 })
   *   .execute();
   */
  async execute() {
    const path = `/${encodeURIComponent(this._collectionName)}/_aggregate`;
    const body = { pipeline: this._pipeline };

    const response = await this._client._request('POST', path, body);
    return response.result;
  }

  /**
   * Get the pipeline stages
   * @returns {Object[]} Array of pipeline stages
   */
  getPipeline() {
    return this._pipeline;
  }
}

module.exports = Aggregation;
