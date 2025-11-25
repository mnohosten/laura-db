package main

import (
	"fmt"
	"log"

	"github.com/mnohosten/laura-db/pkg/connstring"
)

func main() {
	fmt.Println("LauraDB Connection String Parsing Demo")
	fmt.Println("========================================\n")

	// Demo 1: Basic connection string
	fmt.Println("Demo 1: Basic Connection String")
	fmt.Println("--------------------------------")
	demo1()
	fmt.Println()

	// Demo 2: Connection string with authentication
	fmt.Println("Demo 2: Connection String with Authentication")
	fmt.Println("----------------------------------------------")
	demo2()
	fmt.Println()

	// Demo 3: Multiple hosts (replica set)
	fmt.Println("Demo 3: Multiple Hosts (Replica Set)")
	fmt.Println("-------------------------------------")
	demo3()
	fmt.Println()

	// Demo 4: Connection string with options
	fmt.Println("Demo 4: Connection String with Options")
	fmt.Println("---------------------------------------")
	demo4()
	fmt.Println()

	// Demo 5: Complex real-world connection string
	fmt.Println("Demo 5: Complex Real-World Connection String")
	fmt.Println("---------------------------------------------")
	demo5()
	fmt.Println()
}

func demo1() {
	uri := "laura://localhost:8080/mydb"
	fmt.Printf("URI: %s\n", uri)

	cs, err := connstring.Parse(uri)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("Scheme: %s\n", cs.Scheme)
	fmt.Printf("Host: %s:%d\n", cs.Hosts[0].Host, cs.Hosts[0].Port)
	fmt.Printf("Database: %s\n", cs.Database)
}

func demo2() {
	uri := "mongodb://admin:P%40ssw0rd@localhost:27017/production"
	fmt.Printf("URI: %s\n", uri)

	cs, err := connstring.Parse(uri)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("Scheme: %s\n", cs.Scheme)
	fmt.Printf("Host: %s:%d\n", cs.Hosts[0].Host, cs.Hosts[0].Port)
	fmt.Printf("Database: %s\n", cs.Database)
	fmt.Printf("Username: %s\n", cs.Options.Username)
	fmt.Printf("Password: %s\n", maskPassword(cs.Options.Password))
	fmt.Printf("Has Authentication: %v\n", cs.HasAuthentication())
}

func demo3() {
	uri := "laura://host1:8080,host2:8081,host3:8082/mydb?replicaSet=rs0"
	fmt.Printf("URI: %s\n", uri)

	cs, err := connstring.Parse(uri)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("Scheme: %s\n", cs.Scheme)
	fmt.Printf("Number of hosts: %d\n", len(cs.Hosts))
	for i, host := range cs.Hosts {
		fmt.Printf("  Host %d: %s:%d\n", i+1, host.Host, host.Port)
	}
	fmt.Printf("Database: %s\n", cs.Database)
	fmt.Printf("Replica Set: %s\n", cs.Options.ReplicaSet)
}

func demo4() {
	uri := "laura://localhost:8080/mydb?timeout=10000&maxConnections=200&tls=true&appName=MyApp"
	fmt.Printf("URI: %s\n", uri)

	cs, err := connstring.Parse(uri)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("Scheme: %s\n", cs.Scheme)
	fmt.Printf("Host: %s:%d\n", cs.Hosts[0].Host, cs.Hosts[0].Port)
	fmt.Printf("Database: %s\n", cs.Database)
	fmt.Printf("Options:\n")
	fmt.Printf("  Timeout: %v\n", cs.Options.Timeout)
	fmt.Printf("  Max Connections: %d\n", cs.Options.MaxConnections)
	fmt.Printf("  TLS: %v\n", cs.Options.TLS)
	fmt.Printf("  App Name: %s\n", cs.Options.AppName)
}

func demo5() {
	uri := "mongodb://admin:SecretPass123@primary.example.com:27017,secondary1.example.com:27017,secondary2.example.com:27017/production?replicaSet=rs0&readPreference=secondaryPreferred&maxConnections=500&timeout=30000&tls=true&tlsInsecure=false&appName=ProductionApp&retryWrites=true&retryReads=true"
	fmt.Printf("URI: %s\n", uri)

	cs, err := connstring.Parse(uri)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	fmt.Printf("\nParsed Connection String:\n")
	fmt.Printf("  Scheme: %s\n", cs.Scheme)
	fmt.Printf("  Authentication: %v\n", cs.HasAuthentication())
	if cs.HasAuthentication() {
		fmt.Printf("    Username: %s\n", cs.Options.Username)
		fmt.Printf("    Password: %s\n", maskPassword(cs.Options.Password))
	}
	fmt.Printf("  Hosts (%d):\n", len(cs.Hosts))
	for i, host := range cs.Hosts {
		fmt.Printf("    %d. %s:%d\n", i+1, host.Host, host.Port)
	}
	fmt.Printf("  Database: %s\n", cs.Database)
	fmt.Printf("  Replica Set: %s\n", cs.Options.ReplicaSet)
	fmt.Printf("  Read Preference: %s\n", cs.Options.ReadPreference)
	fmt.Printf("  Write Concern: %s\n", cs.Options.WriteConcern)
	fmt.Printf("  Max Connections: %d\n", cs.Options.MaxConnections)
	fmt.Printf("  Timeout: %v\n", cs.Options.Timeout)
	fmt.Printf("  TLS: %v\n", cs.Options.TLS)
	fmt.Printf("  TLS Insecure: %v\n", cs.Options.TLSInsecure)
	fmt.Printf("  App Name: %s\n", cs.Options.AppName)
	fmt.Printf("  Retry Writes: %v\n", cs.Options.RetryWrites)
	fmt.Printf("  Retry Reads: %v\n", cs.Options.RetryReads)

	// Demonstrate String() method
	fmt.Printf("\nReconstructed URI:\n")
	fmt.Printf("  %s\n", cs.String())

	// Get first host
	firstHost := cs.GetFirstHost()
	fmt.Printf("\nFirst Host: %s:%d\n", firstHost.Host, firstHost.Port)
}

func maskPassword(password string) string {
	if password == "" {
		return ""
	}
	return "***masked***"
}
