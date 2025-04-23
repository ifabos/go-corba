package examples

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	gocorba "github.com/ifabos/go-corba"
	"github.com/ifabos/go-corba/corba"
	"github.com/ifabos/go-corba/idl"
)

// CalculatorIDLExample demonstrates how to use IDL with CORBA in Go
func CalculatorIDLExample() {
	fmt.Println("=== CORBA IDL Example ===")

	// Step 1: Generate Go code from IDL
	// In a real application, you'd typically do this step ahead of time with the idlgen tool:
	// $ idlgen -i calculator.idl -o ./generated -package calcservice
	fmt.Println("Step 1: Generate Go code from IDL (simulating)")

	// Get the IDL file path
	idlFile := filepath.Join("examples", "idl", "calculator.idl")
	idlData, err := os.ReadFile(idlFile)
	if err != nil {
		fmt.Printf("Error reading IDL file: %v\n", err)
		return
	}

	// Create a temporary directory for the generated code
	genDir, err := ioutil.TempDir("", "idlgen-*")
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(genDir) // Cleanup on exit

	// Parse the IDL file
	parser := idl.NewParser()
	err = parser.Parse(bytes.NewReader(idlData))
	if err != nil {
		fmt.Printf("Error parsing IDL: %v\n", err)
		return
	}

	// Generate Go code
	generator := idl.NewGenerator(parser.GetRootModule(), genDir)
	generator.SetPackageName("calcservice")
	generator.AddInclude("github.com/ifabos/go-corba/corba")

	err = generator.Generate()
	if err != nil {
		fmt.Printf("Error generating code: %v\n", err)
		return
	}

	fmt.Printf("Generated Go code in %s\n", genDir)

	// Step 2: Using the generated code (simulating, since we can't import the generated code easily here)
	// In a real application, you'd import the generated package and use its types directly
	fmt.Println("\nStep 2: Using the generated code")

	// Initialize the ORB
	orb := gocorba.Init()

	// Create a calculator server implementation
	calcServer := &CalculatorImpl{
		memory:  0.0,
		opCount: 0,
	}

	// Start a CORBA server
	server, err := orb.CreateServer("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		return
	}

	// Register the calculator servant
	calcServant := &CalculatorServant{
		Impl: calcServer,
	}

	// Register with server
	err = server.RegisterServant("Calculator", calcServant)
	if err != nil {
		fmt.Printf("Failed to register servant: %v\n", err)
		return
	}

	// Start the server in a goroutine
	go func() {
		fmt.Println("Starting CORBA server...")
		err := server.Run()
		if err != nil {
			fmt.Printf("Server stopped with error: %v\n", err)
		}
	}()

	// Create a CORBA client
	client := orb.CreateClient()

	// Connect to the server
	if err := client.Connect("localhost", 8099); err != nil {
		fmt.Printf("Failed to connect to server: %v\n", err)
		return
	}

	// Get a reference to the calculator object
	calcRef, err := client.GetObject("Calculator", "localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to get calculator reference: %v\n", err)
		return
	}

	// Create a stub for the calculator
	// In a real application with generated code, you would use:
	// calcHelper := &calcservice.CalcHelper{}
	// calc, err := calcHelper.Narrow(calcRef)
	calcStub := &CalculatorStub{
		ObjectRef: calcRef,
	}

	// Use the calculator
	fmt.Println("\nUsing the calculator:")

	// Basic operations
	sum, err := calcStub.Add(10.5, 20.2)
	fmt.Printf("10.5 + 20.2 = %.2f\n", sum)

	diff, err := calcStub.Subtract(30.0, 15.5)
	fmt.Printf("30.0 - 15.5 = %.2f\n", diff)

	prod, err := calcStub.Multiply(4.5, 2.5)
	fmt.Printf("4.5 * 2.5 = %.2f\n", prod)

	quot, err := calcStub.Divide(10.0, 2.0)
	fmt.Printf("10.0 / 2.0 = %.2f\n", quot)

	// Try division by zero
	_, err = calcStub.Divide(5.0, 0.0)
	if err != nil {
		fmt.Printf("Division by zero error: %v\n", err)
	}

	// Memory operations
	calcStub.StoreValue(42.0)
	val, err := calcStub.RecallValue()
	fmt.Printf("Stored and recalled: %.2f\n", val)

	// Get operation count
	count, err := calcStub.GetOperationCount()
	fmt.Printf("Operation count: %d\n", count)

	// Clean up
	calcStub.ClearMemory()
	server.Shutdown()
	client.Disconnect("localhost", 8099)
	orb.Shutdown(true)

	fmt.Println("CORBA IDL example completed successfully")
}

// --- Server implementation ---

// CalculatorImpl implements the Calc interface from the IDL
type CalculatorImpl struct {
	memory  float64
	opCount int32
}

func (c *CalculatorImpl) Add(a float64, b float64) (float64, error) {
	c.opCount++
	return a + b, nil
}

