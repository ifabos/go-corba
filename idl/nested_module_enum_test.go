package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

// TestNestedModuleEnum tests parsing an enum in a nested module
func TestNestedModuleEnum(t *testing.T) {
	// Create a simple IDL with an enum in a nested module
	idlContent := `
module Outer {
    module Inner {
        enum DeepEnum {
            ALPHA,
            BETA,
            GAMMA,
        };
    };
};
`

	// Create a parser
	parser := idl.NewParser()

	// Parse the IDL from a byte buffer
	err := parser.Parse(bytes.NewBufferString(idlContent))
	if err != nil {
		t.Fatalf("Error parsing IDL: %v", err)
	}

	// Get the outer module
	rootModule := parser.GetRootModule()
	outerModule, exists := rootModule.GetSubmodule("Outer")
	if !exists {
		t.Fatal("Outer module not found")
	}

	// Get the inner module
	innerModule, exists := outerModule.GetSubmodule("Inner")
	if !exists {
		t.Fatal("Inner module not found")
	}

	// Check for the DeepEnum
	enumType, exists := innerModule.Types["DeepEnum"]
	if !exists {
		t.Fatal("DeepEnum not found")
	}

	// Verify it's an enum
	enum, ok := enumType.(*idl.EnumType)
	if !ok {
		t.Fatal("DeepEnum is not an enum type")
	}

	// Expected elements
	expectedElements := []string{"ALPHA", "BETA", "GAMMA"}

	// Verify the number of elements
	if len(enum.Elements) != len(expectedElements) {
		t.Fatalf("Expected %d elements in DeepEnum, got %d",
			len(expectedElements), len(enum.Elements))
	}

	// Verify each element
	for i, expected := range expectedElements {
		if enum.Elements[i] != expected {
			t.Errorf("Expected element %d to be %s, got %s", i, expected, enum.Elements[i])
		}
	}
}
