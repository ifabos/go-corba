package examples

import (
	"fmt"
	"os"
	"reflect"

	"github.com/ifabos/go-corba/corba"
)

// DynamicCalculator is a calculator implementation that uses the Dynamic Skeleton Interface
type DynamicCalculator struct {
	*corba.DynamicServant
	values map[string]float64 // Storage for variables
}

// NewDynamicCalculator creates a new dynamic calculator
func NewDynamicCalculator() *DynamicCalculator {
	calc := &DynamicCalculator{
		DynamicServant: corba.NewDynamicServant("IDL:Calculator:1.0"),
		values:         make(map[string]float64),
	}

	// Define operations dynamically
	calc.AddOperation("add", reflect.TypeOf(float64(0)))
	calc.AddOperation("subtract", reflect.TypeOf(float64(0)))
	calc.AddOperation("multiply", reflect.TypeOf(float64(0)))
	calc.AddOperation("divide", reflect.TypeOf(float64(0)))
	calc.AddOperation("store", reflect.TypeOf(nil))
	calc.AddOperation("retrieve", reflect.TypeOf(float64(0)))

	// Add parameters to operations
	calc.AddParameter("add", "a", reflect.TypeOf(float64(0)), corba.FlagIn)
	calc.AddParameter("add", "b", reflect.TypeOf(float64(0)), corba.FlagIn)

	calc.AddParameter("subtract", "a", reflect.TypeOf(float64(0)), corba.FlagIn)
	calc.AddParameter("subtract", "b", reflect.TypeOf(float64(0)), corba.FlagIn)

	calc.AddParameter("multiply", "a", reflect.TypeOf(float64(0)), corba.FlagIn)
	calc.AddParameter("multiply", "b", reflect.TypeOf(float64(0)), corba.FlagIn)

	calc.AddParameter("divide", "a", reflect.TypeOf(float64(0)), corba.FlagIn)
	calc.AddParameter("divide", "b", reflect.TypeOf(float64(0)), corba.FlagIn)

	calc.AddParameter("store", "name", reflect.TypeOf(""), corba.FlagIn)
	calc.AddParameter("store", "value", reflect.TypeOf(float64(0)), corba.FlagIn)

	calc.AddParameter("retrieve", "name", reflect.TypeOf(""), corba.FlagIn)

	return calc
}

// Invoke implements the DynamicImplementation interface
// It handles all incoming requests dynamically based on the operation name
func (calc *DynamicCalculator) Invoke(request *corba.ServerRequest) error {
	// Validate that the operation exists and has the right number of arguments
	err := calc.ValidateOperation(request.Operation, request.Arguments)
	if err != nil {
		request.SetException(err)
		return err
	}

	// Process the operation based on its name
	switch request.Operation {
	case "add":
		// Convert arguments to float64
		if len(request.Arguments) != 2 {
			err := fmt.Errorf("add requires 2 arguments")
			request.SetException(err)
			return err
		}

		a, ok1 := request.Arguments[0].(float64)
		b, ok2 := request.Arguments[1].(float64)

		if !ok1 || !ok2 {
			err := fmt.Errorf("arguments must be numbers")
			request.SetException(err)
			return err
		}

		// Set the result
		result := a + b
		request.SetResult(result)
		return nil

	case "subtract":
		if len(request.Arguments) != 2 {
			err := fmt.Errorf("subtract requires 2 arguments")
			request.SetException(err)
			return err
		}

		a, ok1 := request.Arguments[0].(float64)
		b, ok2 := request.Arguments[1].(float64)

		if !ok1 || !ok2 {
			err := fmt.Errorf("arguments must be numbers")
			request.SetException(err)
			return err
		}

		result := a - b
		request.SetResult(result)
		return nil

	case "multiply":
		if len(request.Arguments) != 2 {
			err := fmt.Errorf("multiply requires 2 arguments")
			request.SetException(err)
			return err
		}

		a, ok1 := request.Arguments[0].(float64)
		b, ok2 := request.Arguments[1].(float64)

		if !ok1 || !ok2 {
			err := fmt.Errorf("arguments must be numbers")
			request.SetException(err)
			return err
		}

		result := a * b
		request.SetResult(result)
		return nil

	case "divide":
		if len(request.Arguments) != 2 {
			err := fmt.Errorf("divide requires 2 arguments")
			request.SetException(err)
			return err
		}

		a, ok1 := request.Arguments[0].(float64)
		b, ok2 := request.Arguments[1].(float64)

		if !ok1 || !ok2 {
			err := fmt.Errorf("arguments must be numbers")
			request.SetException(err)
			return err
		}

		if b == 0 {
			err := fmt.Errorf("division by zero")
			request.SetException(err)
			return err
		}

		result := a / b
		request.SetResult(result)
		return nil

	case "store":
		if len(request.Arguments) != 2 {
			err := fmt.Errorf("store requires 2 arguments")
			request.SetException(err)
			return err
		}

		name, ok1 := request.Arguments[0].(string)
		value, ok2 := request.Arguments[1].(float64)

		if !ok1 || !ok2 {
			err := fmt.Errorf("invalid argument types")
			request.SetException(err)
			return err
		}

		calc.values[name] = value
		request.SetResult(nil)
		return nil

	case "retrieve":
		if len(request.Arguments) != 1 {
			err := fmt.Errorf("retrieve requires 1 argument")
			request.SetException(err)
			return err
		}

		name, ok := request.Arguments[0].(string)
		if !ok {
			err := fmt.Errorf("name must be a string")
			request.SetException(err)
			return err
		}

		value, exists := calc.values[name]
		if !exists {
			err := fmt.Errorf("value %s not found", name)
			request.SetException(err)
			return err
		}

		request.SetResult(value)
		return nil

	default:
		err := fmt.Errorf("operation %s not supported", request.Operation)
		request.SetException(err)
		return err
	}
}

