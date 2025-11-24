package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/replication"
)

func main() {
	fmt.Println("=== LauraDB Read Preference Demo ===")
	fmt.Println()

	// Demo 1: Primary reads (default and safest)
	fmt.Println("Demo 1: Primary Reads (Default)")
	fmt.Println("--------------------------------")
	demo1PrimaryReads()
	fmt.Println()

	// Demo 2: Secondary reads (read scaling)
	fmt.Println("Demo 2: Secondary Reads (Read Scaling)")
	fmt.Println("---------------------------------------")
	demo2SecondaryReads()
	fmt.Println()

	// Demo 3: PrimaryPreferred reads (high availability)
	fmt.Println("Demo 3: PrimaryPreferred Reads (High Availability)")
	fmt.Println("--------------------------------------------------")
	demo3PrimaryPreferred()
	fmt.Println()

	// Demo 4: SecondaryPreferred reads (read offloading)
	fmt.Println("Demo 4: SecondaryPreferred Reads (Read Offloading)")
	fmt.Println("--------------------------------------------------")
	demo4SecondaryPreferred()
	fmt.Println()

	// Demo 5: Max staleness configuration
	fmt.Println("Demo 5: Max Staleness Configuration")
	fmt.Println("-----------------------------------")
	demo5MaxStaleness()
	fmt.Println()

	fmt.Println("All demos completed successfully!")
}

func demo1PrimaryReads() {
	// Create databases
	primaryDB, err := database.Open(database.DefaultConfig("/tmp/laura-rp-demo1-primary"))
	if err != nil {
		panic(err)
	}
	defer primaryDB.Close()
	defer os.RemoveAll("/tmp/laura-rp-demo1-primary")

	// Create replica set
	rsConfig := replication.DefaultReplicaSetConfig("rs0", "primary", primaryDB, "/tmp/laura-rp-demo1-primary/oplog")
	rs, err := replication.NewReplicaSet(rsConfig)
	if err != nil {
		panic(err)
	}

	// Start as primary
	if err := rs.Start(); err != nil {
		panic(err)
	}
	defer rs.Stop()

	if err := rs.BecomePrimary(); err != nil {
		panic(err)
	}

	// Insert test data
	coll, _ := primaryDB.CreateCollection("users")
	coll.InsertOne(map[string]interface{}{
		"name": "Alice",
		"role": "admin",
		"age":  int64(30),
	})

	// Create read router
	router := replication.NewReadRouter(rs)

	// Read with primary preference (default)
	ctx := context.Background()
	pref := replication.Primary()

	doc, err := router.ReadDocument(ctx, "users", map[string]interface{}{"name": "Alice"}, pref)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Read preference: %s\n", pref)
	fmt.Printf("Selected node: primary\n")
	fmt.Printf("Document: name=%s, role=%s, age=%v\n", doc["name"], doc["role"], doc["age"])
	fmt.Println("✓ Primary reads ensure latest data (strongest consistency)")
}

func demo2SecondaryReads() {
	// Create primary database
	primaryDB, err := database.Open(database.DefaultConfig("/tmp/laura-rp-demo2-primary"))
	if err != nil {
		panic(err)
	}
	defer primaryDB.Close()
	defer os.RemoveAll("/tmp/laura-rp-demo2-primary")

	// Create replica set with secondaries
	rsConfig := replication.DefaultReplicaSetConfig("rs0", "primary", primaryDB, "/tmp/laura-rp-demo2-primary/oplog")
	rs, err := replication.NewReplicaSet(rsConfig)
	if err != nil {
		panic(err)
	}

	if err := rs.Start(); err != nil {
		panic(err)
	}
	defer rs.Stop()

	if err := rs.BecomePrimary(); err != nil {
		panic(err)
	}

	// Add secondary members
	rs.AddMember("secondary1", 1, true)
	rs.AddMember("secondary2", 1, true)
	rs.UpdateMemberHeartbeat("secondary1", 100)
	rs.UpdateMemberHeartbeat("secondary2", 100)

	// Insert test data
	coll, _ := primaryDB.CreateCollection("products")
	coll.InsertOne(map[string]interface{}{
		"name":  "Laptop",
		"price": int64(999),
		"stock": int64(50),
	})

	// Create read router
	router := replication.NewReadRouter(rs)

	// Read from secondary
	ctx := context.Background()
	pref := replication.Secondary()

	// Get selected node (for demonstration)
	selectedNode, _ := router.GetSelectedNode(ctx, pref)

	doc, err := router.ReadDocument(ctx, "products", map[string]interface{}{"name": "Laptop"}, pref)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Read preference: %s\n", pref)
	fmt.Printf("Selected node: %s (randomly chosen from healthy secondaries)\n", selectedNode)
	fmt.Printf("Document: name=%s, price=%v, stock=%v\n", doc["name"], doc["price"], doc["stock"])
	fmt.Println("✓ Secondary reads offload query traffic from primary")
}

