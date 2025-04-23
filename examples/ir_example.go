package examples

import (
	"fmt"
	"os"
	"time"

	"github.com/ifabos/go-corba/corba"
)

// Example interface to register
type Calculator interface {
	Add(a, b float64) float64
	Subtract(a, b float64) float64
	Multiply(a, b float64) float64
	Divide(a, b float64) (float64, error)
}

// Implementation of the Calculator interface
type SimpleCalculator struct{}

func (c *SimpleCalculator) Add(a, b float64) float64 {
	return a + b
}

func (c *SimpleCalculator) Subtract(a, b float64) float64 {
	return a - b
}

func (c *SimpleCalculator) Multiply(a, b float64) float64 {
	return a * b
}

func (c *SimpleCalculator) Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// Helper for dispatching to the calculator implementation
func (c *SimpleCalculator) Dispatch(methodName string, args []interface{}) (interface{}, error) {
	switch methodName {
	case "add":
		if len(args) != 2 {
			return nil, fmt.Errorf("add requires exactly 2 arguments")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("add arguments must be float64")
		}
		return c.Add(a, b), nil

	case "subtract":
		if len(args) != 2 {
			return nil, fmt.Errorf("subtract requires exactly 2 arguments")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("subtract arguments must be float64")
		}
		return c.Subtract(a, b), nil

	case "multiply":
		if len(args) != 2 {
			return nil, fmt.Errorf("multiply requires exactly 2 arguments")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("multiply arguments must be float64")
		}
		return c.Multiply(a, b), nil

	case "divide":
		if len(args) != 2 {
			return nil, fmt.Errorf("divide requires exactly 2 arguments")
		}
		a, ok1 := args[0].(float64)
		b, ok2 := args[1].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("divide arguments must be float64")
		}
		return c.Divide(a, b)

	default:
		return nil, fmt.Errorf("unknown method: %s", methodName)
	}
}

