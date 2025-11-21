package main

import (
	"fmt"
	"log"

	"github.com/mnohosten/laura-db/pkg/database"
)

func main() {
	fmt.Println("=== Aggregation Pipeline Demo ===\n")

	// Open database
	config := database.DefaultConfig("./agg_demo_data")
	db, err := database.Open(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create sales collection
	sales := db.Collection("sales")

	// Insert sample sales data
	fmt.Println("--- Inserting Sample Data ---")
	salesData := []map[string]interface{}{
		{"product": "Laptop", "category": "Electronics", "price": 999.99, "quantity": int64(2), "region": "North"},
		{"product": "Mouse", "category": "Electronics", "price": 29.99, "quantity": int64(10), "region": "North"},
		{"product": "Keyboard", "category": "Electronics", "price": 79.99, "quantity": int64(5), "region": "South"},
		{"product": "Monitor", "category": "Electronics", "price": 299.99, "quantity": int64(3), "region": "North"},
		{"product": "Desk", "category": "Furniture", "price": 399.99, "quantity": int64(2), "region": "South"},
		{"product": "Chair", "category": "Furniture", "price": 199.99, "quantity": int64(4), "region": "South"},
		{"product": "Lamp", "category": "Furniture", "price": 49.99, "quantity": int64(8), "region": "North"},
		{"product": "Tablet", "category": "Electronics", "price": 499.99, "quantity": int64(3), "region": "South"},
	}

	sales.InsertMany(salesData)
	fmt.Printf("Inserted %d sales records\n\n", len(salesData))

	// Example 1: Filter and sort
	fmt.Println("--- Example 1: Filter and Sort ---")
	fmt.Println("Find Electronics over $50, sorted by price:")

	results, _ := sales.Aggregate([]map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"category": "Electronics",
				"price": map[string]interface{}{
					"$gt": 50.0,
				},
			},
		},
		{
			"$sort": map[string]interface{}{
				"price": -1, // Descending
			},
		},
		{
			"$project": map[string]interface{}{
				"product": true,
				"price":   true,
			},
		},
	})

	for _, doc := range results {
		fmt.Printf("  %v\n", doc.ToMap())
	}

	// Example 2: Group by category
	fmt.Println("\n--- Example 2: Group by Category ---")
	fmt.Println("Total sales by category:")

	results, _ = sales.Aggregate([]map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": "$category",
				"totalRevenue": map[string]interface{}{
					"$sum": "$price",
				},
				"totalItems": map[string]interface{}{
					"$sum": "$quantity",
				},
				"avgPrice": map[string]interface{}{
					"$avg": "$price",
				},
			},
		},
		{
			"$sort": map[string]interface{}{
				"totalRevenue": -1,
			},
		},
	})

	for _, doc := range results {
		category, _ := doc.Get("_id")
		revenue, _ := doc.Get("totalRevenue")
		items, _ := doc.Get("totalItems")
		avg, _ := doc.Get("avgPrice")
		fmt.Printf("  %s:\n", category)
		fmt.Printf("    Revenue: $%.2f\n", revenue)
		fmt.Printf("    Items: %v\n", items)
		fmt.Printf("    Avg Price: $%.2f\n", avg)
	}

	// Example 3: Group by region
	fmt.Println("\n--- Example 3: Group by Region ---")
	fmt.Println("Sales summary by region:")

	results, _ = sales.Aggregate([]map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": "$region",
				"totalSales": map[string]interface{}{
					"$sum": "$price",
				},
				"productCount": map[string]interface{}{
					"$count": nil,
				},
				"maxPrice": map[string]interface{}{
					"$max": "$price",
				},
				"minPrice": map[string]interface{}{
					"$min": "$price",
				},
			},
		},
	})

	for _, doc := range results {
		region, _ := doc.Get("_id")
		total, _ := doc.Get("totalSales")
		count, _ := doc.Get("productCount")
		maxPrice, _ := doc.Get("maxPrice")
		minPrice, _ := doc.Get("minPrice")
		fmt.Printf("  %s:\n", region)
		fmt.Printf("    Total: $%.2f\n", total)
		fmt.Printf("    Products: %v\n", count)
		fmt.Printf("    Price Range: $%.2f - $%.2f\n", minPrice, maxPrice)
	}

	// Example 4: Complex pipeline
	fmt.Println("\n--- Example 4: Complex Pipeline ---")
	fmt.Println("Top 3 most expensive items in Electronics:")

	results, _ = sales.Aggregate([]map[string]interface{}{
		{
			"$match": map[string]interface{}{
				"category": "Electronics",
			},
		},
		{
			"$sort": map[string]interface{}{
				"price": -1,
			},
		},
		{
			"$limit": 3,
		},
		{
			"$project": map[string]interface{}{
				"product": true,
				"price":   true,
				"region":  true,
			},
		},
	})

	for i, doc := range results {
		fmt.Printf("  %d. %v\n", i+1, doc.ToMap())
	}

	// Example 5: Skip and Limit
	fmt.Println("\n--- Example 5: Pagination (Skip & Limit) ---")
	fmt.Println("Page 2 of products (2 per page):")

	results, _ = sales.Aggregate([]map[string]interface{}{
		{
			"$sort": map[string]interface{}{
				"product": 1,
			},
		},
		{
			"$skip": 2,
		},
		{
			"$limit": 2,
		},
		{
			"$project": map[string]interface{}{
				"product": true,
				"price":   true,
			},
		},
	})

	for _, doc := range results {
		fmt.Printf("  %v\n", doc.ToMap())
	}

	// Example 6: Match after group
	fmt.Println("\n--- Example 6: Filter After Grouping ---")
	fmt.Println("Categories with average price > $200:")

	results, _ = sales.Aggregate([]map[string]interface{}{
		{
			"$group": map[string]interface{}{
				"_id": "$category",
				"avgPrice": map[string]interface{}{
					"$avg": "$price",
				},
				"count": map[string]interface{}{
					"$count": nil,
				},
			},
		},
		{
			"$match": map[string]interface{}{
				"avgPrice": map[string]interface{}{
					"$gt": 200.0,
				},
			},
		},
	})

	for _, doc := range results {
		category, _ := doc.Get("_id")
		avgPrice, _ := doc.Get("avgPrice")
		count, _ := doc.Get("count")
		fmt.Printf("  %s: $%.2f avg (%v items)\n", category, avgPrice, count)
	}

	fmt.Println("\n=== Aggregation Demo Complete ===")
}
