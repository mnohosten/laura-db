package com.lauradb.client;

import java.util.*;

/**
 * Aggregation pipeline builder for LauraDB.
 *
 * <p>Example usage:
 * <pre>{@code
 * Aggregation pipeline = Aggregation.builder()
 *     .match(Query.builder().gte("age", 18).build())
 *     .group("$city")
 *         .avg("avgAge", "$age")
 *         .count("count")
 *     .sort(Map.of("avgAge", -1))
 *     .limit(10)
 *     .build();
 *
 * List<Map<String, Object>> results = collection.aggregate(pipeline);
 * }</pre>
 */
public class Aggregation {
    private final List<Map<String, Object>> pipeline;

    private Aggregation(List<Map<String, Object>> pipeline) {
        this.pipeline = pipeline;
    }

    /**
     * Get the pipeline stages.
     *
     * @return the pipeline stages
     */
    public List<Map<String, Object>> getPipeline() {
        return pipeline;
    }

    /**
     * Create a new aggregation builder.
     *
     * @return a new Builder instance
     */
    public static Builder builder() {
        return new Builder();
    }

    /**
     * Builder for constructing aggregation pipelines.
     */
    public static class Builder {
        private final List<Map<String, Object>> stages;
        private GroupBuilder currentGroup;

        private Builder() {
            this.stages = new ArrayList<>();
        }

        /**
         * Add a $match stage to filter documents.
         *
         * @param query the filter query
         * @return this builder
         */
        public Builder match(Query query) {
            Map<String, Object> stage = new HashMap<>();
            stage.put("$match", query.getFilter());
            stages.add(stage);
            return this;
        }

        /**
         * Add a $group stage.
         *
         * @param groupBy the field to group by (e.g., "$city")
         * @return a GroupBuilder for adding aggregations
         */
        public GroupBuilder group(String groupBy) {
            currentGroup = new GroupBuilder(this, groupBy);
            return currentGroup;
        }

        /**
         * Add a $project stage for field selection/transformation.
         *
         * @param projection the projection specification
         * @return this builder
         */
        public Builder project(Map<String, Object> projection) {
            Map<String, Object> stage = new HashMap<>();
            stage.put("$project", projection);
            stages.add(stage);
            return this;
        }

        /**
         * Add a $sort stage.
         *
         * @param sort map of field names to 1 (ascending) or -1 (descending)
         * @return this builder
         */
        public Builder sort(Map<String, Integer> sort) {
            Map<String, Object> stage = new HashMap<>();
            stage.put("$sort", sort);
            stages.add(stage);
            return this;
        }

        /**
         * Add a $limit stage.
         *
         * @param limit maximum number of documents
         * @return this builder
         */
        public Builder limit(int limit) {
            Map<String, Object> stage = new HashMap<>();
            stage.put("$limit", limit);
            stages.add(stage);
            return this;
        }

        /**
         * Add a $skip stage.
         *
         * @param skip number of documents to skip
         * @return this builder
         */
        public Builder skip(int skip) {
            Map<String, Object> stage = new HashMap<>();
            stage.put("$skip", skip);
            stages.add(stage);
            return this;
        }

        /**
         * Build the aggregation pipeline.
         *
         * @return a new Aggregation instance
         */
        public Aggregation build() {
            return new Aggregation(new ArrayList<>(stages));
        }

        void addGroupStage(Map<String, Object> groupSpec) {
            Map<String, Object> stage = new HashMap<>();
            stage.put("$group", groupSpec);
            stages.add(stage);
        }
    }

    /**
     * Builder for $group stage with accumulator operations.
     */
    public static class GroupBuilder {
        private final Builder parent;
        private final String groupBy;
        private final Map<String, Object> groupSpec;

        private GroupBuilder(Builder parent, String groupBy) {
            this.parent = parent;
            this.groupBy = groupBy;
            this.groupSpec = new HashMap<>();
            this.groupSpec.put("_id", groupBy);
        }

        /**
         * Add a $sum aggregation.
         *
         * @param field the output field name
         * @param expression the expression to sum (e.g., "$amount" or 1 for count)
         * @return this group builder
         */
        public GroupBuilder sum(String field, Object expression) {
            Map<String, Object> op = new HashMap<>();
            op.put("$sum", expression);
            groupSpec.put(field, op);
            return this;
        }

        /**
         * Add a $avg aggregation.
         *
         * @param field the output field name
         * @param expression the expression to average (e.g., "$age")
         * @return this group builder
         */
        public GroupBuilder avg(String field, String expression) {
            Map<String, Object> op = new HashMap<>();
            op.put("$avg", expression);
            groupSpec.put(field, op);
            return this;
        }

        /**
         * Add a $min aggregation.
         *
         * @param field the output field name
         * @param expression the expression to find minimum (e.g., "$price")
         * @return this group builder
         */
        public GroupBuilder min(String field, String expression) {
            Map<String, Object> op = new HashMap<>();
            op.put("$min", expression);
            groupSpec.put(field, op);
            return this;
        }

        /**
         * Add a $max aggregation.
         *
         * @param field the output field name
         * @param expression the expression to find maximum (e.g., "$price")
         * @return this group builder
         */
        public GroupBuilder max(String field, String expression) {
            Map<String, Object> op = new HashMap<>();
            op.put("$max", expression);
            groupSpec.put(field, op);
            return this;
        }

        /**
         * Add a $count aggregation (counts documents in each group).
         *
         * @param field the output field name
         * @return this group builder
         */
        public GroupBuilder count(String field) {
            Map<String, Object> op = new HashMap<>();
            op.put("$count", new HashMap<>());
            groupSpec.put(field, op);
            return this;
        }

        /**
         * Add a $push aggregation (collects values into an array).
         *
         * @param field the output field name
         * @param expression the expression to push (e.g., "$name")
         * @return this group builder
         */
        public GroupBuilder push(String field, String expression) {
            Map<String, Object> op = new HashMap<>();
            op.put("$push", expression);
            groupSpec.put(field, op);
            return this;
        }

        /**
         * Complete the group stage and return to the main builder.
         *
         * @return the parent builder
         */
        public Builder end() {
            parent.addGroupStage(groupSpec);
            return parent;
        }

        /**
         * Complete the group stage and add a $match stage.
         *
         * @param query the filter query
         * @return the parent builder
         */
        public Builder match(Query query) {
            end();
            return parent.match(query);
        }

        /**
         * Complete the group stage and add a $project stage.
         *
         * @param projection the projection specification
         * @return the parent builder
         */
        public Builder project(Map<String, Object> projection) {
            end();
            return parent.project(projection);
        }

        /**
         * Complete the group stage and add a $sort stage.
         *
         * @param sort the sort specification
         * @return the parent builder
         */
        public Builder sort(Map<String, Integer> sort) {
            end();
            return parent.sort(sort);
        }

        /**
         * Complete the group stage and add a $limit stage.
         *
         * @param limit the limit value
         * @return the parent builder
         */
        public Builder limit(int limit) {
            end();
            return parent.limit(limit);
        }

        /**
         * Complete the group stage and add a $skip stage.
         *
         * @param skip the skip value
         * @return the parent builder
         */
        public Builder skip(int skip) {
            end();
            return parent.skip(skip);
        }

        /**
         * Complete the group stage and build the pipeline.
         *
         * @return the aggregation pipeline
         */
        public Aggregation build() {
            end();
            return parent.build();
        }
    }
}
