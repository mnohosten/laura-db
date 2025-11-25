package com.lauradb.client;

import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

import java.io.IOException;
import java.util.*;

/**
 * Tests for Collection operations.
 *
 * Note: These tests require a running LauraDB server on localhost:8080
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class CollectionTest {
    private static LauraDBClient client;
    private static Collection collection;

    @BeforeAll
    static void setup() {
        client = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .build();
        collection = client.collection("test_collection");
    }

    @AfterAll
    static void teardown() throws IOException {
        // Clean up
        collection.deleteMany(Query.empty());
        if (client != null) {
            client.close();
        }
    }

    @BeforeEach
    void cleanCollection() throws IOException {
        // Clean before each test
        collection.deleteMany(Query.empty());
    }

    @Test
    @Order(1)
    void testInsertOne() throws IOException {
        Map<String, Object> doc = new HashMap<>();
        doc.put("name", "Test User");
        doc.put("age", 25);

        String id = collection.insertOne(doc);
        assertNotNull(id);
        assertFalse(id.isEmpty());
    }

    @Test
    @Order(2)
    void testInsertMany() throws IOException {
        List<Map<String, Object>> docs = new ArrayList<>();

        Map<String, Object> doc1 = new HashMap<>();
        doc1.put("name", "User 1");
        doc1.put("age", 25);
        docs.add(doc1);

        Map<String, Object> doc2 = new HashMap<>();
        doc2.put("name", "User 2");
        doc2.put("age", 30);
        docs.add(doc2);

        List<String> ids = collection.insertMany(docs);
        assertNotNull(ids);
        assertEquals(2, ids.size());
    }

    @Test
    @Order(3)
    void testFindById() throws IOException {
        Map<String, Object> doc = new HashMap<>();
        doc.put("name", "Find Test");
        doc.put("value", 42);

        String id = collection.insertOne(doc);
        Map<String, Object> found = collection.findById(id);

        assertNotNull(found);
        assertEquals("Find Test", found.get("name"));
        assertEquals(42.0, ((Number) found.get("value")).doubleValue());
    }

    @Test
    @Order(4)
    void testFindAll() throws IOException {
        // Insert test documents
        List<Map<String, Object>> docs = new ArrayList<>();
        for (int i = 0; i < 5; i++) {
            Map<String, Object> doc = new HashMap<>();
            doc.put("index", i);
            docs.add(doc);
        }
        collection.insertMany(docs);

        List<Map<String, Object>> results = collection.findAll();
        assertEquals(5, results.size());
    }

    @Test
    @Order(5)
    void testFindWithQuery() throws IOException {
        // Insert test documents
        List<Map<String, Object>> docs = new ArrayList<>();
        for (int i = 0; i < 10; i++) {
            Map<String, Object> doc = new HashMap<>();
            doc.put("age", 20 + i);
            docs.add(doc);
        }
        collection.insertMany(docs);

        // Find documents with age >= 25
        Query query = Query.builder().gte("age", 25).build();
        List<Map<String, Object>> results = collection.find(query);

        assertEquals(5, results.size());
        for (Map<String, Object> doc : results) {
            double age = ((Number) doc.get("age")).doubleValue();
            assertTrue(age >= 25);
        }
    }

    @Test
    @Order(6)
    void testFindOne() throws IOException {
        Map<String, Object> doc = new HashMap<>();
        doc.put("name", "Unique User");
        doc.put("email", "unique@example.com");
        collection.insertOne(doc);

        Query query = Query.builder().eq("email", "unique@example.com").build();
        Map<String, Object> found = collection.findOne(query);

        assertNotNull(found);
        assertEquals("Unique User", found.get("name"));
    }

    @Test
    @Order(7)
    void testFindWithOptions() throws IOException {
        // Insert test documents
        List<Map<String, Object>> docs = new ArrayList<>();
        for (int i = 0; i < 10; i++) {
            Map<String, Object> doc = new HashMap<>();
            doc.put("name", "User " + i);
            doc.put("age", 20 + i);
            docs.add(doc);
        }
        collection.insertMany(docs);

        // Find with projection, sort, and limit
        FindOptions options = new FindOptions()
                .projection(FindOptions.projectionBuilder()
                        .include("name")
                        .exclude("_id")
                        .build())
                .sort(FindOptions.sortBuilder()
                        .descending("age")
                        .build())
                .limit(3);

        List<Map<String, Object>> results = collection.find(Query.empty(), options);

        assertEquals(3, results.size());
        assertFalse(results.get(0).containsKey("_id"));
        assertTrue(results.get(0).containsKey("name"));
    }

    @Test
    @Order(8)
    void testCount() throws IOException {
        // Insert test documents
        List<Map<String, Object>> docs = new ArrayList<>();
        for (int i = 0; i < 15; i++) {
            Map<String, Object> doc = new HashMap<>();
            doc.put("category", i < 10 ? "A" : "B");
            docs.add(doc);
        }
        collection.insertMany(docs);

        long totalCount = collection.count(Query.empty());
        assertEquals(15, totalCount);

        long categoryACount = collection.count(Query.builder().eq("category", "A").build());
        assertEquals(10, categoryACount);
    }

    @Test
    @Order(9)
    void testUpdateOne() throws IOException {
        Map<String, Object> doc = new HashMap<>();
        doc.put("name", "Original Name");
        doc.put("age", 25);
        collection.insertOne(doc);

        Map<String, Object> update = new HashMap<>();
        Map<String, Object> setOp = new HashMap<>();
        setOp.put("name", "Updated Name");
        setOp.put("age", 26);
        update.put("$set", setOp);

        Query query = Query.builder().eq("name", "Original Name").build();
        long modified = collection.updateOne(query, update);

        assertEquals(1, modified);

        Map<String, Object> updated = collection.findOne(Query.builder().eq("name", "Updated Name").build());
        assertNotNull(updated);
        assertEquals(26.0, ((Number) updated.get("age")).doubleValue());
    }

    @Test
    @Order(10)
    void testUpdateMany() throws IOException {
        // Insert test documents
        List<Map<String, Object>> docs = new ArrayList<>();
        for (int i = 0; i < 5; i++) {
            Map<String, Object> doc = new HashMap<>();
            doc.put("status", "pending");
            doc.put("count", 0);
            docs.add(doc);
        }
        collection.insertMany(docs);

        Map<String, Object> update = new HashMap<>();
        Map<String, Object> setOp = new HashMap<>();
        setOp.put("status", "completed");
        update.put("$set", setOp);

        Query query = Query.builder().eq("status", "pending").build();
        long modified = collection.updateMany(query, update);

        assertEquals(5, modified);

        long completedCount = collection.count(Query.builder().eq("status", "completed").build());
        assertEquals(5, completedCount);
    }

    @Test
    @Order(11)
    void testDeleteOne() throws IOException {
        Map<String, Object> doc = new HashMap<>();
        doc.put("name", "To Delete");
        collection.insertOne(doc);

        Query query = Query.builder().eq("name", "To Delete").build();
        long deleted = collection.deleteOne(query);

        assertEquals(1, deleted);

        Map<String, Object> found = collection.findOne(query);
        assertNull(found);
    }

    @Test
    @Order(12)
    void testDeleteMany() throws IOException {
        // Insert test documents
        List<Map<String, Object>> docs = new ArrayList<>();
        for (int i = 0; i < 10; i++) {
            Map<String, Object> doc = new HashMap<>();
            doc.put("category", "temp");
            docs.add(doc);
        }
        collection.insertMany(docs);

        Query query = Query.builder().eq("category", "temp").build();
        long deleted = collection.deleteMany(query);

        assertEquals(10, deleted);

        long count = collection.count(query);
        assertEquals(0, count);
    }

    @Test
    @Order(13)
    void testAsyncOperations() throws Exception {
        Map<String, Object> doc = new HashMap<>();
        doc.put("name", "Async Test");
        doc.put("value", 100);

        // Test async insert
        String id = collection.insertOneAsync(doc).get();
        assertNotNull(id);

        // Test async find
        Query query = Query.builder().eq("name", "Async Test").build();
        List<Map<String, Object>> results = collection.findAsync(query).get();
        assertEquals(1, results.size());
    }
}
