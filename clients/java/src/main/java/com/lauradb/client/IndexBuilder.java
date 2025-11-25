package com.lauradb.client;

import java.io.IOException;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Builder for creating indexes on collections.
 *
 * <p>Example usage:
 * <pre>{@code
 * // Simple B+ tree index
 * collection.createIndex("email")
 *     .unique(true)
 *     .build();
 *
 * // Compound index
 * collection.createCompoundIndex(List.of("city", "age"))
 *     .name("city_age_idx")
 *     .build();
 *
 * // Text index
 * collection.createIndex("content")
 *     .text()
 *     .build();
 *
 * // Geospatial index
 * collection.createIndex("location")
 *     .geo("2dsphere")
 *     .build();
 *
 * // TTL index
 * collection.createIndex("createdAt")
 *     .ttl(3600)
 *     .build();
 *
 * // Partial index
 * collection.createIndex("email")
 *     .unique(true)
 *     .partial(Query.builder().eq("active", true).build())
 *     .build();
 * }</pre>
 */
public class IndexBuilder {
    private final Collection collection;
    private final Object field; // String or List<String> for compound
    private String name;
    private boolean unique;
    private String indexType; // "btree", "text", "geo"
    private String geoType; // "2d", "2dsphere"
    private Integer ttlSeconds;
    private Query partialFilter;
    private boolean background;

    IndexBuilder(Collection collection, String field) {
        this.collection = collection;
        this.field = field;
        this.indexType = "btree";
    }

    IndexBuilder(Collection collection, List<String> fields) {
        this.collection = collection;
        this.field = fields;
        this.indexType = "compound";
    }

    /**
     * Set the index name.
     *
     * @param name the index name
     * @return this builder
     */
    public IndexBuilder name(String name) {
        this.name = name;
        return this;
    }

    /**
     * Make this a unique index.
     *
     * @param unique whether the index should enforce uniqueness
     * @return this builder
     */
    public IndexBuilder unique(boolean unique) {
        this.unique = unique;
        return this;
    }

    /**
     * Make this a text index for full-text search.
     *
     * @return this builder
     */
    public IndexBuilder text() {
        this.indexType = "text";
        return this;
    }

    /**
     * Make this a geospatial index.
     *
     * @param geoType the geo index type ("2d" or "2dsphere")
     * @return this builder
     */
    public IndexBuilder geo(String geoType) {
        this.indexType = "geo";
        this.geoType = geoType;
        return this;
    }

    /**
     * Make this a TTL index for automatic document expiration.
     *
     * @param seconds time-to-live in seconds
     * @return this builder
     */
    public IndexBuilder ttl(int seconds) {
        this.ttlSeconds = seconds;
        return this;
    }

    /**
     * Make this a partial index (only indexes documents matching the filter).
     *
     * @param filter the filter query for partial indexing
     * @return this builder
     */
    public IndexBuilder partial(Query filter) {
        this.partialFilter = filter;
        return this;
    }

    /**
     * Build the index in the background (non-blocking).
     *
     * @param background whether to build in background
     * @return this builder
     */
    public IndexBuilder background(boolean background) {
        this.background = background;
        return this;
    }

    /**
     * Create the index.
     *
     * @throws IOException if the request fails
     */
    public void build() throws IOException {
        Map<String, Object> body = new HashMap<>();

        if (field instanceof String) {
            body.put("field", field);
        } else if (field instanceof List) {
            body.put("fields", field);
        }

        if (name != null) {
            body.put("name", name);
        }

        if (unique) {
            body.put("unique", true);
        }

        if (background) {
            body.put("background", true);
        }

        // Index type specific options
        switch (indexType) {
            case "text":
                body.put("type", "text");
                break;
            case "geo":
                body.put("type", "geo");
                if (geoType != null) {
                    body.put("geoType", geoType);
                }
                break;
            case "compound":
                body.put("type", "compound");
                break;
            default:
                body.put("type", "btree");
                break;
        }

        // TTL index
        if (ttlSeconds != null) {
            body.put("ttl", ttlSeconds);
        }

        // Partial index
        if (partialFilter != null) {
            body.put("partialFilter", partialFilter.getFilter());
        }

        collection.client.request("POST", "/" + collection.getName() + "/_index", body);
    }
}
