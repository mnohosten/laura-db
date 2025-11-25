package com.lauradb.client;

import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

import java.io.IOException;
import java.util.List;
import java.util.Map;

/**
 * Tests for LauraDBClient.
 *
 * Note: These tests require a running LauraDB server on localhost:8080
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class LauraDBClientTest {
    private static LauraDBClient client;

    @BeforeAll
    static void setup() {
        client = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .build();
    }

    @AfterAll
    static void teardown() {
        if (client != null) {
            client.close();
        }
    }

    @Test
    @Order(1)
    void testPing() {
        boolean healthy = client.ping();
        assertTrue(healthy, "Server should be healthy");
    }

    @Test
    @Order(2)
    void testPingAsync() throws Exception {
        Boolean healthy = client.pingAsync().get();
        assertNotNull(healthy);
        assertTrue(healthy, "Server should be healthy");
    }

    @Test
    @Order(3)
    void testBuilder() {
        LauraDBClient testClient = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .https(false)
                .timeout(30000)
                .build();

        assertNotNull(testClient);
        boolean healthy = testClient.ping();
        assertTrue(healthy);
        testClient.close();
    }

    @Test
    @Order(4)
    void testCollection() {
        Collection collection = client.collection("test");
        assertNotNull(collection);
        assertEquals("test", collection.getName());
    }

    @Test
    @Order(5)
    void testStats() throws IOException {
        Map<String, Object> stats = client.stats();
        assertNotNull(stats);
        assertTrue(stats.containsKey("ok"));
    }

    @Test
    @Order(6)
    void testListCollections() throws IOException {
        List<String> collections = client.listCollections();
        assertNotNull(collections);
    }

    @Test
    @Order(7)
    void testCreateAndDropCollection() throws IOException {
        String collectionName = "test_collection_" + System.currentTimeMillis();

        // Create collection
        client.createCollection(collectionName);

        // Verify it exists
        List<String> collections = client.listCollections();
        assertTrue(collections.contains(collectionName));

        // Drop collection
        client.dropCollection(collectionName);

        // Verify it's gone
        collections = client.listCollections();
        assertFalse(collections.contains(collectionName));
    }
}
