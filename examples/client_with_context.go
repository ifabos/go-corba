package examples

import (
	"fmt"
	"log"

	gocorba "github.com/ifabos/go-corba"
)

// ClientWithContextExample demonstrates using CORBA client with contexts
func ClientWithContextExample() {
	// Initialize the ORB
	orb := gocorba.Init()

	// Create a custom context for the client operations
	ctx := gocorba.NewContext()

	// Set properties in the context
	ctx.Set("transaction_id", "tx-12345")
	ctx.Set("security_level", "high")
	ctx.Set("timeout", 30000) // timeout in milliseconds

	// Create a CORBA client
	client := orb.CreateClient()

	// Connect to a CORBA server
	serverHost := "localhost"
	serverPort := 8099
	if err := client.Connect(serverHost, serverPort); err != nil {
		log.Fatalf("Failed to connect to CORBA server: %v", err)
	}

	// Get a reference to a remote object
	objectName := "BankAccount"
	objectRef, err := client.GetObject(objectName, serverHost, serverPort)
	if err != nil {
		log.Fatalf("Failed to get object reference: %v", err)
	}

	// In a real implementation with full CORBA support, the context would be passed
	// with the invocation. This is a placeholder showing the concept.
	fmt.Println("Using context with properties:")
	for key, value := range ctx.GetAll() {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Example invocation that would use the context
	methodName := "transfer"
	args := []interface{}{
		"account123",
		"account456",
		1000.00,
	}
	fmt.Printf("Invoking %s.%s with context\n", objectName, methodName)

	// In a complete CORBA implementation, the context would be passed to the invoke call
	result, err := objectRef.Invoke(methodName, args...)
	if err != nil {
		log.Fatalf("Failed to invoke method: %v", err)
	}

	fmt.Printf("Transfer result: %v\n", result)

	// Disconnect from the server
	client.Disconnect(serverHost, serverPort)

	// Shut down the ORB
	orb.Shutdown(true)
}
