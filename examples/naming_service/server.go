package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ifabos/go-corba/corba"
)

// SimpleCalculator is a basic calculator servant
type SimpleCalculator struct {
}

// Dispatch handles method invocations on the calculator
func (c *SimpleCalculator) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	fmt.Printf("Calculator: Method called: %s with %d arguments\n", methodName, len(args))

	switch methodName {
	case "add":
		if len(args) != 2 {
			return nil, fmt.Errorf("add requires 2 arguments")
		}

		// Try to convert arguments to float64
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("arguments must be numbers")
		}

		return a + b, nil

	case "subtract":
		if len(args) != 2 {
			return nil, fmt.Errorf("subtract requires 2 arguments")
		}

		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("arguments must be numbers")
		}

		return a - b, nil

	case "multiply":
		if len(args) != 2 {
			return nil, fmt.Errorf("multiply requires 2 arguments")
		}

		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("arguments must be numbers")
		}

		return a * b, nil

	case "divide":
		if len(args) != 2 {
			return nil, fmt.Errorf("divide requires 2 arguments")
		}

		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("arguments must be numbers")
		}

		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}

		return a / b, nil

	default:
		return nil, fmt.Errorf("method %s not supported", methodName)
	}
}

func main() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a server on localhost:8099
	server, err := orb.CreateServer("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		return
	}

	// Register a calculator servant
	calculator := &SimpleCalculator{}
	err = server.RegisterServant("Calculator", calculator)
	if err != nil {
		fmt.Printf("Failed to register calculator: %v\n", err)
		return
	}
	fmt.Println("Calculator servant registered")

	// Activate the Naming Service
	err = orb.ActivateNamingService(server)
	if err != nil {
		fmt.Printf("Failed to activate naming service: %v\n", err)
		return
	}
	fmt.Println("Naming Service activated")

	// Get the naming service instance
	namingService, err := orb.GetNamingService()
	if err != nil {
		fmt.Printf("Failed to get naming service: %v\n", err)
		return
	}

	// Get the root naming context
	rootContext := namingService.GetRootContext()

	// Register the calculator in the naming service
	// Create a name for the calculator: "calculators/basic"
	calcName := corba.Name{
		corba.NameComponent{ID: "calculators", Kind: ""},
		corba.NameComponent{ID: "basic", Kind: "Calculator"},
	}

	// Bind the calculator to the name
	err = rootContext.Rebind(calcName, calculator)
	if err != nil {
		fmt.Printf("Failed to bind calculator in naming service: %v\n", err)
		return
	}
	fmt.Println("Calculator bound in naming service as 'calculators/basic'")

	// Start the server
	err = server.Run()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	fmt.Println("Server running at localhost:8099")

	// Keep the server running until interrupted
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan

	fmt.Println("\nShutting down server...")

	// Shutdown the server
	err = server.Shutdown()
	if err != nil {
		fmt.Printf("Error during shutdown: %v\n", err)
	}

	// Shutdown the ORB
	orb.Shutdown(true)

	// Give some time for connections to close
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Server shutdown complete")
}
