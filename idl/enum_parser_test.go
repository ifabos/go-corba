// Test file for enum parser
package idl_test

import (
	"bytes"
	"testing"

	"github.com/ifabos/go-corba/idl"
)

func TestEnumParsing(t *testing.T) {
	// Create a simple IDL with an enum
	idlContent := `
module Test {
    enum Colors {
        RED,
        GREEN,
        BLUE,
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

	// Get the parsed module
	rootModule := parser.GetRootModule()
	testModule, exists := rootModule.GetSubmodule("Test")
	if !exists {
		t.Fatal("Test module not found")
	}

	// Check for the Colors enum
	colorType, exists := testModule.Types["Colors"]
	if !exists {
		t.Fatal("Colors enum not found")
	}

	// Verify it's an enum
	enumType, ok := colorType.(*idl.EnumType)
	if !ok {
		t.Fatal("Colors is not an enum type")
	}

	// Verify the elements
	expectedElements := []string{"RED", "GREEN", "BLUE"}
	if len(enumType.Elements) != len(expectedElements) {
		t.Fatalf("Expected %d elements, got %d", len(expectedElements), len(enumType.Elements))
	}
	
	for i, expected := range expectedElements {
		if enumType.Elements[i] != expected {
			t.Errorf("Expected element %d to be %s, got %s", i, expected, enumType.Elements[i])
		}
	}
}
