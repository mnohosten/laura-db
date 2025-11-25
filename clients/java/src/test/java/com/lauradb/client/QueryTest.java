package com.lauradb.client;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.util.Map;

/**
 * Tests for Query builder.
 */
class QueryTest {

    @Test
    void testEmptyQuery() {
        Query query = Query.empty();
        assertNotNull(query);
        assertTrue(query.getFilter().isEmpty());
    }

    @Test
    void testEqualityQuery() {
        Query query = Query.builder()
                .eq("name", "Alice")
                .build();

        Map<String, Object> filter = query.getFilter();
        assertEquals("Alice", filter.get("name"));
    }

    @Test
    void testComparisonOperators() {
        Query query = Query.builder()
                .gt("age", 25)
                .gte("score", 80)
                .lt("price", 100)
                .lte("count", 50)
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("age"));
        assertTrue(filter.containsKey("score"));
        assertTrue(filter.containsKey("price"));
        assertTrue(filter.containsKey("count"));
    }

    @Test
    void testInOperator() {
        Query query = Query.builder()
                .in("category", "A", "B", "C")
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("category"));
    }

    @Test
    void testExistsOperator() {
        Query query = Query.builder()
                .exists("email", true)
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("email"));
    }

    @Test
    void testTypeOperator() {
        Query query = Query.builder()
                .type("age", "number")
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("age"));
    }

    @Test
    void testRegexOperator() {
        Query query = Query.builder()
                .regex("email", ".*@example\\.com$")
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("email"));
    }

    @Test
    void testArrayOperators() {
        Query query = Query.builder()
                .all("tags", "java", "database")
                .size("items", 5)
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("tags"));
        assertTrue(filter.containsKey("items"));
    }

    @Test
    void testAndOperator() {
        Query query = Query.builder()
                .and(
                        Query.builder().gte("age", 25).build(),
                        Query.builder().eq("active", true).build()
                )
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("$and"));
    }

    @Test
    void testOrOperator() {
        Query query = Query.builder()
                .or(
                        Query.builder().eq("city", "New York").build(),
                        Query.builder().eq("city", "Boston").build()
                )
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("$or"));
    }

    @Test
    void testNotOperator() {
        Query query = Query.builder()
                .not(Query.builder().eq("status", "deleted").build())
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("$not"));
    }

    @Test
    void testComplexQuery() {
        Query query = Query.builder()
                .and(
                        Query.builder()
                                .gte("age", 18)
                                .lt("age", 65)
                                .build(),
                        Query.builder()
                                .eq("active", true)
                                .exists("email", true)
                                .build()
                )
                .build();

        Map<String, Object> filter = query.getFilter();
        assertFalse(filter.isEmpty());
        assertTrue(filter.containsKey("$and"));
    }

    @Test
    void testMultipleConditionsOnSameField() {
        Query query = Query.builder()
                .gte("age", 25)
                .lt("age", 40)
                .build();

        Map<String, Object> filter = query.getFilter();
        assertTrue(filter.containsKey("age"));

        @SuppressWarnings("unchecked")
        Map<String, Object> ageConditions = (Map<String, Object>) filter.get("age");
        assertTrue(ageConditions.containsKey("$gte"));
        assertTrue(ageConditions.containsKey("$lt"));
    }
}
