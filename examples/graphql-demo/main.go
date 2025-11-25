package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const graphqlEndpoint = "http://localhost:8080/graphql"

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   map[string]interface{}   `json:"data"`
	Errors []map[string]interface{} `json:"errors"`
}

func main() {
	fmt.Println("LauraDB GraphQL API Demo")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("⚠️  Prerequisites:")
	fmt.Println("   Start LauraDB server with GraphQL enabled:")
	fmt.Println("   ./bin/laura-server -graphql")
	fmt.Println()

	// Wait a moment for user to start server
	fmt.Println("Waiting 2 seconds for server to be ready...")
	time.Sleep(2 * time.Second)

	// Demo 1: List collections
	fmt.Println("1. List all collections")
	fmt.Println("=======================")
	listCollections()

	// Demo 2: Create collection
	fmt.Println("\n2. Create a new collection")
	fmt.Println("===========================")
	createCollection("products")

	// Demo 3: Insert documents
	fmt.Println("\n3. Insert documents")
	fmt.Println("====================")
	insertDocuments()

	// Demo 4: Query documents
	fmt.Println("\n4. Query documents")
	fmt.Println("===================")
	queryDocuments()

	// Demo 5: Update document
	fmt.Println("\n5. Update document")
	fmt.Println("===================")
	updateDocument()

	// Demo 6: Count documents
	fmt.Println("\n6. Count documents")
	fmt.Println("===================")
	countDocuments()

	// Demo 7: Create index
	fmt.Println("\n7. Create index")
	fmt.Println("================")
	createIndex()

	// Demo 8: List indexes
	fmt.Println("\n8. List indexes")
	fmt.Println("================")
	listIndexes()

	// Demo 9: Aggregate data
	fmt.Println("\n9. Aggregate data")
	fmt.Println("==================")
	aggregateData()

	// Demo 10: Delete document
	fmt.Println("\n10. Delete document")
	fmt.Println("====================")
	deleteDocument()

	// Demo 11: Collection stats
	fmt.Println("\n11. Get collection stats")
	fmt.Println("=========================")
	getStats()

	fmt.Println("\n✅ GraphQL demo completed!")
	fmt.Println("\nNext steps:")
	fmt.Println("   - Open GraphiQL playground: http://localhost:8080/graphiql")
	fmt.Println("   - Explore the schema using the Docs panel")
	fmt.Println("   - Try your own queries and mutations")
}

