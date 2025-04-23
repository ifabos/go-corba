// Package examples provides usage examples for the go-corba SDK.
package examples

import (
	"fmt"
	"log"

	gocorba "github.com/ifabos/go-corba"
)

// BasicClientExample demonstrates the fundamental usage of a CORBA client
func BasicClientExample() {
	// Initialize the ORB
	orb := gocorba.Init()
	if !orb.IsInitialized() {
		log.Fatal("Failed to initialize ORB")
	}

	// Create a CORBA client
	client := orb.CreateClient()

	// Connect to a CORBA server
	serverHost := "localhost"
	serverPort := 8099
	if err := client.Connect(serverHost, serverPort); err != nil {
		log.Fatalf("Failed to connect to CORBA server: %v", err)
	}

	// Get a reference to a remote object
	objectName := "Calculator"
	objectRef, err := client.GetObject(objectName, serverHost, serverPort)
	if err != nil {
		log.Fatalf("Failed to get object reference: %v", err)
	}

	// Invoke a method on the remote object
	methodName := "Add"
	args := []interface{}{5, 3}
	result, err := objectRef.Invoke(methodName, args...)
	if err != nil {
		log.Fatalf("Failed to invoke method: %v", err)
	}

	// Process the result
	fmt.Printf("Result of %s.%s(%v): %v\n", objectName, methodName, args, result)

	// Disconnect from the server
	if err := client.Disconnect(serverHost, serverPort); err != nil {
		log.Fatalf("Failed to disconnect from server: %v", err)
	}

	// Shut down the ORB when done
	orb.Shutdown(true)
}
