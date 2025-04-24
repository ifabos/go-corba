// Package examples contains various examples of using the CORBA implementation
package examples

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ifabos/go-corba/corba"
)

// SimpleCalculator is a basic calculator implementation for demonstrating interceptors
type InterceptorCalculator struct{}

// Dispatch implements the servant interface
func (c *InterceptorCalculator) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	case "add":
		if len(args) != 2 {
			return nil, fmt.Errorf("add requires exactly 2 arguments")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, corba.BAD_PARAM(1, corba.CompletionStatusNo)
		}
		return a + b, nil

	case "subtract":
		if len(args) != 2 {
			return nil, fmt.Errorf("subtract requires exactly 2 arguments")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, corba.BAD_PARAM(1, corba.CompletionStatusNo)
		}
		return a - b, nil

	case "divide":
		if len(args) != 2 {
			return nil, fmt.Errorf("divide requires exactly 2 arguments")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, corba.BAD_PARAM(1, corba.CompletionStatusNo)
		}
		if b == 0 {
			// Return a user-defined exception
			return nil, corba.NewCORBAUserException("DivByZero", "IDL:examples/DivByZero:1.0")
		}
		return a / b, nil

	case "slowOperation":
		// Simulate a slow operation
		time.Sleep(1 * time.Second)
		return "Operation completed", nil

	case "secureOperation":
		// This operation requires authentication
		return "Secure operation completed", nil

	default:
		return nil, corba.BAD_OPERATION(1, corba.CompletionStatusNo)
	}
}

// CustomServerInterceptor demonstrates how to create a custom interceptor
type CustomServerInterceptor struct{}

func (i *CustomServerInterceptor) Name() string {
	return "CustomServerInterceptor"
}

func (i *CustomServerInterceptor) ReceiveRequest(info *corba.RequestInfo) error {
	log.Printf("[CUSTOM] Received request for %s operation", info.Operation)

	// Modify arguments for demonstration (add 10 to the first argument if it's a number)
	if info.Operation == "add" && len(info.Arguments) > 0 {
		if val, ok := info.Arguments[0].(float64); ok {
			info.Arguments[0] = val + 10
			log.Printf("[CUSTOM] Modified first argument: %v", info.Arguments[0])
		}
	}
	return nil
}

func (i *CustomServerInterceptor) SendReply(info *corba.RequestInfo) error {
	log.Printf("[CUSTOM] Sending reply for %s operation", info.Operation)
	return nil
}

func (i *CustomServerInterceptor) SendException(info *corba.RequestInfo, ex corba.Exception) error {
	log.Printf("[CUSTOM] Exception in %s: %s", info.Operation, ex.Error())
	return nil
}

// CustomClientInterceptor demonstrates how to create a custom client interceptor
type CustomClientInterceptor struct{}

func (i *CustomClientInterceptor) Name() string {
	return "CustomClientInterceptor"
}

func (i *CustomClientInterceptor) SendRequest(info *corba.RequestInfo) error {
	log.Printf("[CUSTOM-CLIENT] Sending request for %s operation", info.Operation)
	return nil
}

func (i *CustomClientInterceptor) ReceiveReply(info *corba.RequestInfo) error {
	log.Printf("[CUSTOM-CLIENT] Received reply for %s operation", info.Operation)

	// For demonstration, modify the result if it's a number
	if result, ok := info.Result.(float64); ok {
		info.Result = result * 2
		log.Printf("[CUSTOM-CLIENT] Modified result: %v", info.Result)
	}
	return nil
}

func (i *CustomClientInterceptor) ReceiveException(info *corba.RequestInfo, ex corba.Exception) error {
	log.Printf("[CUSTOM-CLIENT] Received exception for %s: %s", info.Operation, ex.Error())
	return nil
}

func (i *CustomClientInterceptor) ReceiveOther(info *corba.RequestInfo) error {
	log.Printf("[CUSTOM-CLIENT] Received other response for %s", info.Operation)
	return nil
}

