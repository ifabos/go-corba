package main

import (
	"fmt"
	"os"

	"github.com/ifabos/go-corba/corba"
)

func ExampleNS() {
	// Initialize the ORB
	orb := corba.Init()

	// Connect to the naming service
	namingClient, err := orb.ResolveNameService("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to connect to naming service: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected to naming service at localhost:8099")
	defer namingClient.Close()

	// Get a list of all bindings in the root context
	bindings, err := namingClient.List()
	if err != nil {
		fmt.Printf("Failed to list bindings: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Available objects in the naming service:")
	for _, binding := range bindings {
		bindingType := "Object"
		if binding.Type == corba.ContextBinding {
			bindingType = "Context"
		}
		fmt.Printf(" - %s (%s)\n", binding.Name.String(), bindingType)
	}

	// Resolve the calculator by name from the naming service
	calcObj, err := namingClient.Resolve("calculators/basic")
	if err != nil {
		fmt.Printf("Failed to resolve calculator: %v\n", err)
		os.Exit(1)
	}

	// Cast the resolved object to an ObjectRef
	calcRef, ok := calcObj.(*corba.ObjectRef)
	if !ok {
		fmt.Println("Could not cast resolved object to ObjectRef")
		os.Exit(1)
	}

	fmt.Println("\nFound calculator object through naming service")

	// Invoke methods on the calculator
	fmt.Println("\nPerforming calculations:")

	// Add
	result, err := calcRef.Invoke("add", 10.5, 20.7)
	if err != nil {
		fmt.Printf("Add operation failed: %v\n", err)
	} else {
		fmt.Printf("10.5 + 20.7 = %v\n", result)
	}

	// Subtract
	result, err = calcRef.Invoke("subtract", 50.0, 15.5)
	if err != nil {
		fmt.Printf("Subtract operation failed: %v\n", err)
	} else {
		fmt.Printf("50.0 - 15.5 = %v\n", result)
	}

	// Multiply
	result, err = calcRef.Invoke("multiply", 7.0, 8.0)
	if err != nil {
		fmt.Printf("Multiply operation failed: %v\n", err)
	} else {
		fmt.Printf("7.0 * 8.0 = %v\n", result)
	}

	// Divide
	result, err = calcRef.Invoke("divide", 100.0, 4.0)
	if err != nil {
		fmt.Printf("Divide operation failed: %v\n", err)
	} else {
		fmt.Printf("100.0 / 4.0 = %v\n", result)
	}

	// Try division by zero to test error handling
	result, err = calcRef.Invoke("divide", 100.0, 0.0)
	if err != nil {
		fmt.Printf("Division by zero properly handled: %v\n", err)
	} else {
		fmt.Println("Division by zero didn't raise an error!")
	}

	fmt.Println("\nClient operations completed successfully")
}
