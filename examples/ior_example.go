package examples

import (
	"fmt"
	"log"

	"github.com/ifabos/go-corba/corba"
)

// IORExample demonstrates the usage of IORs in the Go-CORBA SDK
func IORExample() {
	// Initialize the ORB
	orb := corba.Init()

	// Create a server
	server, _ := orb.CreateServer("localhost", 8765)

	fmt.Println("=== IOR Creation and Conversion Example ===")

	// Example 1: Creating an IOR programmatically
	ior := corba.NewIOR(corba.FormatRepositoryID("Calculator", "1.0"))
	ior.AddIIOPProfile(
		corba.IIOPVersion{Major: 1, Minor: 2},
		"localhost",
		8765,
		corba.ObjectKeyFromString("CalculatorService"),
	)

	// Convert IOR to string form
	iorString := ior.ToString()
	fmt.Println("Created IOR string:", iorString)

	// Example 2: Parse an IOR string
	parsedIOR, err := corba.ParseIOR(iorString)
	if err != nil {
		log.Fatalf("Error parsing IOR: %v", err)
	}

	fmt.Println("Parsed IOR TypeID:", parsedIOR.TypeID)

	// Extract profile information
	profile, err := parsedIOR.GetPrimaryIIOPProfile()
	if err != nil {
		log.Fatalf("Error getting profile: %v", err)
	}

	fmt.Printf("Profile Info: IIOP %v, Host: %s, Port: %d\n",
		profile.Version, profile.Host, profile.Port)
	fmt.Printf("Object Key: %s\n\n", corba.ObjectKeyToString(profile.ObjectKey))

	// Example 3: Creating Object References with IOR
	fmt.Println("=== Object Reference with IOR Example ===")

	// Register a sample calculator service
	calculator := &CalculatorServant{}
	err = server.RegisterServant("CalculatorService", calculator)
	if err != nil {
		log.Fatalf("Failed to register calculator: %v", err)
	}

	// Create an ObjectRef
	calcRef := &corba.ObjectRef{
		Name:       "CalculatorService",
		ServerHost: "localhost",
		ServerPort: 8765,
	}

	// Set the Type ID (repository ID)
	calcRef.SetTypeID(corba.FormatRepositoryID("Calculator", "1.0"))

	// Convert to IOR string
	objIORString, err := orb.ObjectToString(calcRef)
	if err != nil {
		log.Fatalf("Error converting object to IOR string: %v", err)
	}

	fmt.Println("Object IOR string:", objIORString)

	// Convert string back to object reference
	resolvedRef, err := orb.StringToObject(objIORString)
	if err != nil {
		log.Fatalf("Error converting string to object: %v", err)
	}

	fmt.Printf("Resolved reference - Name: %s, Host: %s, Port: %d\n",
		resolvedRef.Name, resolvedRef.ServerHost, resolvedRef.ServerPort)
	fmt.Printf("Type ID: %s\n\n", resolvedRef.GetTypeID())

	// Example 4: Using POA to create references
	fmt.Println("=== POA and IOR Integration Example ===")

	// Get the root POA
	rootPOA := orb.GetRootPOA()

	// Create a reference with POA
	poaRef := rootPOA.CreateReference("Calculator", corba.ObjectKeyFromString("CalcInstance"))

	// Convert to string
	poaIORString, err := orb.ObjectToString(poaRef)
	if err != nil {
		log.Fatalf("Error converting POA reference to string: %v", err)
	}

	fmt.Println("POA-created IOR string:", poaIORString)

	// Activate an object with the POA
	servant := &CalculatorServant{}
	objectID, err := rootPOA.ActivateObject(servant)
	if err != nil {
		log.Fatalf("Error activating object: %v", err)
	}

	// Get a reference to the activated object
	activeRef, err := rootPOA.ServantToReference(servant)
	if err != nil {
		log.Fatalf("Error getting reference: %v", err)
	}

	// Convert to string
	activeIORString, err := orb.ObjectToString(activeRef)
	if err != nil {
		log.Fatalf("Error converting reference to string: %v", err)
	}

	fmt.Printf("Activated object ID: %s\n", corba.ObjectKeyToString(objectID))
	fmt.Printf("Activated IOR string: %s\n", activeIORString)

	// Clean up
	rootPOA.DeactivateObject(objectID)
	fmt.Println("\nIOR Example completed successfully")
}

// CalculatorServant is a simple calculator service implementation
type CalculatorServant struct{}

// Add adds two numbers
func (c *CalculatorServant) Add(a, b float64) float64 {
	return a + b
}

// Subtract subtracts b from a
func (c *CalculatorServant) Subtract(a, b float64) float64 {
	return a - b
}

// Multiply multiplies a and b
func (c *CalculatorServant) Multiply(a, b float64) float64 {
	return a * b
}

// Divide divides a by b
func (c *CalculatorServant) Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// Dispatch implements the dispatcher interface for CORBA invocations
func (c *CalculatorServant) Dispatch(method string, args []interface{}) (interface{}, error) {
	switch method {
	case "Add":
		if len(args) != 2 {
			return nil, fmt.Errorf("Add requires 2 arguments")
		}
		a, aOk := args[0].(float64)
		b, bOk := args[1].(float64)
		if !aOk || !bOk {
			return nil, fmt.Errorf("Add requires float64 arguments")
		}
		return c.Add(a, b), nil
	case "Subtract":
		if len(args) != 2 {
			return nil, fmt.Errorf("Subtract requires 2 arguments")
		}
		a, aOk := args[0].(float64)
		b, bOk := args[1].(float64)
		if !aOk || !bOk {
			return nil, fmt.Errorf("Subtract requires float64 arguments")
		}
		return c.Subtract(a, b), nil
	case "Multiply":
		if len(args) != 2 {
			return nil, fmt.Errorf("Multiply requires 2 arguments")
		}
		a, aOk := args[0].(float64)
		b, bOk := args[1].(float64)
		if !aOk || !bOk {
			return nil, fmt.Errorf("Multiply requires float64 arguments")
		}
		return c.Multiply(a, b), nil
	case "Divide":
		if len(args) != 2 {
			return nil, fmt.Errorf("Divide requires 2 arguments")
		}
		a, aOk := args[0].(float64)
		b, bOk := args[1].(float64)
		if !aOk || !bOk {
			return nil, fmt.Errorf("Divide requires float64 arguments")
		}
		return c.Divide(a, b)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}
