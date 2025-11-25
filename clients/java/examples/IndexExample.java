import com.lauradb.client.*;

import java.io.IOException;
import java.util.*;

/**
 * Index management example for LauraDB Java client.
 *
 * This example demonstrates:
 * - Creating B+ tree indexes (unique and non-unique)
 * - Creating compound indexes
 * - Creating text indexes for full-text search
 * - Creating geospatial indexes
 * - Creating TTL indexes for automatic expiration
 * - Creating partial indexes
 * - Listing and dropping indexes
 */
public class IndexExample {
    public static void main(String[] args) {
        try (LauraDBClient client = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .build()) {

            System.out.println("=== LauraDB Java Client - Index Example ===\n");

            if (!client.ping()) {
                System.err.println("Failed to connect to LauraDB server");
                return;
            }
            System.out.println("✓ Connected to LauraDB server\n");

            // Example 1: Simple B+ tree index
            System.out.println("=== Example 1: B+ Tree Indexes ===");
            Collection users = client.collection("users");

            // Insert sample data
            List<Map<String, Object>> sampleUsers = new ArrayList<>();
            sampleUsers.add(createUser("alice@example.com", "Alice", 30, "New York", true));
            sampleUsers.add(createUser("bob@example.com", "Bob", 25, "San Francisco", true));
            sampleUsers.add(createUser("charlie@example.com", "Charlie", 35, "Boston", false));
            users.insertMany(sampleUsers);
            System.out.println("Inserted " + sampleUsers.size() + " users");

            // Create unique index on email
            users.createIndex("email")
                    .unique(true)
                    .build();
            System.out.println("✓ Created unique index on 'email'");

            // Create non-unique index on age
            users.createIndex("age")
                    .build();
            System.out.println("✓ Created index on 'age'");

            // List indexes
            List<String> userIndexes = users.listIndexes();
            System.out.println("Indexes: " + userIndexes + "\n");

            // Example 2: Compound index
            System.out.println("=== Example 2: Compound Index ===");
            users.createCompoundIndex(List.of("city", "age"))
                    .name("city_age_idx")
                    .build();
            System.out.println("✓ Created compound index on 'city' and 'age'");

            userIndexes = users.listIndexes();
            System.out.println("Indexes: " + userIndexes + "\n");

            // Example 3: Text index
            System.out.println("=== Example 3: Text Index ===");
            Collection posts = client.collection("posts");

            List<Map<String, Object>> samplePosts = new ArrayList<>();
            samplePosts.add(createPost("Introduction to LauraDB", "LauraDB is a MongoDB-like document database", List.of("database", "nosql")));
            samplePosts.add(createPost("Java Client Tutorial", "Learn how to use the LauraDB Java client", List.of("java", "tutorial")));
            samplePosts.add(createPost("Indexing Guide", "Understanding indexes in LauraDB", List.of("database", "indexing")));
            posts.insertMany(samplePosts);
            System.out.println("Inserted " + samplePosts.size() + " posts");

            // Create text index
            posts.createIndex("content")
                    .text()
                    .name("content_text")
                    .build();
            System.out.println("✓ Created text index on 'content'");

            List<String> postIndexes = posts.listIndexes();
            System.out.println("Indexes: " + postIndexes + "\n");

            // Example 4: Geospatial index
            System.out.println("=== Example 4: Geospatial Index ===");
            Collection locations = client.collection("locations");

            List<Map<String, Object>> sampleLocations = new ArrayList<>();
            sampleLocations.add(createLocation("Central Park", 40.785091, -73.968285));
            sampleLocations.add(createLocation("Golden Gate Bridge", 37.819929, -122.478255));
            sampleLocations.add(createLocation("Boston Common", 42.355238, -71.066291));
            locations.insertMany(sampleLocations);
            System.out.println("Inserted " + sampleLocations.size() + " locations");

            // Create 2dsphere index for Earth coordinates
            locations.createIndex("coordinates")
                    .geo("2dsphere")
                    .build();
            System.out.println("✓ Created 2dsphere geospatial index on 'coordinates'");

            List<String> locationIndexes = locations.listIndexes();
            System.out.println("Indexes: " + locationIndexes + "\n");

            // Example 5: TTL index
            System.out.println("=== Example 5: TTL Index (Time-to-Live) ===");
            Collection sessions = client.collection("sessions");

            Map<String, Object> session = new HashMap<>();
            session.put("userId", "user123");
            session.put("token", "abc123xyz");
            session.put("createdAt", new Date().getTime() / 1000); // Unix timestamp
            sessions.insertOne(session);
            System.out.println("Inserted session document");

            // Create TTL index - documents expire after 3600 seconds (1 hour)
            sessions.createIndex("createdAt")
                    .ttl(3600)
                    .build();
            System.out.println("✓ Created TTL index on 'createdAt' (expire after 3600s)");

            List<String> sessionIndexes = sessions.listIndexes();
            System.out.println("Indexes: " + sessionIndexes + "\n");

            // Example 6: Partial index
            System.out.println("=== Example 6: Partial Index ===");
            Collection products = client.collection("products");

            List<Map<String, Object>> sampleProducts = new ArrayList<>();
            sampleProducts.add(createProduct("P001", "Laptop", 1200, true));
            sampleProducts.add(createProduct("P002", "Mouse", 25, true));
            sampleProducts.add(createProduct("P003", "Keyboard", 80, false));
            sampleProducts.add(createProduct("P004", "Monitor", 400, true));
            products.insertMany(sampleProducts);
            System.out.println("Inserted " + sampleProducts.size() + " products");

            // Create partial index - only index active products
            products.createIndex("sku")
                    .unique(true)
                    .partial(Query.builder().eq("active", true).build())
                    .build();
            System.out.println("✓ Created partial index on 'sku' (only active products)");

            List<String> productIndexes = products.listIndexes();
            System.out.println("Indexes: " + productIndexes + "\n");

            // Example 7: Background index building
            System.out.println("=== Example 7: Background Index Building ===");
            Collection items = client.collection("items");

            // Insert many documents
            List<Map<String, Object>> manyItems = new ArrayList<>();
            for (int i = 0; i < 100; i++) {
                Map<String, Object> item = new HashMap<>();
                item.put("name", "Item " + i);
                item.put("value", i * 10);
                manyItems.add(item);
            }
            items.insertMany(manyItems);
            System.out.println("Inserted " + manyItems.size() + " items");

            // Build index in background (non-blocking)
            items.createIndex("value")
                    .background(true)
                    .build();
            System.out.println("✓ Building index on 'value' in background");

            List<String> itemIndexes = items.listIndexes();
            System.out.println("Indexes: " + itemIndexes + "\n");

            // Clean up
            System.out.println("=== Cleanup ===");
            users.deleteMany(Query.empty());
            posts.deleteMany(Query.empty());
            locations.deleteMany(Query.empty());
            sessions.deleteMany(Query.empty());
            products.deleteMany(Query.empty());
            items.deleteMany(Query.empty());
            System.out.println("Cleaned up all test data");

            System.out.println("\n=== Example Complete ===");

        } catch (IOException e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
        }
    }

    private static Map<String, Object> createUser(String email, String name, int age, String city, boolean active) {
        Map<String, Object> user = new HashMap<>();
        user.put("email", email);
        user.put("name", name);
        user.put("age", age);
        user.put("city", city);
        user.put("active", active);
        return user;
    }

    private static Map<String, Object> createPost(String title, String content, List<String> tags) {
        Map<String, Object> post = new HashMap<>();
        post.put("title", title);
        post.put("content", content);
        post.put("tags", tags);
        return post;
    }

    private static Map<String, Object> createLocation(String name, double lat, double lon) {
        Map<String, Object> location = new HashMap<>();
        location.put("name", name);
        Map<String, Object> coords = new HashMap<>();
        coords.put("type", "Point");
        coords.put("coordinates", List.of(lon, lat)); // [longitude, latitude] for GeoJSON
        location.put("coordinates", coords);
        return location;
    }

    private static Map<String, Object> createProduct(String sku, String name, int price, boolean active) {
        Map<String, Object> product = new HashMap<>();
        product.put("sku", sku);
        product.put("name", name);
        product.put("price", price);
        product.put("active", active);
        return product;
    }
}
