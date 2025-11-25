import com.lauradb.client.*;

import java.io.IOException;
import java.util.*;

/**
 * Aggregation pipeline example for LauraDB Java client.
 *
 * This example demonstrates:
 * - Building aggregation pipelines
 * - Using $match, $group, $project, $sort, $limit stages
 * - Using aggregation operators ($avg, $sum, $count, $min, $max)
 */
public class AggregationExample {
    public static void main(String[] args) {
        try (LauraDBClient client = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .build()) {

            System.out.println("=== LauraDB Java Client - Aggregation Example ===\n");

            if (!client.ping()) {
                System.err.println("Failed to connect to LauraDB server");
                return;
            }
            System.out.println("✓ Connected to LauraDB server\n");

            // Get collection
            Collection orders = client.collection("orders");

            // Insert sample data
            System.out.println("=== Inserting Sample Data ===");
            List<Map<String, Object>> sampleOrders = new ArrayList<>();

            sampleOrders.add(createOrder("Alice", "Electronics", 1200, "New York", "2024-01-15"));
            sampleOrders.add(createOrder("Bob", "Books", 45, "San Francisco", "2024-01-16"));
            sampleOrders.add(createOrder("Alice", "Books", 30, "New York", "2024-01-17"));
            sampleOrders.add(createOrder("Charlie", "Electronics", 800, "Boston", "2024-01-18"));
            sampleOrders.add(createOrder("Diana", "Clothing", 150, "New York", "2024-01-19"));
            sampleOrders.add(createOrder("Bob", "Electronics", 2000, "San Francisco", "2024-01-20"));
            sampleOrders.add(createOrder("Alice", "Clothing", 200, "New York", "2024-01-21"));
            sampleOrders.add(createOrder("Charlie", "Books", 60, "Boston", "2024-01-22"));
            sampleOrders.add(createOrder("Diana", "Electronics", 1500, "New York", "2024-01-23"));
            sampleOrders.add(createOrder("Bob", "Clothing", 100, "San Francisco", "2024-01-24"));

            orders.insertMany(sampleOrders);
            System.out.println("Inserted " + sampleOrders.size() + " sample orders\n");

            // Example 1: Group by customer and calculate totals
            System.out.println("=== Example 1: Total orders and amount per customer ===");
            Aggregation pipeline1 = Aggregation.builder()
                    .group("$customer")
                        .sum("totalAmount", "$amount")
                        .count("orderCount")
                        .avg("avgAmount", "$amount")
                    .sort(Map.of("totalAmount", -1))
                    .build();

            List<Map<String, Object>> results1 = orders.aggregate(pipeline1);
            for (Map<String, Object> result : results1) {
                System.out.println("Customer: " + result.get("_id"));
                System.out.println("  Total Amount: $" + result.get("totalAmount"));
                System.out.println("  Order Count: " + result.get("orderCount"));
                System.out.println("  Avg Amount: $" + String.format("%.2f", ((Number) result.get("avgAmount")).doubleValue()));
            }
            System.out.println();

            // Example 2: Group by category with filtering
            System.out.println("=== Example 2: High-value orders (>$100) by category ===");
            Aggregation pipeline2 = Aggregation.builder()
                    .match(Query.builder().gt("amount", 100).build())
                    .group("$category")
                        .sum("totalAmount", "$amount")
                        .count("count")
                        .max("maxAmount", "$amount")
                        .min("minAmount", "$amount")
                    .sort(Map.of("totalAmount", -1))
                    .build();

            List<Map<String, Object>> results2 = orders.aggregate(pipeline2);
            for (Map<String, Object> result : results2) {
                System.out.println("Category: " + result.get("_id"));
                System.out.println("  Total Amount: $" + result.get("totalAmount"));
                System.out.println("  Order Count: " + result.get("count"));
                System.out.println("  Max Amount: $" + result.get("maxAmount"));
                System.out.println("  Min Amount: $" + result.get("minAmount"));
            }
            System.out.println();

            // Example 3: Group by city
            System.out.println("=== Example 3: Orders by city (top 2) ===");
            Aggregation pipeline3 = Aggregation.builder()
                    .group("$city")
                        .sum("totalRevenue", "$amount")
                        .count("orderCount")
                    .sort(Map.of("totalRevenue", -1))
                    .limit(2)
                    .build();

            List<Map<String, Object>> results3 = orders.aggregate(pipeline3);
            for (Map<String, Object> result : results3) {
                System.out.println("City: " + result.get("_id"));
                System.out.println("  Total Revenue: $" + result.get("totalRevenue"));
                System.out.println("  Order Count: " + result.get("orderCount"));
            }
            System.out.println();

            // Example 4: Multi-stage pipeline with projection
            System.out.println("=== Example 4: Electronics orders with customer details ===");
            Aggregation pipeline4 = Aggregation.builder()
                    .match(Query.builder().eq("category", "Electronics").build())
                    .project(Map.of(
                            "customer", 1,
                            "amount", 1,
                            "city", 1,
                            "_id", 0
                    ))
                    .sort(Map.of("amount", -1))
                    .limit(3)
                    .build();

            List<Map<String, Object>> results4 = orders.aggregate(pipeline4);
            for (Map<String, Object> result : results4) {
                System.out.println("- " + result.get("customer") +
                        " ($" + result.get("amount") +
                        ", " + result.get("city") + ")");
            }
            System.out.println();

            // Example 5: Complex aggregation with multiple groups
            System.out.println("=== Example 5: Category statistics per city ===");
            Aggregation pipeline5 = Aggregation.builder()
                    .group("$city")
                        .push("categories", "$category")
                        .sum("totalOrders", 1)
                        .avg("avgOrderAmount", "$amount")
                    .sort(Map.of("totalOrders", -1))
                    .build();

            List<Map<String, Object>> results5 = orders.aggregate(pipeline5);
            for (Map<String, Object> result : results5) {
                System.out.println("City: " + result.get("_id"));
                System.out.println("  Total Orders: " + result.get("totalOrders"));
                System.out.println("  Avg Order Amount: $" + String.format("%.2f", ((Number) result.get("avgOrderAmount")).doubleValue()));
                System.out.println("  Categories: " + result.get("categories"));
            }
            System.out.println();

            // Clean up
            System.out.println("=== Cleanup ===");
            long deleted = orders.deleteMany(Query.empty());
            System.out.println("Deleted " + deleted + " documents");

            System.out.println("\n=== Example Complete ===");

        } catch (IOException e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
        }
    }

    private static Map<String, Object> createOrder(String customer, String category, int amount, String city, String date) {
        Map<String, Object> order = new HashMap<>();
        order.put("customer", customer);
        order.put("category", category);
        order.put("amount", amount);
        order.put("city", city);
        order.put("date", date);
        return order;
    }
}
