const Client = require('./client');
const Collection = require('./collection');
const Query = require('./query');
const Aggregation = require('./aggregation');
const Index = require('./index');

/**
 * Create a new LauraDB client with default configuration
 * @param {Object} config - Client configuration
 * @returns {Client} LauraDB client instance
 * @example
 * const lauradb = require('lauradb-client');
 * const client = lauradb.createClient({ host: 'localhost', port: 8080 });
 */
function createClient(config) {
  return new Client(config);
}

// Export main client class and helper function
module.exports = {
  Client,
  Collection,
  Query,
  Aggregation,
  Index,
  createClient
};

// Default export for ES6 style: import lauradb from 'lauradb-client'
module.exports.default = {
  Client,
  createClient
};