// RunDSIServer runs a server with the Dynamic Skeleton Interface
func RunDSIServer() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a server on localhost:8099
	server, err := orb.CreateServer("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		return
	}

	// Create a dynamic calculator
	calculator := NewDynamicCalculator()

	// Register the dynamic servant
	err = server.RegisterDynamicServant("DynamicCalculator", calculator)
	if err != nil {
		fmt.Printf("Failed to register dynamic calculator: %v\n", err)
		return
	}
	fmt.Println("Dynamic calculator servant registered")

	// Start the server
	err = server.Run()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	fmt.Println("Server running at localhost:8099")

	// Keep the server running until you press a key
	fmt.Println("Press any key to stop the server...")
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

// TestDSIClient runs a client that connects to the DSI server
func TestDSIClient() {
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
	calcRef, err := client.GetObject("DynamicCalculator", "localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to get object reference: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Got reference to DynamicCalculator object")

	// Invoke methods on the calculator
	fmt.Println("\nTesting Dynamic Calculator:")

	// Addition
	result, err := calcRef.Invoke("add", 30.5, 12.5)
	if err != nil {
		fmt.Printf("Add operation failed: %v\n", err)
	} else {
		fmt.Printf("30.5 + 12.5 = %v\n", result)
	}

	// Multiplication
	result, err = calcRef.Invoke("multiply", 5.0, 7.0)
	if err != nil {
		fmt.Printf("Multiply operation failed: %v\n", err)
	} else {
		fmt.Printf("5.0 * 7.0 = %v\n", result)
	}

	// Store a value
	_, err = calcRef.Invoke("store", "pi", 3.14159)
	if err != nil {
		fmt.Printf("Store operation failed: %v\n", err)
	} else {
		fmt.Println("Stored pi = 3.14159")
	}

	// Retrieve the stored value
	result, err = calcRef.Invoke("retrieve", "pi")
	if err != nil {
		fmt.Printf("Retrieve operation failed: %v\n", err)
	} else {
		fmt.Printf("Retrieved pi = %v\n", result)
	}

	// Test error handling with division by zero
	result, err = calcRef.Invoke("divide", 10.0, 0.0)
	if err != nil {
		fmt.Printf("Division by zero properly rejected: %v\n", err)
	} else {
		fmt.Printf("10.0 / 0.0 = %v (This should not happen!)\n", result)
	}

	// Disconnect from the server
	err = client.Disconnect("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to disconnect: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nDisconnected from server")
}
