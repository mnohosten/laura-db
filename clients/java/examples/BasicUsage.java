import com.lauradb.client.*;

import java.io.IOException;
import java.util.*;

/**
 * Basic usage example for LauraDB Java client.
 *
 * This example demonstrates:
 * - Connecting to LauraDB
 * - Creating a collection
 * - Inserting documents
 * - Finding documents with queries
 * - Updating documents
 * - Deleting documents
 */
public class BasicUsage {
    public static void main(String[] args) {
        // Connect to LauraDB server
        try (LauraDBClient client = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .build()) {

            System.out.println("=== LauraDB Java Client - Basic Usage ===\n");

            // Check connection
            if (!client.ping()) {
                System.err.println("Failed to connect to LauraDB server");
                return;
            }
            System.out.println("✓ Connected to LauraDB server\n");

            // Get a collection
            Collection users = client.collection("users");
            System.out.println("=== Working with 'users' collection ===\n");

            // Insert a single document
            Map<String, Object> user1 = new HashMap<>();
            user1.put("name", "Alice");
            user1.put("email", "alice@example.com");
            user1.put("age", 30);
            user1.put("city", "New York");
            user1.put("active", true);

            String id1 = users.insertOne(user1);
            System.out.println("Inserted user: " + id1);

            // Insert multiple documents
            List<Map<String, Object>> moreUsers = new ArrayList<>();

            Map<String, Object> user2 = new HashMap<>();
            user2.put("name", "Bob");
            user2.put("email", "bob@example.com");
            user2.put("age", 25);
            user2.put("city", "San Francisco");
            user2.put("active", true);
            moreUsers.add(user2);

            Map<String, Object> user3 = new HashMap<>();
            user3.put("name", "Charlie");
            user3.put("email", "charlie@example.com");
            user3.put("age", 35);
            user3.put("city", "New York");
            user3.put("active", false);
            moreUsers.add(user3);

            Map<String, Object> user4 = new HashMap<>();
            user4.put("name", "Diana");
            user4.put("email", "diana@example.com");
            user4.put("age", 28);
            user4.put("city", "Boston");
            user4.put("active", true);
            moreUsers.add(user4);

            List<String> ids = users.insertMany(moreUsers);
            System.out.println("Inserted " + ids.size() + " more users\n");

            // Find all users
            System.out.println("=== Find All Users ===");
            List<Map<String, Object>> allUsers = users.findAll();
            System.out.println("Total users: " + allUsers.size() + "\n");

            // Find users with a query
            System.out.println("=== Find Users age >= 30 ===");
            Query ageQuery = Query.builder()
                    .gte("age", 30)
                    .build();
            List<Map<String, Object>> olderUsers = users.find(ageQuery);
            for (Map<String, Object> user : olderUsers) {
                System.out.println("- " + user.get("name") + " (age: " + user.get("age") + ")");
            }
            System.out.println();

            // Find users in New York
            System.out.println("=== Find Users in New York ===");
            Query cityQuery = Query.builder()
                    .eq("city", "New York")
                    .build();
            List<Map<String, Object>> nyUsers = users.find(cityQuery);
            for (Map<String, Object> user : nyUsers) {
                System.out.println("- " + user.get("name") + " (" + user.get("city") + ")");
            }
            System.out.println();

            // Find with options (projection, sort, limit)
            System.out.println("=== Find with Options (name and email only, sorted by age desc, limit 2) ===");
            FindOptions options = new FindOptions()
                    .projection(FindOptions.projectionBuilder()
                            .include("name")
                            .include("email")
                            .exclude("_id")
                            .build())
                    .sort(FindOptions.sortBuilder()
                            .descending("age")
                            .build())
                    .limit(2);

            List<Map<String, Object>> limitedUsers = users.find(Query.empty(), options);
            for (Map<String, Object> user : limitedUsers) {
                System.out.println("- " + user.get("name") + " (" + user.get("email") + ")");
            }
            System.out.println();

            // Count documents
            long activeCount = users.count(Query.builder().eq("active", true).build());
            System.out.println("Active users: " + activeCount + "\n");

            // Update a document
            System.out.println("=== Update Document ===");
            Map<String, Object> update = new HashMap<>();
            Map<String, Object> setOp = new HashMap<>();
            setOp.put("age", 31);
            setOp.put("city", "Los Angeles");
            update.put("$set", setOp);

            Query updateQuery = Query.builder().eq("name", "Alice").build();
            long modified = users.updateOne(updateQuery, update);
            System.out.println("Modified documents: " + modified);

            // Verify update
            Map<String, Object> alice = users.findOne(Query.builder().eq("name", "Alice").build());
            if (alice != null) {
                System.out.println("Alice's new age: " + alice.get("age"));
                System.out.println("Alice's new city: " + alice.get("city") + "\n");
            }

            // Update multiple documents
            System.out.println("=== Update Multiple Documents ===");
            Map<String, Object> multiUpdate = new HashMap<>();
            Map<String, Object> incOp = new HashMap<>();
            incOp.put("age", 1);
            multiUpdate.put("$inc", incOp);

            Query activeQuery = Query.builder().eq("active", true).build();
            long multiModified = users.updateMany(activeQuery, multiUpdate);
            System.out.println("Incremented age for " + multiModified + " active users\n");

            // Delete a document
            System.out.println("=== Delete Document ===");
            Query deleteQuery = Query.builder().eq("name", "Charlie").build();
            long deleted = users.deleteOne(deleteQuery);
            System.out.println("Deleted documents: " + deleted);

            long totalAfterDelete = users.count(Query.empty());
            System.out.println("Total users after delete: " + totalAfterDelete + "\n");

            // Complex query with logical operators
            System.out.println("=== Complex Query (active AND (age >= 25 AND age < 35)) ===");
            Query complexQuery = Query.builder()
                    .and(
                            Query.builder().eq("active", true).build(),
                            Query.builder().gte("age", 25).lt("age", 35).build()
                    )
                    .build();

            List<Map<String, Object>> filtered = users.find(complexQuery);
            for (Map<String, Object> user : filtered) {
                System.out.println("- " + user.get("name") + " (age: " + user.get("age") + ", active: " + user.get("active") + ")");
            }
            System.out.println();

            // Clean up - delete all test documents
            System.out.println("=== Cleanup ===");
            long allDeleted = users.deleteMany(Query.empty());
            System.out.println("Deleted " + allDeleted + " documents");

            System.out.println("\n=== Example Complete ===");

        } catch (IOException e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
        }
    }
}
