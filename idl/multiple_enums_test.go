package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

// TestMultipleEnumsInModule tests parsing multiple enum declarations within a single module
func TestMultipleEnumsInModule(t *testing.T) {
	// Create a simple IDL with multiple enums in one module
	idlContent := `
module MultipleEnums {
    enum First {
        A,
        B,
        C,
    };
    
    enum Second {
        X,
        Y,
        Z
    };
    
    enum Third { ONE, TWO, THREE };
};
`

	// Create a parser
	parser := idl.NewParser()

	// Parse the IDL from a byte buffer
	err := parser.Parse(bytes.NewBufferString(idlContent))
	if err != nil {
		t.Fatalf("Error parsing IDL: %v", err)
	}

	// Get the parsed module
	rootModule := parser.GetRootModule()
	testModule, exists := rootModule.GetSubmodule("MultipleEnums")
	if !exists {
		t.Fatal("MultipleEnums module not found")
	}

	// Define expected enums
	enums := map[string][]string{
		"First":  {"A", "B", "C"},
		"Second": {"X", "Y", "Z"},
		"Third":  {"ONE", "TWO", "THREE"},
	}

	// Check each enum
	for enumName, expectedElements := range enums {
		// Check if enum exists
		enumType, exists := testModule.Types[enumName]
		if !exists {
			t.Fatalf("Enum %s not found", enumName)
		}

		// Verify it's an enum
		enum, ok := enumType.(*idl.EnumType)
		if !ok {
			t.Fatalf("%s is not an enum type", enumName)
		}

		// Verify the number of elements
		if len(enum.Elements) != len(expectedElements) {
			t.Fatalf("Expected %d elements in enum %s, got %d", 
				len(expectedElements), enumName, len(enum.Elements))
		}

		// Verify each element
		for i, expected := range expectedElements {
			if enum.Elements[i] != expected {
				t.Errorf("Expected element %d in %s to be %s, got %s", 
					i, enumName, expected, enum.Elements[i])
			}
		}
	}
}