// RunServer starts a server with the Interface Repository
func RunIRServer() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a server on localhost:8099
	server, err := orb.CreateServer("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Register a calculator servant
	calculator := &SimpleCalculator{}
	err = server.RegisterServant("Calculator", calculator)
	if err != nil {
		fmt.Printf("Failed to register calculator: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Calculator servant registered")

	// Activate the Interface Repository
	err = orb.ActivateInterfaceRepository(server)
	if err != nil {
		fmt.Printf("Failed to activate Interface Repository: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Interface Repository activated")

	// Get the Interface Repository
	ir, err := orb.GetInterfaceRepository()
	if err != nil {
		fmt.Printf("Failed to get Interface Repository: %v\n", err)
		os.Exit(1)
	}

	// Register the Calculator interface
	calculatorID := "IDL:examples/Calculator:1.0"
	err = orb.RegisterInterface(calculator, calculatorID, "Calculator")
	if err != nil {
		fmt.Printf("Failed to register Calculator interface: %v\n", err)
		os.Exit(1)
	}

	// Create the Calculator interface definition in the IR
	repo := ir.GetRepository()
	calcIface, err := repo.LookupId(calculatorID)
	if err != nil {
		// Interface doesn't exist, create it
		calcIface, err = repo.CreateInterface(calculatorID, "Calculator")
		if err != nil {
			fmt.Printf("Failed to create Calculator interface: %v\n", err)
			os.Exit(1)
		}
	}

	// Add operations to the Calculator interface
	if iface, ok := calcIface.(corba.InterfaceDef); ok {
		// Look up or create float64 type
		floatTC, err := repo.LookupId("IDL:omg.org/CORBA/Double:1.0")
		if err != nil {
			fmt.Printf("Failed to lookup Double type: %v\n", err)
			os.Exit(1)
		}
		floatTypeCode := floatTC.(corba.TypeCode)

		// Add operation definitions
		addOp, err := iface.CreateOperation("IDL:examples/Calculator/add:1.0", "add", floatTypeCode, 0)
		if err != nil {
			fmt.Printf("Failed to create add operation: %v\n", err)
			os.Exit(1)
		}
		addOp.AddParameter("a", floatTypeCode, corba.PARAM_IN)
		addOp.AddParameter("b", floatTypeCode, corba.PARAM_IN)

		subOp, err := iface.CreateOperation("IDL:examples/Calculator/subtract:1.0", "subtract", floatTypeCode, 0)
		if err != nil {
			fmt.Printf("Failed to create subtract operation: %v\n", err)
			os.Exit(1)
		}
		subOp.AddParameter("a", floatTypeCode, corba.PARAM_IN)
		subOp.AddParameter("b", floatTypeCode, corba.PARAM_IN)

		mulOp, err := iface.CreateOperation("IDL:examples/Calculator/multiply:1.0", "multiply", floatTypeCode, 0)
		if err != nil {
			fmt.Printf("Failed to create multiply operation: %v\n", err)
			os.Exit(1)
		}
		mulOp.AddParameter("a", floatTypeCode, corba.PARAM_IN)
		mulOp.AddParameter("b", floatTypeCode, corba.PARAM_IN)

		divOp, err := iface.CreateOperation("IDL:examples/Calculator/divide:1.0", "divide", floatTypeCode, 0)
		if err != nil {
			fmt.Printf("Failed to create divide operation: %v\n", err)
			os.Exit(1)
		}
		divOp.AddParameter("a", floatTypeCode, corba.PARAM_IN)
		divOp.AddParameter("b", floatTypeCode, corba.PARAM_IN)

		// Create an exception for divide by zero
		divExcept, err := repo.CreateException("IDL:examples/DivByZero:1.0", "DivByZero")
		if err != nil {
			fmt.Printf("Failed to create divide by zero exception: %v\n", err)
			os.Exit(1)
		}
		divOp.AddException(divExcept)

		fmt.Println("Calculator interface defined in the Interface Repository")
	}

	// Start the server
	fmt.Println("Starting server on localhost:8099...")
	err = server.Run()
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}

// RunClient connects to the server and uses the Interface Repository to learn about services
func RunIRClient() {
	// Initialize the ORB
	orb := corba.Init()

	// Connect to the Interface Repository
	irClient, err := orb.ResolveInterfaceRepository("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to connect to Interface Repository: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected to Interface Repository")

	// Look up the Calculator interface
	calculatorID := "IDL:examples/Calculator:1.0"
	calcDesc, err := irClient.DescribeInterface(calculatorID)
	if err != nil {
		fmt.Printf("Failed to lookup Calculator interface: %v\n", err)
		os.Exit(1)
	}

	// Display the interface information
	fmt.Println("\nInterface Information:")
	fmt.Printf("Name: %s\n", calcDesc["name"])
	fmt.Printf("ID: %s\n", calcDesc["id"])

	// Display operations
	fmt.Println("\nOperations:")
	if ops, ok := calcDesc["operations"].([]map[string]interface{}); ok {
		for _, op := range ops {
			fmt.Printf("  - %s\n", op["name"])
			if params, ok := op["parameters"].([]map[string]interface{}); ok {
				fmt.Println("    Parameters:")
				for _, param := range params {
					mode := "IN"
					if m, ok := param["mode"].(int); ok {
						if m == int(corba.PARAM_OUT) {
							mode = "OUT"
						} else if m == int(corba.PARAM_INOUT) {
							mode = "INOUT"
						}
					}
					fmt.Printf("      %s: %s (%s)\n", param["name"], param["type_id"], mode)
				}
			}
			if rt, ok := op["result_type"].(string); ok {
				fmt.Printf("    Returns: %s\n", rt)
			}
		}
	}

	// Now that we know the interface, use it to call operations
	fmt.Println("\nUsing the calculator...")
	client := orb.CreateClient()
	err = client.Connect("localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}

	calcRef, err := client.GetObject("Calculator", "localhost", 8099)
	if err != nil {
		fmt.Printf("Failed to get Calculator reference: %v\n", err)
		os.Exit(1)
	}

	// Call add
	result, err := calcRef.Invoke("add", 10.5, 20.7)
	if err != nil {
		fmt.Printf("Error calling add: %v\n", err)
	} else {
		fmt.Printf("10.5 + 20.7 = %v\n", result)
	}

	// Call divide
	result, err = calcRef.Invoke("divide", 30.0, 5.0)
	if err != nil {
		fmt.Printf("Error calling divide: %v\n", err)
	} else {
		fmt.Printf("30.0 / 5.0 = %v\n", result)
	}

	// Try divide by zero (should result in an error)
	result, err = calcRef.Invoke("divide", 10.0, 0.0)
	if err != nil {
		fmt.Printf("Error calling divide by zero (expected): %v\n", err)
	} else {
		fmt.Printf("10.0 / 0.0 = %v\n", result)
	}
}

// DemoInterfaceRepository runs both the server and client for the IR example
func DemoInterfaceRepository() {
	go RunIRServer()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	RunIRClient()
}