func demo3PrimaryPreferred() {
	// Create primary database
	primaryDB, err := database.Open(database.DefaultConfig("/tmp/laura-rp-demo3-primary"))
	if err != nil {
		panic(err)
	}
	defer primaryDB.Close()
	defer os.RemoveAll("/tmp/laura-rp-demo3-primary")

	// Create replica set
	rsConfig := replication.DefaultReplicaSetConfig("rs0", "primary", primaryDB, "/tmp/laura-rp-demo3-primary/oplog")
	rs, err := replication.NewReplicaSet(rsConfig)
	if err != nil {
		panic(err)
	}

	if err := rs.Start(); err != nil {
		panic(err)
	}
	defer rs.Stop()

	if err := rs.BecomePrimary(); err != nil {
		panic(err)
	}

	// Add secondary members
	rs.AddMember("secondary1", 1, true)
	rs.UpdateMemberHeartbeat("secondary1", 100)

	// Insert test data
	coll, _ := primaryDB.CreateCollection("orders")
	coll.InsertOne(map[string]interface{}{
		"order_id": "ORD-001",
		"customer": "Bob",
		"total":    int64(250),
	})

	// Create read router
	router := replication.NewReadRouter(rs)

	// Read with primary preferred
	ctx := context.Background()
	pref := replication.PrimaryPreferred()

	selectedNode, _ := router.GetSelectedNode(ctx, pref)

	doc, err := router.ReadDocument(ctx, "orders", map[string]interface{}{"order_id": "ORD-001"}, pref)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Read preference: %s\n", pref)
	fmt.Printf("Selected node: %s (primary available)\n", selectedNode)
	fmt.Printf("Document: order_id=%s, customer=%s, total=%v\n", doc["order_id"], doc["customer"], doc["total"])
	fmt.Println("✓ PrimaryPreferred reads from primary when available")
	fmt.Println("  Falls back to secondary if primary is down (high availability)")
}

func demo4SecondaryPreferred() {
	// Create primary database
	primaryDB, err := database.Open(database.DefaultConfig("/tmp/laura-rp-demo4-primary"))
	if err != nil {
		panic(err)
	}
	defer primaryDB.Close()
	defer os.RemoveAll("/tmp/laura-rp-demo4-primary")

	// Create replica set
	rsConfig := replication.DefaultReplicaSetConfig("rs0", "primary", primaryDB, "/tmp/laura-rp-demo4-primary/oplog")
	rs, err := replication.NewReplicaSet(rsConfig)
	if err != nil {
		panic(err)
	}

	if err := rs.Start(); err != nil {
		panic(err)
	}
	defer rs.Stop()

	if err := rs.BecomePrimary(); err != nil {
		panic(err)
	}

	// Add secondary members
	rs.AddMember("secondary1", 1, true)
	rs.AddMember("secondary2", 1, true)
	rs.UpdateMemberHeartbeat("secondary1", 100)
	rs.UpdateMemberHeartbeat("secondary2", 100)

	// Insert test data
	coll, _ := primaryDB.CreateCollection("analytics")
	coll.InsertOne(map[string]interface{}{
		"metric":    "page_views",
		"value":     int64(1000),
		"timestamp": time.Now(),
	})

	// Create read router
	router := replication.NewReadRouter(rs)

	// Read with secondary preferred (good for analytics queries)
	ctx := context.Background()
	pref := replication.SecondaryPreferred()

	selectedNode, _ := router.GetSelectedNode(ctx, pref)

	doc, err := router.ReadDocument(ctx, "analytics", map[string]interface{}{"metric": "page_views"}, pref)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Read preference: %s\n", pref)
	fmt.Printf("Selected node: %s (secondary preferred for analytics)\n", selectedNode)
	fmt.Printf("Document: metric=%s, value=%v\n", doc["metric"], doc["value"])
	fmt.Println("✓ SecondaryPreferred offloads analytics queries to secondaries")
	fmt.Println("  Falls back to primary if no secondaries available")
}

func demo5MaxStaleness() {
	// Create primary database
	primaryDB, err := database.Open(database.DefaultConfig("/tmp/laura-rp-demo5-primary"))
	if err != nil {
		panic(err)
	}
	defer primaryDB.Close()
	defer os.RemoveAll("/tmp/laura-rp-demo5-primary")

	// Create replica set
	rsConfig := replication.DefaultReplicaSetConfig("rs0", "primary", primaryDB, "/tmp/laura-rp-demo5-primary/oplog")
	rs, err := replication.NewReplicaSet(rsConfig)
	if err != nil {
		panic(err)
	}

	if err := rs.Start(); err != nil {
		panic(err)
	}
	defer rs.Stop()

	if err := rs.BecomePrimary(); err != nil {
		panic(err)
	}

	// Add secondary members with different lag
	rs.AddMember("secondary1", 1, true) // Fresh secondary
	rs.AddMember("secondary2", 1, true) // Stale secondary

	// secondary1 is caught up (OpID 1000)
	rs.UpdateMemberHeartbeat("secondary1", 1000)

	// secondary2 is lagging behind (OpID 100) - ~900ms lag
	rs.UpdateMemberHeartbeat("secondary2", 100)

	// Insert test data
	coll, _ := primaryDB.CreateCollection("inventory")
	coll.InsertOne(map[string]interface{}{
		"product": "Widget",
		"stock":   int64(100),
	})

	// Create read router
	router := replication.NewReadRouter(rs)

	// Read with max staleness constraint
	ctx := context.Background()
	pref := replication.Secondary().WithMaxStaleness(1) // Max 1 second lag

	selectedNode, _ := router.GetSelectedNode(ctx, pref)

	doc, err := router.ReadDocument(ctx, "inventory", map[string]interface{}{"product": "Widget"}, pref)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Read preference: %s\n", pref)
	fmt.Printf("Selected node: %s (only fresh secondaries with <1s lag)\n", selectedNode)
	fmt.Printf("Document: product=%s, stock=%v\n", doc["product"], doc["stock"])
	fmt.Println("✓ MaxStaleness ensures reads are from relatively fresh secondaries")
	fmt.Println("  Excludes secondaries that are too far behind")
}
