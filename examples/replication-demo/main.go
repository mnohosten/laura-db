package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mnohosten/laura-db/pkg/database"
	"github.com/mnohosten/laura-db/pkg/replication"
)

func main() {
	fmt.Println("=== LauraDB Master-Slave Replication Demo ===")
	fmt.Println()

	// Clean up old data
	os.RemoveAll("./replication-demo-data")

	// Demo 1: Basic Master-Slave Replication
	fmt.Println("Demo 1: Basic Master-Slave Replication")
	fmt.Println("---------------------------------------")
	basicReplication()

	fmt.Println()

	// Demo 2: Multiple Slaves
	fmt.Println("Demo 2: Multiple Slaves")
	fmt.Println("-----------------------")
	multipleSlaves()

	fmt.Println()

	// Demo 3: Initial Sync for New Slave
	fmt.Println("Demo 3: Initial Sync for New Slave")
	fmt.Println("-----------------------------------")
	initialSync()

	fmt.Println()

	// Demo 4: Replication Lag Monitoring
	fmt.Println("Demo 4: Replication Lag Monitoring")
	fmt.Println("-----------------------------------")
	replicationLag()

	fmt.Println("\n=== Demo Complete ===")

	// Cleanup
	os.RemoveAll("./replication-demo-data")
}

