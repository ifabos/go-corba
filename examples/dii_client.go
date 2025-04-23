package examples

import (
	"fmt"
	"os"

	"github.com/ifabos/go-corba/corba"
)

func InvokeDII() {
	// Initialize the ORB
	orb := corba.Init()

	// Connect to a server
	client := orb.CreateClient()
	err := client.Connect("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to server")

	// Get an object reference
	calcRef, err := client.GetObject("Calculator", "localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to get object reference: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Got reference to Calculator object")

	// Create a request using DII
	fmt.Println("\nUsing Dynamic Invocation Interface (DII):")

	// Method 1: Use the Request directly
	fmt.Println("\nMethod 1 - Using Request directly:")
	request := corba.NewRequest(calcRef, "add")

	// Add parameters
	err = request.AddParameter("a", 10.5, corba.FlagIn)
	if err != nil {
		fmt.Printf("Failed to add parameter: %v\n", err)
		os.Exit(1)
	}

	err = request.AddParameter("b", 20.7, corba.FlagIn)
	if err != nil {
		fmt.Printf("Failed to add parameter: %v\n", err)
		os.Exit(1)
	}

	// Invoke the request
	err = request.Invoke()
	if err != nil {
		fmt.Printf("Invocation failed: %v\n", err)
		os.Exit(1)
	}

	// Get the result
	result, err := request.GetResponse()
	if err != nil {
		fmt.Printf("Failed to get response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Result: 10.5 + 20.7 = %v\n", result)

	// Method 2: Use the ORB's request processor
	fmt.Println("\nMethod 2 - Using ORB's RequestProcessor:")
	processor := orb.GetRequestProcessor()

	// Create named values for parameters
	params := []*corba.NamedValue{
		{Name: "a", Value: 50.0, Flags: corba.FlagIn},
		{Name: "b", Value: 15.5, Flags: corba.FlagIn},
	}

	// Create a result placeholder
	resultVal := &corba.NamedValue{Name: "_result_", Value: nil}

	// Create the request
	subtractRequest := processor.CreateRequest(
		calcRef,                 // Target object
		"subtract",              // Operation name
		params,                  // Parameters
		resultVal,               // Result
		nil,                     // No exceptions
		orb.GetDefaultContext(), // Context
	)

	// Invoke the request
	err = subtractRequest.Invoke()
	if err != nil {
		fmt.Printf("Invocation failed: %v\n", err)
		os.Exit(1)
	}

	// Get the result
	subtractResult, err := subtractRequest.GetResponse()
	if err != nil {
		fmt.Printf("Failed to get response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Result: 50.0 - 15.5 = %v\n", subtractResult)

	// Method 3: One-way invocation (no response expected)
	fmt.Println("\nMethod 3 - One-way invocation:")
	logRequest := corba.NewRequest(calcRef, "log")
	err = logRequest.AddParameter("message", "DII test log message", corba.FlagIn)
	if err != nil {
		fmt.Printf("Failed to add parameter: %v\n", err)
		os.Exit(1)
	}

	// Send without waiting for response
	err = logRequest.Invoke()
	if err != nil {
		fmt.Printf("One-way invocation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("One-way invocation sent successfully")

	// Method 4: Deferred synchronous invocation
	fmt.Println("\nMethod 4 - Deferred synchronous invocation:")
	multiplyRequest := corba.NewRequest(calcRef, "multiply")
	err = multiplyRequest.AddParameter("a", 6.0, corba.FlagIn)
	if err != nil {
		fmt.Printf("Failed to add parameter: %v\n", err)
		os.Exit(1)
	}

	err = multiplyRequest.AddParameter("b", 7.0, corba.FlagIn)
	if err != nil {
		fmt.Printf("Failed to add parameter: %v\n", err)
		os.Exit(1)
	}

	// Send deferred request
	err = multiplyRequest.SendDeferred()
	if err != nil {
		fmt.Printf("Deferred invocation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Deferred request sent, doing other work...")

	// Poll until response is ready
	for !multiplyRequest.PollResponse() {
		fmt.Println("Waiting for response...")
		// In a real application, we would do other work here
	}

	// Get response
	multiplyResult, err := multiplyRequest.GetResponse()
	if err != nil {
		fmt.Printf("Failed to get deferred response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Result: 6.0 * 7.0 = %v\n", multiplyResult)

	// Disconnect from the server
	err = client.Disconnect("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to disconnect: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nDisconnected from server")
}
