package examples

import (
	"fmt"
	"log"
	"os"

	gocorba "github.com/ifabos/go-corba"
)

// NameServiceClient shows how to work with a CORBA Naming Service
// using IOR (Interoperable Object Reference) strings
func NameServiceClient() {
	// Initialize the ORB
	orb := gocorba.Init()

	// Create a CORBA client
	client := orb.CreateClient()

	// In a real CORBA implementation, you can typically:
	// 1. Read an IOR from a file, environment variable, or command line
	// 2. Convert the IOR string to an object reference
	// 3. Narrow the reference to a specific interface type

	// Example of reading an IOR from an environment variable
	iorString := os.Getenv("NAME_SERVICE_IOR")
	if iorString == "" {
		// For demonstration, use a placeholder IOR
		iorString = "IOR:000000000000002249444c3a6f6d672e6f72672f436f734e616d696e672f4e616d696e67436f6e746578743a312e30000000000002000000000000007c000102000000000e3139322e3136382e312e32303000000bf3000000004a4143000000000000010000000000000024000000000001000000000000000100000001000000140000000100000001000100000000000901010000000000"
		fmt.Println("Using example IOR (normally this would come from environment)")
	}

	fmt.Println("Working with IOR:", iorString)

	// Note: In the current SDK implementation, this is a placeholder.
	// A full CORBA implementation would need to parse the IOR string and
	// create an appropriate object reference.

	// Conceptual example - this function doesn't actually exist in the current SDK
	// nameService := client.StringToObject(iorString)

	// Instead, for this example:
	// Connect to the service using information we would extract from the IOR
	serverHost := "naming.example.com"
	serverPort := 8099

	if err := client.Connect(serverHost, serverPort); err != nil {
		log.Fatalf("Failed to connect to naming service: %v", err)
	}

	// Get a reference to the naming service
	nameServiceRef, err := client.GetObject("NameService", serverHost, serverPort)
	if err != nil {
		log.Fatalf("Failed to get NameService reference: %v", err)
	}

	// In a real CORBA application with full IDL support, we would use strongly-typed interfaces
	// For demonstration, we use the dynamic invocation style

	// Resolve a name using the naming service
	fmt.Println("Looking up 'Banking/AccountManager' in naming service")

	// Create a name component sequence (this would be handled by IDL-generated code)
	nameComponents := []interface{}{
		map[string]interface{}{
			"id":   "Banking",
			"kind": "",
		},
		map[string]interface{}{
			"id":   "AccountManager",
			"kind": "",
		},
	}

	// Invoke the resolve method on the naming service
	/*objectRef*/
	_, err = nameServiceRef.Invoke("resolve", nameComponents)
	if err != nil {
		log.Fatalf("Failed to resolve name: %v", err)
	}

	fmt.Println("Successfully resolved object reference")

	// In a real application with IDL support, we would narrow the reference to a specific type
	// accountManager := AccountManagerHelper.narrow(objectRef)

	// For demonstration, we'll just invoke methods directly
	fmt.Println("Invoking methods on the resolved object")

	// Example method call to create an account
	customerID := "customer456"
	initialDeposit := 1000.0
	result, err := nameServiceRef.Invoke("createAccount", customerID, initialDeposit)
	if err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}

	fmt.Printf("Account created with ID: %v\n", result)

	// Clean up
	if err := client.Disconnect(serverHost, serverPort); err != nil {
		fmt.Printf("Warning: Error during disconnect: %v\n", err)
	}

	orb.Shutdown(true)
	fmt.Println("CORBA client shutdown complete")
}