func executeGraphQL(query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	req := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(graphqlEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var graphqlResp GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(graphqlResp.Errors) > 0 {
		return &graphqlResp, fmt.Errorf("GraphQL errors: %v", graphqlResp.Errors)
	}

	return &graphqlResp, nil
}

func listCollections() {
	query := `
		query {
			listCollections
		}
	`

	resp, err := executeGraphQL(query, nil)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Collections: %v\n", resp.Data["listCollections"])
}

func createCollection(name string) {
	query := `
		mutation CreateCollection($name: String!) {
			createCollection(name: $name)
		}
	`

	variables := map[string]interface{}{
		"name": name,
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Collection '%s' created: %v\n", name, resp.Data["createCollection"])
}

func insertDocuments() {
	query := `
		mutation InsertProducts($collection: String!, $documents: [JSON!]) {
			insertMany(collection: $collection, documents: $documents) {
				insertedIds
				insertedCount
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
		"documents": []interface{}{
			map[string]interface{}{
				"name":     "Laptop",
				"price":    999,
				"category": "electronics",
				"stock":    50,
			},
			map[string]interface{}{
				"name":     "Mouse",
				"price":    25,
				"category": "electronics",
				"stock":    200,
			},
			map[string]interface{}{
				"name":     "Desk",
				"price":    299,
				"category": "furniture",
				"stock":    30,
			},
		},
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	result := resp.Data["insertMany"].(map[string]interface{})
	fmt.Printf("✅ Inserted %v documents\n", result["insertedCount"])
	fmt.Printf("   IDs: %v\n", result["insertedIds"])
}

func queryDocuments() {
	query := `
		query FindProducts($collection: String!, $filter: JSON) {
			find(collection: $collection, filter: $filter, limit: 10) {
				_id
				data
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
		"filter": map[string]interface{}{
			"category": "electronics",
		},
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	docs := resp.Data["find"].([]interface{})
	fmt.Printf("✅ Found %d electronics products:\n", len(docs))
	for _, doc := range docs {
		docMap := doc.(map[string]interface{})
		data := docMap["data"].(map[string]interface{})
		fmt.Printf("   - %s: $%v\n", data["name"], data["price"])
	}
}

func updateDocument() {
	query := `
		mutation UpdateProduct($collection: String!, $filter: JSON!, $update: JSON!) {
			updateOne(collection: $collection, filter: $filter, update: $update) {
				matchedCount
				modifiedCount
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
		"filter": map[string]interface{}{
			"name": "Laptop",
		},
		"update": map[string]interface{}{
			"$set": map[string]interface{}{
				"price": 899, // Discount!
			},
		},
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	result := resp.Data["updateOne"].(map[string]interface{})
	fmt.Printf("✅ Updated %v documents\n", result["modifiedCount"])
}

func countDocuments() {
	query := `
		query CountProducts($collection: String!, $filter: JSON) {
			count(collection: $collection, filter: $filter)
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
		"filter": map[string]interface{}{
			"category": "electronics",
		},
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	count := resp.Data["count"]
	fmt.Printf("✅ Electronics count: %v\n", count)
}

func createIndex() {
	query := `
		mutation CreateIndex($collection: String!, $field: String!, $unique: Boolean) {
			createIndex(collection: $collection, field: $field, unique: $unique)
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
		"field":      "category",
		"unique":     false,
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Index created: %v\n", resp.Data["createIndex"])
}

func listIndexes() {
	query := `
		query ListIndexes($collection: String!) {
			listIndexes(collection: $collection) {
				name
				field
				unique
				type
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	indexes := resp.Data["listIndexes"].([]interface{})
	fmt.Printf("✅ Indexes (%d):\n", len(indexes))
	for _, idx := range indexes {
		idxMap := idx.(map[string]interface{})
		fmt.Printf("   - %s on field '%s' (unique: %v)\n",
			idxMap["name"], idxMap["field"], idxMap["unique"])
	}
}

func aggregateData() {
	query := `
		query AggregateProducts($collection: String!, $pipeline: [JSON]) {
			aggregate(collection: $collection, pipeline: $pipeline) {
				results
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
		"pipeline": []interface{}{
			map[string]interface{}{
				"$group": map[string]interface{}{
					"_id":        "$category",
					"totalStock": map[string]interface{}{"$sum": "$stock"},
					"avgPrice":   map[string]interface{}{"$avg": "$price"},
				},
			},
		},
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	result := resp.Data["aggregate"].(map[string]interface{})
	results := result["results"].([]interface{})
	fmt.Printf("✅ Aggregation results:\n")
	for _, r := range results {
		rMap := r.(map[string]interface{})
		fmt.Printf("   - Category: %v, Total Stock: %v, Avg Price: $%.2f\n",
			rMap["_id"], rMap["totalStock"], rMap["avgPrice"])
	}
}

func deleteDocument() {
	query := `
		mutation DeleteProduct($collection: String!, $filter: JSON!) {
			deleteOne(collection: $collection, filter: $filter) {
				deletedCount
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
		"filter": map[string]interface{}{
			"name": "Mouse",
		},
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	result := resp.Data["deleteOne"].(map[string]interface{})
	fmt.Printf("✅ Deleted %v documents\n", result["deletedCount"])
}

func getStats() {
	query := `
		query GetStats($collection: String!) {
			collectionStats(collection: $collection) {
				name
				documentCount
				indexCount
			}
		}
	`

	variables := map[string]interface{}{
		"collection": "products",
	}

	resp, err := executeGraphQL(query, variables)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	stats := resp.Data["collectionStats"].(map[string]interface{})
	fmt.Printf("✅ Collection Stats:\n")
	fmt.Printf("   - Name: %v\n", stats["name"])
	fmt.Printf("   - Documents: %v\n", stats["documentCount"])
	fmt.Printf("   - Indexes: %v\n", stats["indexCount"])
}
