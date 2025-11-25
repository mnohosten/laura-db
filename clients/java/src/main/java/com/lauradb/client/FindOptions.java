package com.lauradb.client;

import java.util.HashMap;
import java.util.Map;

/**
 * Options for find operations.
 *
 * <p>Example usage:
 * <pre>{@code
 * FindOptions options = new FindOptions()
 *     .projection(Map.of("name", 1, "email", 1, "_id", 0))
 *     .sort(Map.of("age", -1, "name", 1))
 *     .limit(10)
 *     .skip(20);
 * }</pre>
 */
public class FindOptions {
    private Map<String, Object> projection;
    private Map<String, Integer> sort;
    private int limit;
    private int skip;

    public FindOptions() {
        this.projection = null;
        this.sort = null;
        this.limit = 0;
        this.skip = 0;
    }

    /**
     * Set field projection.
     *
     * @param projection map of field names to 1 (include) or 0 (exclude)
     * @return this options instance
     */
    public FindOptions projection(Map<String, Object> projection) {
        this.projection = projection;
        return this;
    }

    /**
     * Set sort order.
     *
     * @param sort map of field names to 1 (ascending) or -1 (descending)
     * @return this options instance
     */
    public FindOptions sort(Map<String, Integer> sort) {
        this.sort = sort;
        return this;
    }

    /**
     * Set limit.
     *
     * @param limit maximum number of documents to return
     * @return this options instance
     */
    public FindOptions limit(int limit) {
        this.limit = limit;
        return this;
    }

    /**
     * Set skip.
     *
     * @param skip number of documents to skip
     * @return this options instance
     */
    public FindOptions skip(int skip) {
        this.skip = skip;
        return this;
    }

    public Map<String, Object> getProjection() {
        return projection;
    }

    public Map<String, Integer> getSort() {
        return sort;
    }

    public int getLimit() {
        return limit;
    }

    public int getSkip() {
        return skip;
    }

    /**
     * Create a builder-style projection map.
     *
     * @return a new ProjectionBuilder
     */
    public static ProjectionBuilder projectionBuilder() {
        return new ProjectionBuilder();
    }

    /**
     * Create a builder-style sort map.
     *
     * @return a new SortBuilder
     */
    public static SortBuilder sortBuilder() {
        return new SortBuilder();
    }

    public static class ProjectionBuilder {
        private final Map<String, Object> projection = new HashMap<>();

        public ProjectionBuilder include(String field) {
            projection.put(field, 1);
            return this;
        }

        public ProjectionBuilder exclude(String field) {
            projection.put(field, 0);
            return this;
        }

        public Map<String, Object> build() {
            return projection;
        }
    }

    public static class SortBuilder {
        private final Map<String, Integer> sort = new HashMap<>();

        public SortBuilder ascending(String field) {
            sort.put(field, 1);
            return this;
        }

        public SortBuilder descending(String field) {
            sort.put(field, -1);
            return this;
        }

        public Map<String, Integer> build() {
            return sort;
        }
    }
}
