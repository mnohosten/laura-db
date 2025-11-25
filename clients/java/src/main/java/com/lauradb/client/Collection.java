package com.lauradb.client;

import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;

import java.io.IOException;
import java.util.*;
import java.util.concurrent.CompletableFuture;

/**
 * Represents a collection in LauraDB.
 *
 * <p>Example usage:
 * <pre>{@code
 * Collection users = client.collection("users");
 *
 * // Insert a document
 * Map<String, Object> doc = new HashMap<>();
 * doc.put("name", "Alice");
 * doc.put("age", 30);
 * String id = users.insertOne(doc);
 *
 * // Find documents
 * List<Map<String, Object>> results = users.find(
 *     Query.builder().gte("age", 25).build()
 * );
 * }</pre>
 */
public class Collection {
    private final LauraDBClient client;
    private final String name;

    Collection(LauraDBClient client, String name) {
        this.client = client;
        this.name = name;
    }

    /**
     * Get the collection name.
     *
     * @return the collection name
     */
    public String getName() {
        return name;
    }

    /**
     * Insert a single document.
     *
     * @param document the document to insert
     * @return the inserted document's ID
     * @throws IOException if the request fails
     */
    public String insertOne(Map<String, Object> document) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("document", document);

        JsonObject response = client.request("POST", "/" + name + "/_doc", body);
        return response.get("id").getAsString();
    }

    /**
     * Insert multiple documents.
     *
     * @param documents the documents to insert
     * @return list of inserted document IDs
     * @throws IOException if the request fails
     */
    @SuppressWarnings("unchecked")
    public List<String> insertMany(List<Map<String, Object>> documents) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("documents", documents);

        JsonObject response = client.request("POST", "/" + name + "/_bulk", body);
        return client.getGson().fromJson(response.get("ids"), List.class);
    }

    /**
     * Find a single document by ID.
     *
     * @param id the document ID
     * @return the document, or null if not found
     * @throws IOException if the request fails
     */
    @SuppressWarnings("unchecked")
    public Map<String, Object> findById(String id) throws IOException {
        try {
            JsonObject response = client.request("GET", "/" + name + "/_doc/" + id, null);
            return client.getGson().fromJson(response.get("document"), Map.class);
        } catch (IOException e) {
            if (e.getMessage().contains("404") || e.getMessage().contains("not found")) {
                return null;
            }
            throw e;
        }
    }

    /**
     * Find documents matching a query.
     *
     * @param query the query filter
     * @return list of matching documents
     * @throws IOException if the request fails
     */
    public List<Map<String, Object>> find(Query query) throws IOException {
        return find(query, new FindOptions());
    }

    /**
     * Find documents with options.
     *
     * @param query the query filter
     * @param options find options (projection, sort, limit, skip)
     * @return list of matching documents
     * @throws IOException if the request fails
     */
    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> find(Query query, FindOptions options) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("filter", query.getFilter());

        if (options.getProjection() != null) {
            body.put("projection", options.getProjection());
        }
        if (options.getSort() != null) {
            body.put("sort", options.getSort());
        }
        if (options.getLimit() > 0) {
            body.put("limit", options.getLimit());
        }
        if (options.getSkip() > 0) {
            body.put("skip", options.getSkip());
        }

        JsonObject response = client.request("POST", "/" + name + "/_query", body);
        JsonArray documents = response.getAsJsonArray("documents");

        List<Map<String, Object>> results = new ArrayList<>();
        for (JsonElement doc : documents) {
            results.add(client.getGson().fromJson(doc, Map.class));
        }
        return results;
    }

    /**
     * Find all documents in the collection.
     *
     * @return list of all documents
     * @throws IOException if the request fails
     */
    public List<Map<String, Object>> findAll() throws IOException {
        return find(Query.empty());
    }

    /**
     * Find one document matching a query.
     *
     * @param query the query filter
     * @return the first matching document, or null if none found
     * @throws IOException if the request fails
     */
    public Map<String, Object> findOne(Query query) throws IOException {
        FindOptions options = new FindOptions().limit(1);
        List<Map<String, Object>> results = find(query, options);
        return results.isEmpty() ? null : results.get(0);
    }

    /**
     * Count documents matching a query.
     *
     * @param query the query filter
     * @return the count of matching documents
     * @throws IOException if the request fails
     */
    public long count(Query query) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("filter", query.getFilter());

        JsonObject response = client.request("POST", "/" + name + "/_count", body);
        return response.get("count").getAsLong();
    }

    /**
     * Update a single document.
     *
     * @param query the query filter to find the document
     * @param update the update operations
     * @return the number of modified documents
     * @throws IOException if the request fails
     */
    public long updateOne(Query query, Map<String, Object> update) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("filter", query.getFilter());
        body.put("update", update);

        JsonObject response = client.request("POST", "/" + name + "/_update", body);
        return response.get("modified").getAsLong();
    }

    /**
     * Update a document by ID.
     *
     * @param id the document ID
     * @param update the update operations
     * @return the number of modified documents
     * @throws IOException if the request fails
     */
    public long updateById(String id, Map<String, Object> update) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("update", update);

        JsonObject response = client.request("PUT", "/" + name + "/_doc/" + id, body);
        return response.get("modified").getAsLong();
    }

    /**
     * Update multiple documents.
     *
     * @param query the query filter
     * @param update the update operations
     * @return the number of modified documents
     * @throws IOException if the request fails
     */
    public long updateMany(Query query, Map<String, Object> update) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("filter", query.getFilter());
        body.put("update", update);
        body.put("multi", true);

        JsonObject response = client.request("POST", "/" + name + "/_update", body);
        return response.get("modified").getAsLong();
    }

    /**
     * Delete a single document.
     *
     * @param query the query filter
     * @return the number of deleted documents
     * @throws IOException if the request fails
     */
    public long deleteOne(Query query) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("filter", query.getFilter());

        JsonObject response = client.request("POST", "/" + name + "/_delete", body);
        return response.get("deleted").getAsLong();
    }

    /**
     * Delete a document by ID.
     *
     * @param id the document ID
     * @return the number of deleted documents
     * @throws IOException if the request fails
     */
    public long deleteById(String id) throws IOException {
        JsonObject response = client.request("DELETE", "/" + name + "/_doc/" + id, null);
        return response.get("deleted").getAsLong();
    }

    /**
     * Delete multiple documents.
     *
     * @param query the query filter
     * @return the number of deleted documents
     * @throws IOException if the request fails
     */
    public long deleteMany(Query query) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("filter", query.getFilter());
        body.put("multi", true);

        JsonObject response = client.request("POST", "/" + name + "/_delete", body);
        return response.get("deleted").getAsLong();
    }

    /**
     * Execute an aggregation pipeline.
     *
     * @param pipeline the aggregation pipeline
     * @return list of aggregation results
     * @throws IOException if the request fails
     */
    @SuppressWarnings("unchecked")
    public List<Map<String, Object>> aggregate(Aggregation pipeline) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("pipeline", pipeline.getPipeline());

        JsonObject response = client.request("POST", "/" + name + "/_aggregate", body);
        JsonArray results = response.getAsJsonArray("results");

        List<Map<String, Object>> output = new ArrayList<>();
        for (JsonElement result : results) {
            output.add(client.getGson().fromJson(result, Map.class));
        }
        return output;
    }

    /**
     * Create an index on a field.
     *
     * @param field the field to index
     * @return an IndexBuilder for further configuration
     */
    public IndexBuilder createIndex(String field) {
        return new IndexBuilder(this, field);
    }

    /**
     * Create a compound index on multiple fields.
     *
     * @param fields the fields to index
     * @return an IndexBuilder for further configuration
     */
    public IndexBuilder createCompoundIndex(List<String> fields) {
        return new IndexBuilder(this, fields);
    }

    /**
     * List all indexes in the collection.
     *
     * @return list of index names
     * @throws IOException if the request fails
     */
    @SuppressWarnings("unchecked")
    public List<String> listIndexes() throws IOException {
        JsonObject response = client.request("GET", "/" + name + "/_index", null);
        return client.getGson().fromJson(response.get("indexes"), List.class);
    }

    /**
     * Drop an index.
     *
     * @param field the indexed field
     * @throws IOException if the request fails
     */
    public void dropIndex(String field) throws IOException {
        client.request("DELETE", "/" + name + "/_index/" + field, null);
    }

    /**
     * Asynchronously insert a document.
     *
     * @param document the document to insert
     * @return a CompletableFuture with the document ID
     */
    public CompletableFuture<String> insertOneAsync(Map<String, Object> document) {
        return CompletableFuture.supplyAsync(() -> {
            try {
                return insertOne(document);
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
        });
    }

    /**
     * Asynchronously find documents.
     *
     * @param query the query filter
     * @return a CompletableFuture with the results
     */
    public CompletableFuture<List<Map<String, Object>>> findAsync(Query query) {
        return CompletableFuture.supplyAsync(() -> {
            try {
                return find(query);
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
        });
    }
}