// RunInterceptorServer runs a server with various interceptors configured
func RunInterceptorServer() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a server on localhost:8099
	server, err := orb.CreateServer("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Register server interceptors
	orb.RegisterServerRequestInterceptor(&CustomServerInterceptor{})
	orb.RegisterServerRequestInterceptor(corba.NewLoggingServerInterceptor(2))

	// Create and configure a timing interceptor
	timingInterceptor := corba.NewTimingInterceptor()
	orb.RegisterServerRequestInterceptor(timingInterceptor)

	// Create and configure a security interceptor
	securityInterceptor := corba.NewSecurityInterceptor("secret-token-123")
	securityInterceptor.RequireRole("secureOperation", "admin")
	orb.RegisterServerRequestInterceptor(securityInterceptor)

	// Register a transaction interceptor
	orb.RegisterServerRequestInterceptor(corba.NewTransactionInterceptor())

	// Register a calculator servant
	calculator := &InterceptorCalculator{}
	err = server.RegisterServant("Calculator", calculator)
	if err != nil {
		fmt.Printf("Failed to register calculator: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Calculator servant registered with interceptors")

	// Start the server
	fmt.Println("Starting server on localhost:8099...")
	err = server.Run()
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}

	// Keep the server running until a key is pressed
	fmt.Println("Press Enter to stop the server...")
	fmt.Scanln()

	// Shutdown the server
	err = server.Shutdown()
	if err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
		os.Exit(1)
	}

	// Shutdown the ORB
	orb.Shutdown(true)
	fmt.Println("Server shutdown complete")
}

// RunInterceptorClient runs a client with various interceptors configured
func RunInterceptorClient() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a client
	client := orb.CreateClient()

	// Register client interceptors
	orb.RegisterClientRequestInterceptor(&CustomClientInterceptor{})
	orb.RegisterClientRequestInterceptor(corba.NewLoggingClientInterceptor(2))

	// Create and configure a timing interceptor
	timingInterceptor := corba.NewTimingInterceptor()
	orb.RegisterClientRequestInterceptor(timingInterceptor)

	// Register a security interceptor that adds authentication tokens
	orb.RegisterClientRequestInterceptor(corba.NewClientSecurityInterceptor("secret-token-123"))

	// Register a parameter validation interceptor
	validationInterceptor := corba.NewParameterValidationInterceptor()
	validationInterceptor.AddValidator("divide", func(args []interface{}) error {
		if len(args) != 2 {
			return fmt.Errorf("divide requires exactly 2 arguments")
		}
		if b, ok := args[1].(float64); ok && b == 0 {
			return fmt.Errorf("cannot divide by zero")
		}
		return nil
	})
	orb.RegisterClientRequestInterceptor(validationInterceptor)

	// Connect to the server
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
	fmt.Println()

	// Invoke methods with interceptors active
	fmt.Println("Testing add operation with interceptors:")
	result, err := calcRef.Invoke("add", 10.0, 20.0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v (note: interceptors modified the input and output)\n", result)
	}
	fmt.Println()

	fmt.Println("Testing slow operation with timing interceptor:")
	result, err = calcRef.Invoke("slowOperation")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
	fmt.Println()

	fmt.Println("Testing secure operation with security interceptor:")
	result, err = calcRef.Invoke("secureOperation")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
	fmt.Println()

	fmt.Println("Testing divide operation with validation interceptor:")
	result, err = calcRef.Invoke("divide", 30.0, 5.0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
	fmt.Println()

	fmt.Println("Testing divide by zero (should be caught by validation interceptor):")
	result, err = calcRef.Invoke("divide", 10.0, 0.0)
	if err != nil {
		fmt.Printf("Error (expected): %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}
	fmt.Println()

	// Disconnect the client
	err = client.Disconnect("localhost", 8099)
	if err != nil {
		fmt.Printf("Error disconnecting: %v\n", err)
	}
}

// DemoInterceptors runs both the server and client for the interceptor example
func DemoInterceptors() {
	// Start the server in a goroutine
	go RunInterceptorServer()

	// Wait for the server to start
	time.Sleep(2 * time.Second)

	// Run the client
	RunInterceptorClient()
}