func (c *CalculatorImpl) Subtract(a float64, b float64) (float64, error) {
	c.opCount++
	return a - b, nil
}

func (c *CalculatorImpl) Multiply(a float64, b float64) (float64, error) {
	c.opCount++
	return a * b, nil
}

func (c *CalculatorImpl) Divide(a float64, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("DivByZeroException: division by zero")
	}
	c.opCount++
	return a / b, nil
}

func (c *CalculatorImpl) StoreValue(value float64) error {
	c.memory = value
	c.opCount++
	return nil
}

func (c *CalculatorImpl) RecallValue() (float64, error) {
	c.opCount++
	return c.memory, nil
}

func (c *CalculatorImpl) ClearMemory() error {
	c.memory = 0.0
	c.opCount++
	return nil
}

func (c *CalculatorImpl) GetOperationCount() (int32, error) {
	return c.opCount, nil
}

// CalculatorServant is the CORBA servant for the calculator
type CalculatorServant struct {
	Impl *CalculatorImpl
}

// Dispatch handles method calls to the servant
func (s *CalculatorServant) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	case "add":
		if len(args) != 2 {
			return nil, fmt.Errorf("wrong number of arguments for add")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("invalid argument types for add")
		}
		return s.Impl.Add(a, b)

	case "subtract":
		if len(args) != 2 {
			return nil, fmt.Errorf("wrong number of arguments for subtract")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("invalid argument types for subtract")
		}
		return s.Impl.Subtract(a, b)

	case "multiply":
		if len(args) != 2 {
			return nil, fmt.Errorf("wrong number of arguments for multiply")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("invalid argument types for multiply")
		}
		return s.Impl.Multiply(a, b)

	case "divide":
		if len(args) != 2 {
			return nil, fmt.Errorf("wrong number of arguments for divide")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("invalid argument types for divide")
		}
		return s.Impl.Divide(a, b)

	case "storeValue":
		if len(args) != 1 {
			return nil, fmt.Errorf("wrong number of arguments for storeValue")
		}
		value, ok := args[0].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid argument type for storeValue")
		}
		return nil, s.Impl.StoreValue(value)

	case "recallValue":
		if len(args) != 0 {
			return nil, fmt.Errorf("wrong number of arguments for recallValue")
		}
		return s.Impl.RecallValue()

	case "clearMemory":
		if len(args) != 0 {
			return nil, fmt.Errorf("wrong number of arguments for clearMemory")
		}
		return nil, s.Impl.ClearMemory()

	case "_get_operationCount":
		if len(args) != 0 {
			return nil, fmt.Errorf("wrong number of arguments for _get_operationCount")
		}
		return s.Impl.GetOperationCount()

	default:
		return nil, fmt.Errorf("method %s not found", methodName)
	}
}

// --- Client stub ---

// CalculatorStub implements the client side of the Calc interface
type CalculatorStub struct {
	ObjectRef *corba.ObjectRef
}

func (s *CalculatorStub) Add(a float64, b float64) (float64, error) {
	result, err := s.ObjectRef.Invoke("add", a, b)
	if err != nil {
		return 0, err
	}

	if val, ok := result.(float64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected result type: %T", result)
}

func (s *CalculatorStub) Subtract(a float64, b float64) (float64, error) {
	result, err := s.ObjectRef.Invoke("subtract", a, b)
	if err != nil {
		return 0, err
	}

	if val, ok := result.(float64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected result type: %T", result)
}

func (s *CalculatorStub) Multiply(a float64, b float64) (float64, error) {
	result, err := s.ObjectRef.Invoke("multiply", a, b)
	if err != nil {
		return 0, err
	}

	if val, ok := result.(float64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected result type: %T", result)
}

func (s *CalculatorStub) Divide(a float64, b float64) (float64, error) {
	result, err := s.ObjectRef.Invoke("divide", a, b)
	if err != nil {
		return 0, err
	}

	if val, ok := result.(float64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected result type: %T", result)
}

func (s *CalculatorStub) StoreValue(value float64) error {
	_, err := s.ObjectRef.Invoke("storeValue", value)
	return err
}

func (s *CalculatorStub) RecallValue() (float64, error) {
	result, err := s.ObjectRef.Invoke("recallValue")
	if err != nil {
		return 0, err
	}

	if val, ok := result.(float64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected result type: %T", result)
}

func (s *CalculatorStub) ClearMemory() error {
	_, err := s.ObjectRef.Invoke("clearMemory")
	return err
}

func (s *CalculatorStub) GetOperationCount() (int32, error) {
	result, err := s.ObjectRef.Invoke("_get_operationCount")
	if err != nil {
		return 0, err
	}

	if val, ok := result.(int32); ok {
		return val, nil
	}
	return 0, fmt.Errorf("unexpected result type: %T", result)
}
