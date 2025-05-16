package main

import (
	"fmt"
	"os"

	"github.com/ifabos/go-corba/idl"
)

func main() {
	// Create a parser
	parser := idl.NewParser()

	// Open the IDL file
	file, err := os.Open("examples/idl/enum_no_trailing_comma.idl")
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Parse the IDL file
	err = parser.Parse(file)
	if err != nil {
		fmt.Printf("Error parsing IDL: %v\n", err)
		os.Exit(1)
	}

	// Get the parsed module
	rootModule := parser.GetRootModule()
	testModule, exists := rootModule.GetSubmodule("Test")
	if !exists {
		fmt.Println("Test module not found")
		os.Exit(1)
	}

	// Check for the Status enum
	statusType, exists := testModule.Types["Status"]
	if !exists {
		fmt.Println("Status enum not found")
		os.Exit(1)
	}

	// Verify it's an enum
	enumType, ok := statusType.(*idl.EnumType)
	if !ok {
		fmt.Println("Status is not an enum type")
		os.Exit(1)
	}

	// Print the elements
	fmt.Println("Enum Status has elements:")
	for i, elem := range enumType.Elements {
		fmt.Printf("%d: %s\n", i, elem)
	}

	fmt.Println("Enum parsing without trailing comma test passed successfully!")
}
