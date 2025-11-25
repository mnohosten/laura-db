const http = require('http');
const https = require('https');
const { URL } = require('url');
const Collection = require('./collection');

/**
 * Client configuration options
 * @typedef {Object} ClientConfig
 * @property {string} [host='localhost'] - Server hostname or IP address
 * @property {number} [port=8080] - Server port
 * @property {boolean} [https=false] - Use HTTPS instead of HTTP
 * @property {number} [timeout=30000] - Request timeout in milliseconds
 * @property {number} [maxSockets=10] - Maximum number of sockets to keep open
 */

/**
 * LauraDB Client - Main entry point for interacting with LauraDB server
 * @class
 */
class Client {
  /**
   * Create a new LauraDB client
   * @param {ClientConfig} config - Client configuration
   */
  constructor(config = {}) {
    this.config = {
      host: config.host || 'localhost',
      port: config.port || 8080,
      https: config.https || false,
      timeout: config.timeout || 30000,
      maxSockets: config.maxSockets || 10
    };

    this.baseURL = `${this.config.https ? 'https' : 'http'}://${this.config.host}:${this.config.port}`;

    // Create HTTP agent for connection pooling
    const AgentClass = this.config.https ? https.Agent : http.Agent;
    this.agent = new AgentClass({
      keepAlive: true,
      maxSockets: this.config.maxSockets,
      maxFreeSockets: this.config.maxSockets,
      timeout: this.config.timeout
    });
  }

  /**
   * Perform an HTTP request to the LauraDB server
   * @private
   * @param {string} method - HTTP method (GET, POST, PUT, DELETE)
   * @param {string} path - Request path
   * @param {Object} [body] - Request body (will be JSON encoded)
   * @returns {Promise<Object>} API response
   */
  async _request(method, path, body = null) {
    return new Promise((resolve, reject) => {
      const url = new URL(path, this.baseURL);

      const options = {
        method,
        hostname: url.hostname,
        port: url.port,
        path: url.pathname + url.search,
        headers: {
          'Accept': 'application/json',
          'User-Agent': 'LauraDB-NodeJS-Client/1.0.0'
        },
        agent: this.agent,
        timeout: this.config.timeout
      };

      if (body !== null) {
        const bodyData = JSON.stringify(body);
        options.headers['Content-Type'] = 'application/json';
        options.headers['Content-Length'] = Buffer.byteLength(bodyData);
      }

      const client = this.config.https ? https : http;
      const req = client.request(options, (res) => {
        let data = '';

        res.on('data', (chunk) => {
          data += chunk;
        });

        res.on('end', () => {
          try {
            const response = JSON.parse(data);

            if (!response.ok) {
              const error = new Error(response.message || response.error || 'API request failed');
              error.code = response.code;
              error.apiError = response.error;
              error.response = response;
              reject(error);
              return;
            }

            resolve(response);
          } catch (err) {
            reject(new Error(`Failed to parse response: ${err.message}`));
          }
        });
      });

      req.on('error', (err) => {
        reject(new Error(`Request failed: ${err.message}`));
      });

      req.on('timeout', () => {
        req.destroy();
        reject(new Error(`Request timeout after ${this.config.timeout}ms`));
      });

      if (body !== null) {
        req.write(JSON.stringify(body));
      }

      req.end();
    });
  }

  /**
   * Check server health
   * @returns {Promise<Object>} Health status
   * @example
   * const health = await client.health();
   * console.log(health.status); // 'healthy'
   */
  async health() {
    const response = await this._request('GET', '/_health');
    return response.result;
  }

  /**
   * Get database statistics
   * @returns {Promise<Object>} Database statistics
   * @example
   * const stats = await client.stats();
   * console.log(stats.collections); // Number of collections
   */
  async stats() {
    const response = await this._request('GET', '/_stats');
    return response.result;
  }

  /**
   * List all collections
   * @returns {Promise<string[]>} Array of collection names
   * @example
   * const collections = await client.listCollections();
   * console.log(collections); // ['users', 'products']
   */
  async listCollections() {
    const response = await this._request('GET', '/_collections');
    return response.result.collections;
  }

  /**
   * Get a collection handle
   * @param {string} name - Collection name
   * @returns {Collection} Collection instance
   * @example
   * const users = client.collection('users');
   */
  collection(name) {
    return new Collection(this, name);
  }

  /**
   * Create a new collection
   * @param {string} name - Collection name
   * @returns {Promise<void>}
   * @example
   * await client.createCollection('users');
   */
  async createCollection(name) {
    await this._request('PUT', `/${encodeURIComponent(name)}`);
  }

  /**
   * Drop a collection
   * @param {string} name - Collection name
   * @returns {Promise<void>}
   * @example
   * await client.dropCollection('users');
   */
  async dropCollection(name) {
    await this._request('DELETE', `/${encodeURIComponent(name)}`);
  }

  /**
   * Close the client and clean up resources
   * @example
   * await client.close();
   */
  close() {
    if (this.agent) {
      this.agent.destroy();
    }
  }
}

module.exports = Client;