func basicReplication() {
	// Create master database
	masterDB, err := database.Open(database.DefaultConfig("./replication-demo-data/master"))
	if err != nil {
		log.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	// Create slave database
	slaveDB, err := database.Open(database.DefaultConfig("./replication-demo-data/slave"))
	if err != nil {
		log.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	// Set up replication
	masterConfig := replication.DefaultMasterConfig(masterDB, "./replication-demo-data/oplog.bin")
	slaveConfig := replication.DefaultSlaveConfig("slave1", slaveDB, nil)
	slaveConfig.PollInterval = 100 * time.Millisecond // Fast polling for demo

	pair, err := replication.NewReplicationPair(masterConfig, slaveConfig)
	if err != nil {
		log.Fatalf("Failed to create replication pair: %v", err)
	}
	defer pair.Stop()

	if err := pair.Start(); err != nil {
		log.Fatalf("Failed to start replication: %v", err)
	}

	fmt.Println("✓ Replication started")

	// Insert documents on master
	fmt.Println("✓ Inserting 5 users on master...")
	for i := 1; i <= 5; i++ {
		doc := map[string]interface{}{
			"_id":  fmt.Sprintf("user%d", i),
			"name": fmt.Sprintf("User %d", i),
			"age":  int64(20 + i),
		}

		// Insert on master
		masterDB.Collection("users").InsertOne(doc)

		// Log to oplog for replication
		entry := replication.CreateInsertEntry("default", "users", doc)
		pair.Master.LogOperation(entry)
	}

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify on slave
	count, _ := slaveDB.Collection("users").Count(nil)
	fmt.Printf("✓ Master has 5 users, Slave has %d users\n", count)

	// Read from slave
	docs, _ := slaveDB.Collection("users").Find(nil)
	fmt.Println("✓ Documents on slave:")
	for _, doc := range docs {
		m := doc.ToMap()
		fmt.Printf("  - %s: %v (age: %v)\n", m["_id"], m["name"], m["age"])
	}
}

func multipleSlaves() {
	// Create master database
	masterDB, err := database.Open(database.DefaultConfig("./replication-demo-data/master2"))
	if err != nil {
		log.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	// Create master
	masterConfig := replication.DefaultMasterConfig(masterDB, "./replication-demo-data/oplog2.bin")
	master, err := replication.NewMaster(masterConfig)
	if err != nil {
		log.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		log.Fatalf("Failed to start master: %v", err)
	}

	// Create 3 slaves
	slaves := make([]*replication.Slave, 3)
	for i := 0; i < 3; i++ {
		slaveDB, err := database.Open(database.DefaultConfig(fmt.Sprintf("./replication-demo-data/slave%d", i+1)))
		if err != nil {
			log.Fatalf("Failed to open slave database: %v", err)
		}
		defer slaveDB.Close()

		client := replication.NewLocalMasterClient(master)
		config := replication.DefaultSlaveConfig(fmt.Sprintf("slave%d", i+1), slaveDB, client)
		config.PollInterval = 100 * time.Millisecond

		slave, err := replication.NewSlave(config)
		if err != nil {
			log.Fatalf("Failed to create slave: %v", err)
		}
		defer slave.Stop()

		if err := slave.Start(); err != nil {
			log.Fatalf("Failed to start slave: %v", err)
		}

		slaves[i] = slave
	}

	fmt.Println("✓ Master with 3 slaves started")

	// Insert data
	for i := 1; i <= 10; i++ {
		doc := map[string]interface{}{
			"_id":   fmt.Sprintf("product%d", i),
			"name":  fmt.Sprintf("Product %d", i),
			"price": int64(100 * i),
		}

		masterDB.Collection("products").InsertOne(doc)
		entry := replication.CreateInsertEntry("default", "products", doc)
		master.LogOperation(entry)
	}

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Show slave status
	allSlaves := master.GetAllSlaves()
	fmt.Printf("✓ Master reports %d active slaves:\n", len(allSlaves))
	for _, info := range allSlaves {
		fmt.Printf("  - %s: last OpID=%d, lag=%v\n", info.ID, info.LastOpID, info.Lag)
	}
}

func initialSync() {
	// Create master with existing data
	masterDB, err := database.Open(database.DefaultConfig("./replication-demo-data/master3"))
	if err != nil {
		log.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	masterConfig := replication.DefaultMasterConfig(masterDB, "./replication-demo-data/oplog3.bin")
	master, err := replication.NewMaster(masterConfig)
	if err != nil {
		log.Fatalf("Failed to create master: %v", err)
	}
	defer master.Stop()

	if err := master.Start(); err != nil {
		log.Fatalf("Failed to start master: %v", err)
	}

	// Pre-populate master
	fmt.Println("✓ Populating master with 20 documents...")
	for i := 1; i <= 20; i++ {
		doc := map[string]interface{}{
			"_id":   fmt.Sprintf("item%d", i),
			"value": int64(i * 10),
		}
		masterDB.Collection("items").InsertOne(doc)
		entry := replication.CreateInsertEntry("default", "items", doc)
		master.LogOperation(entry)
	}

	fmt.Printf("✓ Master oplog has %d operations\n", master.GetCurrentOpID())

	// Create new slave and perform initial sync
	slaveDB, err := database.Open(database.DefaultConfig("./replication-demo-data/slave-new"))
	if err != nil {
		log.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	client := replication.NewLocalMasterClient(master)
	slaveConfig := replication.DefaultSlaveConfig("slave-new", slaveDB, client)
	slave, err := replication.NewSlave(slaveConfig)
	if err != nil {
		log.Fatalf("Failed to create slave: %v", err)
	}

	// Perform initial sync
	fmt.Println("✓ Performing initial sync...")
	ctx := context.Background()
	if err := slave.InitialSync(ctx); err != nil {
		log.Fatalf("Initial sync failed: %v", err)
	}

	count, _ := slaveDB.Collection("items").Count(nil)
	fmt.Printf("✓ Initial sync complete: slave has %d documents\n", count)
	fmt.Printf("✓ Last applied OpID: %d\n", slave.GetLastAppliedOpID())
}

func replicationLag() {
	// Create master and slave
	masterDB, err := database.Open(database.DefaultConfig("./replication-demo-data/master4"))
	if err != nil {
		log.Fatalf("Failed to open master database: %v", err)
	}
	defer masterDB.Close()

	slaveDB, err := database.Open(database.DefaultConfig("./replication-demo-data/slave4"))
	if err != nil {
		log.Fatalf("Failed to open slave database: %v", err)
	}
	defer slaveDB.Close()

	masterConfig := replication.DefaultMasterConfig(masterDB, "./replication-demo-data/oplog4.bin")
	slaveConfig := replication.DefaultSlaveConfig("slave4", slaveDB, nil)
	slaveConfig.PollInterval = 500 * time.Millisecond // Slower for lag demo

	pair, err := replication.NewReplicationPair(masterConfig, slaveConfig)
	if err != nil {
		log.Fatalf("Failed to create replication pair: %v", err)
	}
	defer pair.Stop()

	if err := pair.Start(); err != nil {
		log.Fatalf("Failed to start replication: %v", err)
	}

	fmt.Println("✓ Monitoring replication lag...")

	// Insert many documents quickly
	for i := 1; i <= 100; i++ {
		doc := map[string]interface{}{
			"_id":  fmt.Sprintf("doc%d", i),
			"data": fmt.Sprintf("Data %d", i),
		}
		masterDB.Collection("logs").InsertOne(doc)
		entry := replication.CreateInsertEntry("default", "logs", doc)
		pair.Master.LogOperation(entry)

		if i%25 == 0 {
			// Check lag periodically
			masterOpID := pair.Master.GetCurrentOpID()
			slaveOpID := pair.Slave.GetLastAppliedOpID()
			lag := pair.Slave.GetLag(masterOpID)

			fmt.Printf("  After %d docs: Master OpID=%d, Slave OpID=%d, Lag=%v\n",
				i, masterOpID, slaveOpID, lag)

			time.Sleep(300 * time.Millisecond) // Give slave time to catch up
		}
	}

	// Wait for full sync
	fmt.Println("✓ Waiting for slave to catch up...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pair.Master.WaitForSlaves(ctx, pair.Master.GetCurrentOpID(), 10*time.Second); err != nil {
		fmt.Printf("⚠ Warning: %v\n", err)
	} else {
		fmt.Println("✓ Slave fully synchronized!")
	}

	finalCount, _ := slaveDB.Collection("logs").Count(nil)
	fmt.Printf("✓ Final state: Slave has %d/100 documents\n", finalCount)
}
