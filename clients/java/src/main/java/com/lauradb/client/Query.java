package com.lauradb.client;

import java.util.*;

/**
 * Query builder for constructing MongoDB-style queries.
 *
 * <p>Example usage:
 * <pre>{@code
 * // Simple equality
 * Query query = Query.builder().eq("name", "Alice").build();
 *
 * // Range query
 * Query query = Query.builder()
 *     .gte("age", 25)
 *     .lt("age", 40)
 *     .build();
 *
 * // Logical operators
 * Query query = Query.builder()
 *     .and(
 *         Query.builder().gte("age", 25).build(),
 *         Query.builder().eq("active", true).build()
 *     )
 *     .build();
 * }</pre>
 */
public class Query {
    private final Map<String, Object> filter;

    private Query(Map<String, Object> filter) {
        this.filter = filter;
    }

    /**
     * Get the filter map.
     *
     * @return the filter map
     */
    public Map<String, Object> getFilter() {
        return filter;
    }

    /**
     * Create a new query builder.
     *
     * @return a new Builder instance
     */
    public static Builder builder() {
        return new Builder();
    }

    /**
     * Create an empty query (matches all documents).
     *
     * @return an empty query
     */
    public static Query empty() {
        return new Query(new HashMap<>());
    }

    /**
     * Builder for constructing queries.
     */
    public static class Builder {
        private final Map<String, Object> filter;

        private Builder() {
            this.filter = new HashMap<>();
        }

        /**
         * Add an equality condition.
         *
         * @param field the field name
         * @param value the value to match
         * @return this builder
         */
        public Builder eq(String field, Object value) {
            filter.put(field, value);
            return this;
        }

        /**
         * Add a not-equals condition.
         *
         * @param field the field name
         * @param value the value to not match
         * @return this builder
         */
        public Builder ne(String field, Object value) {
            return addOperator(field, "$ne", value);
        }

        /**
         * Add a greater-than condition.
         *
         * @param field the field name
         * @param value the value to compare
         * @return this builder
         */
        public Builder gt(String field, Object value) {
            return addOperator(field, "$gt", value);
        }

        /**
         * Add a greater-than-or-equal condition.
         *
         * @param field the field name
         * @param value the value to compare
         * @return this builder
         */
        public Builder gte(String field, Object value) {
            return addOperator(field, "$gte", value);
        }

        /**
         * Add a less-than condition.
         *
         * @param field the field name
         * @param value the value to compare
         * @return this builder
         */
        public Builder lt(String field, Object value) {
            return addOperator(field, "$lt", value);
        }

        /**
         * Add a less-than-or-equal condition.
         *
         * @param field the field name
         * @param value the value to compare
         * @return this builder
         */
        public Builder lte(String field, Object value) {
            return addOperator(field, "$lte", value);
        }

        /**
         * Add an in condition (value in array).
         *
         * @param field the field name
         * @param values the values to match
         * @return this builder
         */
        public Builder in(String field, Object... values) {
            return addOperator(field, "$in", Arrays.asList(values));
        }

        /**
         * Add a not-in condition (value not in array).
         *
         * @param field the field name
         * @param values the values to not match
         * @return this builder
         */
        public Builder nin(String field, Object... values) {
            return addOperator(field, "$nin", Arrays.asList(values));
        }

        /**
         * Add an exists condition.
         *
         * @param field the field name
         * @param exists whether the field should exist
         * @return this builder
         */
        public Builder exists(String field, boolean exists) {
            return addOperator(field, "$exists", exists);
        }

        /**
         * Add a type condition.
         *
         * @param field the field name
         * @param type the expected type ("string", "number", "boolean", "array", "document", "null")
         * @return this builder
         */
        public Builder type(String field, String type) {
            return addOperator(field, "$type", type);
        }

        /**
         * Add a regex condition.
         *
         * @param field the field name
         * @param pattern the regex pattern
         * @return this builder
         */
        public Builder regex(String field, String pattern) {
            return addOperator(field, "$regex", pattern);
        }

        /**
         * Add an all condition (array contains all values).
         *
         * @param field the field name
         * @param values the values that must all be present
         * @return this builder
         */
        public Builder all(String field, Object... values) {
            return addOperator(field, "$all", Arrays.asList(values));
        }

        /**
         * Add a size condition (array size).
         *
         * @param field the field name
         * @param size the expected array size
         * @return this builder
         */
        public Builder size(String field, int size) {
            return addOperator(field, "$size", size);
        }

        /**
         * Add an elemMatch condition (array element matching).
         *
         * @param field the field name
         * @param query the query for array elements
         * @return this builder
         */
        public Builder elemMatch(String field, Query query) {
            return addOperator(field, "$elemMatch", query.getFilter());
        }

        /**
         * Add an AND condition.
         *
         * @param queries the queries to AND together
         * @return this builder
         */
        public Builder and(Query... queries) {
            List<Map<String, Object>> conditions = new ArrayList<>();
            for (Query q : queries) {
                conditions.add(q.getFilter());
            }
            filter.put("$and", conditions);
            return this;
        }

        /**
         * Add an OR condition.
         *
         * @param queries the queries to OR together
         * @return this builder
         */
        public Builder or(Query... queries) {
            List<Map<String, Object>> conditions = new ArrayList<>();
            for (Query q : queries) {
                conditions.add(q.getFilter());
            }
            filter.put("$or", conditions);
            return this;
        }

        /**
         * Add a NOT condition.
         *
         * @param query the query to negate
         * @return this builder
         */
        public Builder not(Query query) {
            filter.put("$not", query.getFilter());
            return this;
        }

        /**
         * Build the query.
         *
         * @return a new Query instance
         */
        public Query build() {
            return new Query(new HashMap<>(filter));
        }

        @SuppressWarnings("unchecked")
        private Builder addOperator(String field, String operator, Object value) {
            Map<String, Object> fieldConditions;
            if (filter.containsKey(field) && filter.get(field) instanceof Map) {
                fieldConditions = (Map<String, Object>) filter.get(field);
            } else {
                fieldConditions = new HashMap<>();
                filter.put(field, fieldConditions);
            }
            fieldConditions.put(operator, value);
            return this;
        }
    }
}
